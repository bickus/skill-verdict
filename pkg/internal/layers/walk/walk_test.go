// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk_test

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func open() config.WalkConfig {
	return config.WalkConfig{MaxFileBytes: 1 << 20, MaxBundleBytes: 1 << 20, MaxTextBytes: 1 << 20, MaxFiles: 1000, MaxDepth: 32}
}

func newState(t *testing.T, dir string) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	off := false
	setting := config.RuleSetting{Interrupt: &off}
	layers := config.Default().Layers
	layers.Walk.Rules = map[string]config.RuleSetting{
		"FILE-TOO-LARGE": setting, "BUNDLE-TOO-LARGE": setting, "BUNDLE-TOO-MUCH-TEXT": setting, "BUNDLE-TOO-MANY-FILES": setting,
		"BUNDLE-TOO-DEEP": setting, "FILE-PDF": setting, "ARCHIVE-ENCRYPTED": setting, "ARCHIVE-RAR": setting,
		"FILE-BINARY": setting, "FILE-INSTALLER": setting,
	}
	set, err := rules.Load("", layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	return &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
}

func walked(t *testing.T, dir string, cfg config.WalkConfig) *pipeline.State {
	t.Helper()
	s := newState(t, dir)
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: cfg}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	return s
}

func artifact(t *testing.T, s *inventory.Skill, name string) inventory.Artifact {
	t.Helper()
	i := slices.IndexFunc(s.Artifacts, func(a inventory.Artifact) bool { return a.Path == name })
	if i < 0 {
		t.Fatalf("no artifact for %s", name)
	}
	return s.Artifacts[i]
}

func TestWalkContentTypes(t *testing.T) {
	skill := walked(t, "testdata/basic", open()).Skill
	cases := []struct {
		path string
		want inventory.ContentType
	}{
		{"SKILL.md", inventory.Markdown},
		{"docs/multibyte.md", inventory.Markdown},
		{"scripts/run.sh", inventory.Code},
		{"node_modules/evil/index.js", inventory.Code},
		{"tool", inventory.Code},
		{"Makefile", inventory.Code},
		{"LICENSE", inventory.Text},
		{"notes.txt", inventory.Text},
		{"latin1.txt", inventory.Text},
		{"deep/a/b/file.txt", inventory.Text},
		{"config.json", inventory.Data},
		{"image.pdf", inventory.Media},
		{"blob", inventory.Unknown},
		{"archive.zip", inventory.Unknown},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			a := artifact(t, skill, tc.path)
			if a.ContentType != tc.want || a.ReadStatus != inventory.ReadOK {
				t.Fatalf("got %s %s, want %s ok", a.ContentType, a.ReadStatus, tc.want)
			}
		})
	}
}

type hit struct {
	rule     string
	file     string
	evidence string
}

func hits(findings []finding.Finding) []hit {
	var out []hit
	for _, f := range findings {
		out = append(out, hit{f.Rule, f.Location.File, f.Evidence})
	}
	return out
}

func with(change func(*config.WalkConfig)) config.WalkConfig {
	cfg := open()
	change(&cfg)
	return cfg
}

func TestWalkRules(t *testing.T) {
	pdf := hit{"FILE-PDF", "image.pdf", "matches *.pdf"}
	cases := []struct {
		name string
		dir  string
		cfg  config.WalkConfig
		want []hit
	}{
		{"file over maxFileBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxFileBytes = 100 }), []hit{{"FILE-TOO-LARGE", "SKILL.md", "119 bytes, limit 100"}, pdf}},
		{"file at maxFileBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxFileBytes = 119 }), []hit{pdf}},
		{"bundle over maxBundleBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxBundleBytes = 400 }), []hit{pdf, {"BUNDLE-TOO-LARGE", "scripts/run.sh", "406 bytes in the bundle, limit 400"}}},
		{"bundle at maxBundleBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxBundleBytes = 443 }), []hit{pdf}},
		{"text over maxTextBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxTextBytes = 400 }), []hit{pdf, {"BUNDLE-TOO-MUCH-TEXT", "tool", "418 bytes of text, limit 400"}}},
		{"text at maxTextBytes", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxTextBytes = 418 }), []hit{pdf}},
		{"more files than maxFiles", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxFiles = 13 }), []hit{pdf, {"BUNDLE-TOO-MANY-FILES", "tool", "14 files, limit 13"}}},
		{"files at maxFiles", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxFiles = 14 }), []hit{pdf}},
		{"file below maxDepth", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxDepth = 2 }), []hit{{"BUNDLE-TOO-DEEP", "deep/a/b/file.txt", "depth 3, limit 2"}, pdf}},
		{"file at maxDepth", "testdata/basic", with(func(c *config.WalkConfig) { c.MaxDepth = 3 }), []hit{pdf}},
		{"no file matches a name rule", "testdata/hash/a", open(), nil},
		{"a name rule ignores case", "testdata/named", open(), []hit{{"FILE-PDF", "Report.PDF", "matches *.pdf"}}},
		{"encrypted archives", "testdata/archives", open(), []hit{
			{"ARCHIVE-ENCRYPTED", "secret.7z", "7z archive, contents encrypted"},
			{"ARCHIVE-ENCRYPTED", "secret.zip", "zip archive, contents encrypted"},
		}},
		{"binaries and installers by content or by name", "testdata/formats", open(), []hit{
			{"FILE-BINARY", "Main.class", "Java class"},
			{"FILE-INSTALLER", "Setup.MSI", "matches *.msi, *.msp, *.msm, *.msix, *.msixbundle, *.appx, *.appxbundle, *.ipa, *.dmg, *.pkg.tar.zst, *.pkg.tar.xz"},
			{"FILE-BINARY", "app.jar", "matches *.jar, *.war, *.apk, *.aab"},
			{"FILE-BINARY", "cache.pyc", "Python bytecode"},
			{"FILE-BINARY", "helper", "ELF binary"},
			{"FILE-BINARY", "libfoo.a", "static library"},
			{"FILE-BINARY", "setup.dat", "Windows PE binary"},
			{"FILE-INSTALLER", "tool.deb", "Debian package"},
			{"FILE-BINARY", "universal", "Mach-O universal binary"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := walked(t, tc.dir, tc.cfg)
			if got := hits(s.Findings); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestWalkReplacesInvalidUTF8(t *testing.T) {
	skill := walked(t, "testdata/basic", open()).Skill
	if got := artifact(t, skill, "latin1.txt").Content; got != "caf�\n" {
		t.Fatalf("text = %q", got)
	}
	if notes := artifact(t, skill, "latin1.txt").ScanNotes; len(notes) != 1 || !strings.Contains(notes[0], "1") {
		t.Fatalf("notes = %q", notes)
	}
}

func TestWalkReadsOnlyFilesWithinMaxFileBytes(t *testing.T) {
	cases := []struct {
		name   string
		limit  int64
		status inventory.ReadStatus
		read   bool
	}{
		{"file over the limit is not read", 118, inventory.ReadFailed, false},
		{"file at the limit is read", 119, inventory.ReadOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skill := walked(t, "testdata/basic", with(func(c *config.WalkConfig) { c.MaxFileBytes = tc.limit })).Skill
			a := artifact(t, skill, "SKILL.md")
			if a.ReadStatus != tc.status || (a.Content != "") != tc.read {
				t.Fatalf("got %s with content %q, want %s and read %v", a.ReadStatus, a.Content, tc.status, tc.read)
			}
		})
	}
}
