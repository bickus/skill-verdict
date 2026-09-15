// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
)

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func link(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func TestWalkSymlinkInsideTree(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "SKILL.md"), "# s\n")
	link(t, "SKILL.md", filepath.Join(root, "link.md"))
	skill := walked(t, root, open()).Skill
	a := artifact(t, skill, "link.md")
	if a.ReadStatus != inventory.ReadOK || a.ContentType != inventory.Symlink || a.Content != "SKILL.md" {
		t.Fatalf("got %+v, want ok symlink to SKILL.md", a)
	}
}

func TestWalkSymlinkedRoot(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	write(t, filepath.Join(real, "SKILL.md"), "# s\n")
	alias := filepath.Join(base, "alias")
	link(t, real, alias)
	skill := walked(t, alias, open()).Skill
	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	if skill.Root != want {
		t.Fatalf("root = %s, want %s", skill.Root, want)
	}
	if got := artifact(t, skill, "SKILL.md").ReadStatus; got != inventory.ReadOK {
		t.Fatalf("got %s, want %s", got, inventory.ReadOK)
	}
}

func TestWalkUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("cannot make a file unreadable here")
	}
	root := t.TempDir()
	write(t, filepath.Join(root, "SKILL.md"), "# s\n")
	locked := filepath.Join(root, "secret.txt")
	write(t, locked, "x\n")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	skill := walked(t, root, open()).Skill
	if got := artifact(t, skill, "secret.txt").ReadStatus; got != inventory.ReadFailed {
		t.Fatalf("got %s, want %s", got, inventory.ReadFailed)
	}
}

func TestResolveErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{"missing root", func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "none")
		}},
		{"root is a file", func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "SKILL.md")
			write(t, p, "# s\n")
			return p
		}},
		{"no SKILL.md", func(t *testing.T) string {
			return t.TempDir()
		}},
		{"SKILL.md is a directory", func(t *testing.T) string {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "SKILL.md"), 0o750); err != nil {
				t.Fatal(err)
			}
			return root
		}},
		{"SKILL.md is a symlink", func(t *testing.T) string {
			root := t.TempDir()
			write(t, filepath.Join(root, "real.md"), "# s\n")
			link(t, "real.md", filepath.Join(root, "SKILL.md"))
			return root
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := walk.Resolve(tc.setup(t)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
