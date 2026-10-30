// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static_test

import "testing"

func TestProtectionOffOfEachTool(t *testing.T) {
	found := inFile(scan(t, "testdata/protection", ""), "PACKAGE-PROTECTION-OFF", "SKILL.md")
	cases := []struct {
		name string
		line int
		want bool
	}{
		{"pip with a zero upload age", 5, true},
		{"pip with a three day upload age", 6, false},
		{"pip with its age variable cleared", 7, true},
		{"pip age variable set", 8, false},
		{"uv with the age limit off", 9, true},
		{"uv with a three day age limit", 10, false},
		{"uv with one package exempt from the age limit", 11, true},
		{"pipx with a zero cooldown", 12, true},
		{"pipx with a seven day cooldown", 13, false},
		{"poetry with a zero release age", 14, true},
		{"poetry with a three day release age", 15, false},
		{"bundler config with a zero cooldown", 16, true},
		{"bundler config with a three day cooldown", 17, false},
		{"bundler cooldown variable at zero", 18, true},
		{"gem config with a zero cooldown", 19, true},
		{"pipenv with a zero cooldown period", 20, true},
		{"pipenv with a three day cooldown period", 21, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := onLine(found, tc.line); got != tc.want {
				t.Errorf("finding on line %d = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}
