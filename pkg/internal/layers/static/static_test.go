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
		if f.Location.File == file && strings.HasPrefix(f.Rule, "T-") {
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

func TestGlobExcludeAndOneFindingPerLine(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	cases := []struct {
		name string
		file string
		want []hit
	}{
		{"glob matches base name", "glob.cfg", []hit{{"T-GLOB", 1, "glob.cfg"}}},
		{"glob rejects other names", "glob.txt", nil},
		{"glob with a slash matches the end of the path", "plugins/p/commands/run.md", []hit{{"T-PATH", 1, "plugins/p/commands/run.md"}}},
		{"glob with a slash matches whole folder names only", "subcommands/run.md", nil},
		{"exclude applies to its own line only", "excl.txt", []hit{{"T-EXCL", 2, "excl.txt"}}},
		{"line with an invisible character is reported once", "dedup.txt", []hit{{"T-ANY", 1, "dedup.txt"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := probes(s, tc.file); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMatchInCleanTextPointsAtTheWrittenLine(t *testing.T) {
	s := scan(t, "testdata/mechanics", "testdata/rules")
	cases := []struct {
		name string
		want finding.Finding
	}{
		{"homoglyph", finding.Finding{
			Rule:     "T-ANY",
			Severity: finding.Low,
			Location: finding.Location{File: "homoglyph.txt", Line: 3},
			Evidence: "probeany",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inFile(s, tc.want.Rule, tc.want.Location.File)
			if !reflect.DeepEqual(got, []finding.Finding{tc.want}) {
				t.Errorf("got %+v, want %+v", got, tc.want)
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
		{"supply-chain", "unpinned.md", ""},
		{"remote-execution", "install.sh", "install-miss.sh"},
		{"obfuscation", "obf.py", "obf-miss.py"},
		{"deserialization", "load.rb", "load-miss.rb"},
		{"ssrf", "meta.py", "meta-miss.py"},
		{"tool-misuse", "client.py", "client-miss.py"},
		{"load-time-exec", "commands/loadtime.md", "commands/loadtime-miss.md"},
		{"privilege-escalation", "sudoers.sh", ""},
		{"prompt-injection", "skillref.md", "skillref-miss.md"},
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

func TestSecretEnvNameMatchesWholeTokens(t *testing.T) {
	s := scan(t, "testdata/categories", "")
	found := inFile(s, "SECRETS-ENV-USED", "secrets-env.sh")
	cases := []struct {
		name string
		line int
		want bool
	}{
		{"name ending in _PAT", 1, true},
		{"_PAT_ inside the name", 2, true},
		{"bare PAT", 3, true},
		{"_PATH is not a token", 4, false},
		{"PATCH is not a token", 5, false},
		{"_SECRET_PATH names a file", 6, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := onLine(found, tc.line); got != tc.want {
				t.Errorf("finding on line %d = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func onLine(found []finding.Finding, line int) bool {
	return slices.ContainsFunc(found, func(f finding.Finding) bool { return f.Location.Line == line })
}

func severityOnLine(found []finding.Finding, line int) finding.Severity {
	i := slices.IndexFunc(found, func(f finding.Finding) bool { return f.Location.Line == line })
	if i < 0 {
		return ""
	}
	return found[i].Severity
}

func TestCommandOnLoadSeverity(t *testing.T) {
	s := scan(t, "testdata/loadtime", "")
	cases := []struct {
		name string
		file string
		line int
		want finding.Severity
	}{
		{"a read-only command is medium", "SKILL.md", 7, finding.Medium},
		{"a command outside the read-only list is high", "SKILL.md", 8, finding.High},
		{"a read-only command with a pipe is high", "SKILL.md", 9, finding.High},
		{"a script shipped in the skill folder is medium", "SKILL.md", 10, finding.Medium},
		{"a path that leaves the skill folder is high", "SKILL.md", 11, finding.High},
		{"a fenced block of read-only commands is medium", "SKILL.md", 13, finding.Medium},
		{"a fenced block that runs a shipped script is medium", "SKILL.md", 17, finding.Medium},
		{"a fenced block that fetches is high", "SKILL.md", 21, finding.High},
		{"a command file in a nested commands folder runs too", "commands/git/log.md", 1, finding.Medium},
		{"a file the agent does not load gives no finding", "notes.md", 1, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := severityOnLine(inFile(s, "COMMAND-ON-LOAD", tc.file), tc.line); got != tc.want {
				t.Errorf("severity on line %d = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestUnpinnedDependencyNeedsAnInstall(t *testing.T) {
	cases := []struct {
		name     string
		dir      string
		manifest string
		want     int
	}{
		{"an installer in the skill keeps the finding", "testdata/dependencies/installed", "package.json", 1},
		{"a file that names the manifest keeps the finding", "testdata/dependencies/named", "requirements.txt", 1},
		{"an installer of another ecosystem gives no finding", "testdata/dependencies/other", "package.json", 0},
		{"a manifest nothing installs gives no finding", "testdata/dependencies/unused", "requirements.txt", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := scan(t, tc.dir, "")
			if got := len(inFile(s, "UNPINNED-DEPENDENCY", tc.manifest)); got != tc.want {
				t.Errorf("%d findings in %s, want %d", got, tc.manifest, tc.want)
			}
		})
	}
}

func TestUnpinnedDependencyLockfileFixesVersions(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		want int
	}{
		{"a lockfile next to the manifest gives no finding", "testdata/dependencies/locked", 0},
		{"a lockfile in another folder keeps the finding", "testdata/dependencies/elsewhere", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := scan(t, tc.dir, "")
			if got := len(inFile(s, "UNPINNED-DEPENDENCY", "package.json")); got != tc.want {
				t.Errorf("%d findings in package.json, want %d", got, tc.want)
			}
		})
	}
}

func TestUnpinnedInstallRuleOfEachTool(t *testing.T) {
	s := scan(t, "testdata/unpinned", "")
	cases := []struct {
		name string
		rule string
		line int
		want bool
	}{
		{"npm install without a version", "UNPINNED-NPM", 5, true},
		{"npm install with an exact version", "UNPINNED-NPM", 6, false},
		{"bunx without a version", "UNPINNED-NPM", 7, true},
		{"npx with a package flag", "UNPINNED-NPM", 8, true},
		{"deno run of an npm spec", "UNPINNED-NPM", 9, true},
		{"pip install without a version", "UNPINNED-PYPI", 10, true},
		{"pip install with an exact version", "UNPINNED-PYPI", 11, false},
		{"uv tool install without a version", "UNPINNED-PYPI", 12, true},
		{"gem install without a version", "UNPINNED-GEM", 13, true},
		{"gem install with a version", "UNPINNED-GEM", 14, false},
		{"cargo verbose flag is no version", "UNPINNED-CRATE", 15, true},
		{"cargo install with a version", "UNPINNED-CRATE", 16, false},
		{"docker pull without a tag", "UNPINNED-IMAGE", 17, true},
		{"docker pull with a version tag", "UNPINNED-IMAGE", 18, false},
		{"brew tap of a third party", "UNPINNED-TAP", 19, true},
		{"brew tap of homebrew", "UNPINNED-TAP", 20, false},
		{"clawhub install without a version", "UNPINNED-CLAWHUB", 21, true},
		{"clawhub install with a version", "UNPINNED-CLAWHUB", 22, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := onLine(inFile(s, tc.rule, "SKILL.md"), tc.line); got != tc.want {
				t.Errorf("%s on line %d = %v, want %v", tc.rule, tc.line, got, tc.want)
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
