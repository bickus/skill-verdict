// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk_test

import (
	"testing"
)

func TestFrontmatter(t *testing.T) {
	cases := []struct {
		dir         string
		name        string
		description string
	}{
		{"testdata/frontmatter/plain", "plain-skill", "Plain description"},
		{"testdata/frontmatter/quoted", "quoted-skill", "Quoted description"},
		{"testdata/frontmatter/folded", "folded-skill", "Folded line one line two"},
		{"testdata/frontmatter/literal", "literal-skill", "Line one\nLine two"},
		{"testdata/frontmatter/indented", "indented", "ok"},
		{"testdata/frontmatter/absent", "absent", ""},
		{"testdata/frontmatter/malformed", "malformed", ""},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			skill := walked(t, tc.dir, open()).Skill
			if skill.Name != tc.name {
				t.Errorf("name = %q, want %q", skill.Name, tc.name)
			}
			if skill.Description != tc.description {
				t.Errorf("description = %q, want %q", skill.Description, tc.description)
			}
		})
	}
}
