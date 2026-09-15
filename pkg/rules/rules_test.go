// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func defaults() map[string]map[string]config.RuleSetting {
	return config.Default().Layers.Rules()
}

func withStatic(settings map[string]config.RuleSetting) map[string]map[string]config.RuleSetting {
	byLayer := defaults()
	byLayer[config.StaticLayer] = settings
	return byLayer
}

func TestDirectoryRuleReplacesEmbeddedRuleByID(t *testing.T) {
	set, err := rules.Load("testdata/replace", defaults())
	if err != nil {
		t.Fatal(err)
	}
	want := rules.Rule{
		ID:          "REMOTE-ENDPOINT-UPLOAD",
		Category:    "exfiltration",
		Severity:    finding.Low,
		Title:       "Replaced",
		Description: "Replaced by a directory rule.",
		Kind:        finding.KindHeuristic,
		Layer:       "static",
		Scope:       rules.Any,
		Patterns:    []string{"(?i)replaced"},
		Judge:       rules.JudgePolicy{Downgrade: true},
		Enabled:     true,
	}
	if got := set.ByID["REMOTE-ENDPOINT-UPLOAD"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("REMOTE-ENDPOINT-UPLOAD = %+v, want %+v", got, want)
	}
	isRule := func(c rules.Compiled) bool { return c.ID == "REMOTE-ENDPOINT-UPLOAD" }
	i := slices.IndexFunc(set.Static, isRule)
	if i < 0 || set.Static[i].Title != "Replaced" {
		t.Fatalf("static set does not carry the replacement: %d", i)
	}
	if slices.ContainsFunc(set.Static[i+1:], isRule) {
		t.Fatal("REMOTE-ENDPOINT-UPLOAD appears twice in the static set")
	}
}

func TestDirectoryRuleIsAddedToEmbeddedSet(t *testing.T) {
	set, err := rules.Load("testdata/add", defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := set.ByID["X1"]; !ok {
		t.Fatal("X1 missing from ByID")
	}
	if _, ok := set.ByID["REMOTE-ENDPOINT-UPLOAD"]; !ok {
		t.Fatal("REMOTE-ENDPOINT-UPLOAD missing after adding a directory rule")
	}
}

type values struct {
	category  string
	severity  finding.Severity
	enabled   bool
	interrupt bool
	downgrade bool
	floor     finding.Severity
}

func valuesOf(r rules.Rule) values {
	return values{r.Category, r.Severity, r.Enabled, r.Interrupt, r.Judge.Downgrade, r.Judge.DowngradeFloor}
}

func TestSettingsChangeShippedValues(t *testing.T) {
	on, off := true, false
	low, none := finding.Low, finding.Severity("")
	custom := "my-own-category"
	cases := []struct {
		name     string
		settings map[string]map[string]config.RuleSetting
		id       string
		want     values
		static   bool
	}{
		{"shipped off stays off", defaults(), "PROMPT-INJECTION-SEMANTIC", values{"prompt-injection", finding.Medium, false, false, true, finding.Low}, false},
		{"enabled turns a shipped-off rule on", map[string]map[string]config.RuleSetting{"walk": nil, "reveal": nil, "static": nil, "references": nil, "honeypot": nil, "discovery": {"PROMPT-INJECTION-SEMANTIC": {Enabled: &on}}, "judge": nil}, "PROMPT-INJECTION-SEMANTIC", values{"prompt-injection", finding.Medium, true, false, true, finding.Low}, false},
		{"enabled false turns a rule off", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Enabled: &off}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, false, false, true, finding.Medium}, false},
		{"category changes to any name", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Category: &custom}}), "COMMAND-ON-LOAD", values{"my-own-category", finding.High, true, false, true, finding.Medium}, true},
		{"severity changes", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Severity: &low}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.Low, true, false, true, finding.Medium}, true},
		{"interrupt turns on for a text rule", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Interrupt: &on}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, true, true, true, finding.Medium}, true},
		{"shipped interrupt stays on", defaults(), "FILE-TOO-LARGE", values{"unreviewable", finding.High, true, true, false, ""}, false},
		{"floor changes alone", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Judge: &config.JudgeSetting{DowngradeFloor: &low}}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, true, false, true, finding.Low}, true},
		{"empty floor removes the floor", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Judge: &config.JudgeSetting{DowngradeFloor: &none}}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, true, false, true, ""}, true},
		{"downgrade off keeps the floor", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Judge: &config.JudgeSetting{Downgrade: &off}}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, true, false, false, finding.Medium}, true},
		{"title, description and kind are ignored", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Title: "renamed", Description: "rewritten", Kind: finding.KindFact}}), "COMMAND-ON-LOAD", values{"load-time-exec", finding.High, true, false, true, finding.Medium}, true},
		{"untouched rule keeps its values", withStatic(map[string]config.RuleSetting{"COMMAND-ON-LOAD": {Enabled: &off}}), "REMOTE-SCRIPT-PIPED", values{"remote-execution", finding.Critical, true, false, true, finding.High}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set, err := rules.Load("", tc.settings)
			if err != nil {
				t.Fatal(err)
			}
			if got := valuesOf(set.ByID[tc.id]); got != tc.want {
				t.Fatalf("%s = %+v, want %+v", tc.id, got, tc.want)
			}
			static := slices.ContainsFunc(set.Static, func(c rules.Compiled) bool { return c.ID == tc.id })
			if static != tc.static {
				t.Fatalf("in static set = %v, want %v", static, tc.static)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name     string
		dir      string
		settings map[string]map[string]config.RuleSetting
		want     string
	}{
		{"setting for an unknown rule", "", withStatic(map[string]config.RuleSetting{"NOPE": {Title: "x"}}), "NOPE"},
		{"setting under the wrong layer", "", withStatic(map[string]config.RuleSetting{"PROMPT-INJECTION-SEMANTIC": {Title: "x"}}), "layers.static.rules.PROMPT-INJECTION-SEMANTIC"},
		{"rule in an unknown layer", "testdata/badlayer", defaults(), "X6"},
		{"malformed JSON names the file", "testdata/malformed", defaults(), "broken.json"},
		{"bad regex names file and id", "testdata/badregex", defaults(), "badregex.json: rule X2"},
		{"static rule without patterns", "testdata/mismatch-static", defaults(), "X3"},
		{"llm rule with patterns", "testdata/mismatch-llm", defaults(), "X4"},
		{"rule without a layer", "testdata/nolayer", defaults(), "X5"},
		{"missing directory", "testdata/none", defaults(), "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rules.Load(tc.dir, tc.settings)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
