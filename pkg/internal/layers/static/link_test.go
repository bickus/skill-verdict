// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static_test

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
)

func skillWithLinks(t *testing.T, files, links map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	return root
}

type linkHit struct {
	rule     string
	evidence string
}

func linkHits(s *pipeline.State, file string) []linkHit {
	var out []linkHit
	for _, f := range s.Findings {
		if f.Location.File == file {
			out = append(out, linkHit{f.Rule, f.Evidence})
		}
	}
	slices.SortFunc(out, func(a, b linkHit) int { return cmp.Compare(a.rule, b.rule) })
	return out
}

func TestLinkScopeMatchesOnlySymlinkTargets(t *testing.T) {
	root := skillWithLinks(t,
		map[string]string{"SKILL.md": "# s\n", "probe.txt": "probeany probelink\n"},
		map[string]string{"probe.link": "probeany/probelink"},
	)
	s := scan(t, root, "testdata/rules")
	cases := []struct {
		file string
		want []hit
	}{
		{"probe.link", []hit{{"T-LINK", 0, "probe.link"}}},
		{"probe.txt", []hit{{"T-ANY", 1, "probe.txt"}}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			if got := probes(s, tc.file); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSymlinkTargetFindings(t *testing.T) {
	cases := []struct {
		name   string
		link   string
		target string
		want   []linkHit
	}{
		{"env file above the skill", "env", "../../.env", []linkHit{{"SYMLINK-OUT-OF-SKILL", "../../.env"}, {"SYMLINK-TO-SECRETS", "../../.env"}}},
		{"secret folder by absolute path", "key", "/home/alice/.ssh/id_ed25519", []linkHit{{"SYMLINK-OUT-OF-SKILL", "/home/alice/.ssh/id_ed25519"}, {"SYMLINK-TO-SECRETS", "/home/alice/.ssh/id_ed25519"}}},
		{"home folder", "home", "/Users/alice", []linkHit{{"SYMLINK-OUT-OF-SKILL", "/Users/alice"}, {"SYMLINK-TO-SECRETS", "/Users/alice"}}},
		{"windows home folder", "win", `C:\Users\alice`, []linkHit{{"SYMLINK-OUT-OF-SKILL", `C:\Users\alice`}, {"SYMLINK-TO-SECRETS", `C:\Users\alice`}}},
		{"filesystem root", "root", "/", []linkHit{{"SYMLINK-OUT-OF-SKILL", "/"}, {"SYMLINK-TO-SECRETS", "/"}}},
		{"folder above the skill", "parent", "../..", []linkHit{{"SYMLINK-OUT-OF-SKILL", "../.."}, {"SYMLINK-TO-SECRETS", "../.."}}},
		{"file outside the skill", "license", "../shared/LICENSE", []linkHit{{"SYMLINK-OUT-OF-SKILL", "../shared/LICENSE"}}},
		{"links inside the skill that lead out together", "chain", "docs/back/../shared", []linkHit{{"SYMLINK-OUT-OF-SKILL", "docs/back/../shared"}}},
		{"tilde left unexpanded", "tilde", "~/.ssh", []linkHit{{"SYMLINK-TO-SECRETS", "~/.ssh"}}},
		{"link back to the skill root", "docs/back", "..", nil},
	}
	links := make(map[string]string, len(cases))
	for _, tc := range cases {
		links[tc.link] = tc.target
	}
	root := skillWithLinks(t, map[string]string{"SKILL.md": "# s\n", "docs/guide.md": "# guide\n"}, links)
	s := scan(t, root, "")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := linkHits(s, tc.link); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
