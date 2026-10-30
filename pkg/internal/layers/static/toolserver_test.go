// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static_test

import "testing"

func TestRemoteToolServerOfEachForm(t *testing.T) {
	s := scan(t, "testdata/toolserver", "")
	cases := []struct {
		name string
		file string
		line int
		want bool
	}{
		{"mcp add of a remote server", "SKILL.md", 5, true},
		{"mcp add with a url flag", "SKILL.md", 6, true},
		{"mcp add of a local server", "SKILL.md", 7, false},
		{"bridge to a remote server", "SKILL.md", 8, true},
		{"link to a page about MCP", "SKILL.md", 9, false},
		{"configuration entry with a type and a url", ".mcp.json", 8, true},
		{"configuration block that starts with a local server", ".mcp.json", 2, false},
		{"configuration block that starts with a url entry", "mcp.json", 2, true},
		{"entry with a url before its type", "settings.json", 11, true},
		{"entry with an http url key", "settings.json", 8, true},
		{"bridge with the url on the next line", "settings.json", 18, true},
		{"server table with a url", "config.toml", 5, true},
		{"server table with a command", "config.toml", 1, false},
		{"package repository with a url", "package.json", 3, false},
		{"package issue tracker with a url", "package.json", 4, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := onLine(inFile(s, "TOOL-SERVER-REMOTE", tc.file), tc.line); got != tc.want {
				t.Errorf("finding in %s on line %d = %v, want %v", tc.file, tc.line, got, tc.want)
			}
		})
	}
}
