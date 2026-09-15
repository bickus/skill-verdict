// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/rules"
	"github.com/bickus/skill-verdict/pkg/scan"
)

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSkill(t, filepath.Join(root, "skills", "a"))
	writeSkill(t, filepath.Join(root, "skills", "b"))
	writeSkill(t, filepath.Join(root, "elsewhere", "c"))
	if err := os.Symlink(filepath.Join(root, "elsewhere", "c"), filepath.Join(root, "skills", "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skills", "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "notes.txt"), []byte("notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func names(results []scan.Result) []string {
	var out []string
	for _, r := range results {
		out = append(out, r.Skill.Name)
	}
	return out
}

func local() config.Config {
	cfg := config.Default()
	cfg.Layers.References.Enabled = false
	cfg.Layers.Honeypot.Enabled = false
	cfg.Layers.Discovery.Enabled = false
	cfg.Layers.Judge.Enabled = false
	return cfg
}

func options(t *testing.T, cfg config.Config) scan.Options {
	t.Helper()
	set, err := rules.Load("", cfg.Layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	return scan.Options{Config: cfg, Rules: set}
}

func TestRunResolvesTargets(t *testing.T) {
	root := tree(t)
	cases := []struct {
		name    string
		path    string
		want    []string
		wantErr bool
	}{
		{"directory of skills", "skills", []string{"a", "b", "c"}, false},
		{"single skill", "skills/a", []string{"a"}, false},
		{"symlinked skill", "skills/link", []string{"c"}, false},
		{"directory without skills", "skills/empty", nil, true},
		{"plain file", "skills/notes.txt", nil, true},
		{"missing path", "nowhere", nil, true},
	}
	opts := options(t, local())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := scan.Run(context.Background(), []string{filepath.Join(root, tc.path)}, opts)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got := names(results); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

type outcome struct {
	verdict  scan.Verdict
	layers   []scan.LayerStatus
	hashed   bool
	findings []string
}

func summary(r scan.Result) outcome {
	out := outcome{verdict: r.Verdict, hashed: r.Skill.Hash != ""}
	for _, l := range r.Layers {
		out.layers = append(out.layers, scan.LayerStatus{Layer: l.Layer, Status: l.Status, Interrupt: l.Interrupt})
	}
	for _, f := range r.Findings {
		out.findings = append(out.findings, f.Rule+"@"+f.Location.File)
	}
	return out
}

func TestRunInterrupt(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root)
	if err := os.WriteFile(filepath.Join(root, "guide.pdf"), []byte("%PDF-1.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	off, low := false, finding.Low
	stopped := []scan.LayerStatus{{Layer: "walk", Status: "interrupted", Interrupt: "FILE-PDF"}, {Layer: "reveal", Status: "skipped"}, {Layer: "static", Status: "skipped"}}
	ran := []scan.LayerStatus{{Layer: "walk", Status: "done"}, {Layer: "reveal", Status: "done"}, {Layer: "static", Status: "done"}}
	cases := []struct {
		name    string
		setting config.RuleSetting
		want    outcome
	}{
		{"shipped rule stops the scan and blocks", config.RuleSetting{}, outcome{scan.Block, stopped, false, []string{"FILE-PDF@guide.pdf"}}},
		{"severity below the gate stops the scan and reviews", config.RuleSetting{Severity: &low}, outcome{scan.Review, stopped, false, []string{"FILE-PDF@guide.pdf"}}},
		{"interrupt off runs every layer", config.RuleSetting{Interrupt: &off}, outcome{scan.Block, ran, true, []string{"FILE-PDF@guide.pdf"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := local()
			cfg.Layers.Walk.Rules = map[string]config.RuleSetting{"FILE-PDF": tc.setting}
			results, err := scan.Run(context.Background(), []string{root}, options(t, cfg))
			if err != nil {
				t.Fatal(err)
			}
			if got := summary(results[0]); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

type cancelling struct {
	cancel  context.CancelFunc
	answers int
}

func (c *cancelling) Complete(ctx context.Context, _ llm.Request) (llm.Message, error) {
	if c.answers == 0 {
		c.cancel()
		return llm.Message{}, ctx.Err()
	}
	c.answers--
	return llm.Message{Role: llm.Assistant, Text: "no findings"}, nil
}

func TestRunStopsWhenCancelled(t *testing.T) {
	skills := filepath.Join(tree(t), "skills")
	cases := []struct {
		name     string
		before   bool
		answers  int
		finished []string
		skipped  int
	}{
		{"cancelled before the first skill", true, 0, nil, 3},
		{"cancelled during a model call of the second skill", false, 1, []string{"a"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.before {
				cancel()
			}
			cfg := local()
			cfg.LLM.Model = "m"
			cfg.Layers.Discovery.Enabled = true
			cfg.Layers.Discovery.Calls = 1
			opts := options(t, cfg)
			opts.Provider = &cancelling{cancel: cancel, answers: tc.answers}
			var reported []string
			opts.Each = func(r scan.Result) error {
				reported = append(reported, r.Skill.Name)
				return nil
			}
			results, err := scan.Run(ctx, []string{skills}, opts)
			var stopped *scan.StoppedError
			if !errors.As(err, &stopped) {
				t.Fatalf("err = %v, want a stopped scan", err)
			}
			if !reflect.DeepEqual(names(results), tc.finished) || !reflect.DeepEqual(reported, tc.finished) || stopped.Skipped != tc.skipped {
				t.Errorf("results %v, reported %v, skipped %d, want %v, %v, %d", names(results), reported, stopped.Skipped, tc.finished, tc.finished, tc.skipped)
			}
		})
	}
}
