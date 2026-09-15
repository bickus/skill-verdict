// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static_test

import (
	"cmp"
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/reveal"
	"github.com/bickus/skill-verdict/pkg/internal/layers/static"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func newState(t *testing.T, skillDir, rulesDir string) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	layers := config.Default().Layers
	set, err := rules.Load(rulesDir, layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	return s
}

func scan(t *testing.T, skillDir, rulesDir string) *pipeline.State {
	t.Helper()
	s := newState(t, skillDir, rulesDir)
	pipeline.Run(context.Background(), s, []pipeline.Layer{reveal.Layer{}, static.Layer{}})
	return s
}

type hit struct {
	rule string
	line int
	file string
}

func probes(s *pipeline.State, file string) []hit {
	var out []hit
	for _, f := range s.Findings {
		if inventory.OriginalPath(f.Location.File) == file && strings.HasPrefix(f.Rule, "T-") {
			out = append(out, hit{f.Rule, f.Location.Line, f.Location.File})
		}
	}
	slices.SortFunc(out, func(a, b hit) int {
		return cmp.Or(cmp.Compare(a.rule, b.rule), cmp.Compare(a.line, b.line), cmp.Compare(a.file, b.file))
	})
	return out
}

func inFile(s *pipeline.State, rule, file string) []finding.Finding {
	var out []finding.Finding
	for _, f := range s.Findings {
		if f.Rule == rule && f.Location.File == file {
			out = append(out, f)
		}
	}
	return out
}

func hasClass(s *pipeline.State, class string, file string) bool {
	return slices.ContainsFunc(s.Findings, func(f finding.Finding) bool {
		return s.Rules.ByID[f.Rule].Category == class && f.Location.File == file
	})
}

func TestScopeFollowsArtifactKindAndFences(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	cases := []struct {
		file string
		want []hit
	}{
		{"matrix.md", []hit{
			{"T-ANY", 2, "matrix.md"}, {"T-ANY", 4, "matrix.md"}, {"T-ANY", 7, "matrix.md"}, {"T-ANY", 10, "matrix.md"}, {"T-ANY", 12, "matrix.md"}, {"T-ANY", 14, "matrix.md"},
			{"T-CODE", 4, "matrix.md"}, {"T-CODE", 7, "matrix.md"}, {"T-CODE", 12, "matrix.md"}, {"T-CODE", 14, "matrix.md"},
			{"T-PROSE", 2, "matrix.md"}, {"T-PROSE", 10, "matrix.md"},
		}},
		{"matrix.py", []hit{{"T-ANY", 1, "matrix.py"}, {"T-CODE", 1, "matrix.py"}}},
		{"matrix.json", []hit{{"T-ANY", 1, "matrix.json"}, {"T-CODE", 1, "matrix.json"}}},
		{"matrix.txt", []hit{{"T-ANY", 1, "matrix.txt"}, {"T-PROSE", 1, "matrix.txt"}}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			if got := probes(s, tc.file); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGlobExcludeAndRevealedDedup(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	cases := []struct {
		name string
		file string
		want []hit
	}{
		{"glob matches base name", "glob.cfg", []hit{{"T-GLOB", 1, "glob.cfg"}}},
		{"glob rejects other names", "glob.txt", nil},
		{"exclude applies to its own line only", "excl.txt", []hit{{"T-EXCL", 2, "excl.txt"}}},
		{"same line in original and revealed is credited to the original", "dedup.txt", []hit{{"T-ANY", 1, "dedup.txt"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := probes(s, tc.file); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRevealedArtifactFindings(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	cases := []struct {
		name string
		want finding.Finding
	}{
		{"homoglyph", finding.Finding{
			Rule:     "T-ANY",
			Severity: finding.Low,
			Location: finding.Location{File: "homoglyph.txt.revealed", Line: 3},
			Evidence: "probeany",
		}},
		{"letter spacing", finding.Finding{
			Rule:     "T-ANY",
			Severity: finding.Low,
			Location: finding.Location{File: "spaced.txt.revealed", Line: 2},
			Evidence: "probeany",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inFile(s, tc.want.Rule, tc.want.Location.File)
			if !reflect.DeepEqual(got, []finding.Finding{tc.want}) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
			if original := inFile(s, tc.want.Rule, inventory.OriginalPath(tc.want.Location.File)); len(original) != 0 {
				t.Errorf("original has %+v", original)
			}
		})
	}
}

func TestFiftyFindingsPerRulePerFile(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	if got := len(probes(s, "cap.txt")); got != 50 {
		t.Errorf("cap.txt produced %d findings, want 50", got)
	}
	want := []string{"static: T-ANY capped at 50"}
	if got := s.Skill.Artifact("cap.txt").ScanNotes; !reflect.DeepEqual(got, want) {
		t.Errorf("notes = %v, want %v", got, want)
	}
}

func TestEveryClassHasHitAndNearMiss(t *testing.T) {
	s := scan(t, "testdata/categories", "")
	cases := []struct {
		class string
		hit   string
		miss  string
	}{
		{"exfiltration", "exfil.py", "exfil-miss.py"},
		{"agent-snooping", "snoop.py", "snoop-miss.py"},
		{"supply-chain", "requirements.txt", "requirements-dev.txt"},
		{"remote-execution", "install.sh", "install-miss.sh"},
		{"obfuscation", "obf.py", "obf-miss.py"},
		{"deserialization", "load.rb", "load-miss.rb"},
		{"ssrf", "meta.py", "meta-miss.py"},
		{"tool-misuse", "client.py", "client-miss.py"},
		{"load-time-exec", "loadtime.md", "loadtime-miss.md"},
		{"privilege-escalation", "sudoers.sh", ""},
	}
	for _, tc := range cases {
		t.Run(string(tc.class), func(t *testing.T) {
			if !hasClass(s, tc.class, tc.hit) {
				t.Errorf("%s produced no %s finding", tc.hit, tc.class)
			}
			if hasClass(s, tc.class, tc.miss) {
				t.Errorf("%s produced a %s finding", tc.miss, tc.class)
			}
		})
	}
}

func TestRunRecordsLayerOutcome(t *testing.T) {
	cases := []struct {
		name   string
		cancel bool
		want   pipeline.LayerStatus
	}{
		{"finished layer", false, pipeline.LayerStatus{Layer: "static", Status: "done"}},
		{"cancelled scan", true, pipeline.LayerStatus{Layer: "static", Status: "skipped"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, "testdata/mechanics", "testdata/rules")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			pipeline.Run(ctx, s, []pipeline.Layer{static.Layer{}})
			if want := []pipeline.LayerStatus{{Layer: "walk", Status: "done"}, tc.want}; !reflect.DeepEqual(s.Layers, want) {
				t.Errorf("layers %+v, want %+v", s.Layers, want)
			}
		})
	}
}
