// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
)

func TestRunExitCodeAndOutput(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
		out  string
	}{
		{"block fails by default", []string{"scan", "--config", "testdata/config.json", "testdata/block"}, 1, "COMMAND-ON-LOAD"},
		{"review passes by default", []string{"scan", "--config", "testdata/config.json", "testdata/review"}, 0, "UNPINNED-DEPENDENCY"},
		{"fail-on review fails review", []string{"scan", "--config", "testdata/config.json", "--fail-on", "review", "testdata/review"}, 1, "UNPINNED-DEPENDENCY"},
		{"fail-on incomplete passes review", []string{"scan", "--config", "testdata/config.json", "--fail-on", "incomplete", "testdata/review"}, 0, "UNPINNED-DEPENDENCY"},
		{"fail-on never passes block", []string{"scan", "--config", "testdata/config.json", "--fail-on", "never", "testdata/block"}, 0, "COMMAND-ON-LOAD"},
		{"json report", []string{"scan", "--config", "testdata/config.json", "--format", "json", "testdata/block"}, 1, `"rule": "COMMAND-ON-LOAD"`},
		{"unknown fail-on level", []string{"scan", "--config", "testdata/config.json", "--fail-on", "maybe", "testdata/block"}, 2, ""},
		{"unknown report format", []string{"scan", "--config", "testdata/config.json", "--format", "xml", "testdata/block"}, 2, ""},
		{"missing config file", []string{"scan", "--config", "testdata/none.json", "testdata/block"}, 2, ""},
		{"missing path", []string{"scan"}, 2, ""},
		{"directory without skills", []string{"scan", "--config", "testdata/config.json", "testdata/empty"}, 2, ""},
		{"rules as text", []string{"rules", "--config", "testdata/config.json"}, 0, "COMMAND-ON-LOAD"},
		{"rules as markdown", []string{"rules", "--config", "testdata/config.json", "--format", "markdown"}, 0, "| COMMAND-ON-LOAD |"},
		{"rules in an unknown format", []string{"rules", "--config", "testdata/config.json", "--format", "yaml"}, 2, ""},
		{"config-template refuses an existing file", []string{"config-template", "testdata/config.json"}, 2, ""},
		{"config-template without a path", []string{"config-template"}, 2, ""},
		{"no command", nil, 2, ""},
		{"unknown command", []string{"lint"}, 2, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if code := run(tc.args, &out); code != tc.code {
				t.Errorf("exit code %d, want %d", code, tc.code)
			}
			if !strings.Contains(out.String(), tc.out) {
				t.Errorf("output lacks %q:\n%s", tc.out, out.String())
			}
		})
	}
}

func TestConfigTemplateLoadsAsConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	var out strings.Builder
	if code := run([]string{"config-template", path}, &out); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.LLM.Model = "m"
	if err = cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
