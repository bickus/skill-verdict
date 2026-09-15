// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
)

func TestLoadMergesPartialFileOverDefaults(t *testing.T) {
	cfg, err := config.Load("testdata/partial.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Model != "openai/gpt-4o-mini" {
		t.Errorf("llm.model = %q", cfg.LLM.Model)
	}
	refs := cfg.Layers.References
	if refs.Timeout != config.Duration(30*time.Second) {
		t.Errorf("layers.references.timeout = %v", time.Duration(refs.Timeout))
	}
	if refs.Budget != config.Duration(120*time.Second) || !refs.Enabled {
		t.Errorf("layers.references = %+v", refs)
	}
	if cfg.Layers.Judge.Enabled || cfg.Layers.Judge.Calls != 3 {
		t.Errorf("layers.judge = %+v", cfg.Layers.Judge)
	}
	if !cfg.Layers.Discovery.Enabled || cfg.Layers.Discovery.Calls != 24 || !cfg.Layers.Static.Enabled {
		t.Errorf("layers = %+v", cfg.Layers)
	}
	if cfg.Layers.Walk.MaxFiles != 50 {
		t.Errorf("layers.walk.maxFiles = %d", cfg.Layers.Walk.MaxFiles)
	}
}

func TestLoadRejects(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"malformed JSON", "testdata/malformed.json"},
		{"unknown key", "testdata/unknown.json"},
		{"duration as number", "testdata/duration-number.json"},
		{"missing explicit file", "testdata/missing.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := config.Load(tc.path); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestValidateRejects(t *testing.T) {
	bad := finding.Severity("severe")
	empty := ""
	cases := []struct {
		name   string
		mutate func(*config.Config)
	}{
		{"unknown collector", func(c *config.Config) { c.Layers.References.Collectors["docker"] = true }},
		{"bad gate severity", func(c *config.Config) { c.Gate.Severity = "severe" }},
		{"zero walk limit", func(c *config.Config) { c.Layers.Walk.MaxFiles = 0 }},
		{"negative timeout", func(c *config.Config) { c.Layers.References.Timeout = -1 }},
		{"zero budget", func(c *config.Config) { c.Layers.References.Budget = 0 }},
		{"zero concurrency", func(c *config.Config) { c.Layers.References.Concurrency = 0 }},
		{"zero call cap", func(c *config.Config) { c.Layers.Judge.Calls = 0 }},
		{"empty setting category", func(c *config.Config) {
			c.Layers.Static.Rules = map[string]config.RuleSetting{"REMOTE-ENDPOINT-UPLOAD": {Category: &empty}}
		}},
		{"bad setting severity", func(c *config.Config) {
			c.Layers.Static.Rules = map[string]config.RuleSetting{"REMOTE-ENDPOINT-UPLOAD": {Severity: &bad}}
		}},
		{"bad setting floor", func(c *config.Config) {
			c.Layers.Static.Rules = map[string]config.RuleSetting{"REMOTE-ENDPOINT-UPLOAD": {Judge: &config.JudgeSetting{DowngradeFloor: &bad}}}
		}},
		{"no model with an LLM layer on", func(c *config.Config) { c.LLM.Model = "" }},
		{"unknown provider", func(c *config.Config) { c.LLM.Provider = "anthropic" }},
		{"zero compaction threshold", func(c *config.Config) { c.LLM.CompactionAt = 0 }},
		{"compaction above the context size", func(c *config.Config) { c.LLM.CompactionAt = c.LLM.InputTokens + 1 }},
		{"negative price", func(c *config.Config) { c.LLM.Price.Output = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.LLM.Model = "m"
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestValidateAccepts(t *testing.T) {
	none := finding.Severity("")
	cases := []struct {
		name   string
		mutate func(*config.Config)
	}{
		{"defaults with a model", func(*config.Config) {}},
		{"setting with a title only", func(c *config.Config) {
			c.Layers.Static.Rules = map[string]config.RuleSetting{"REMOTE-ENDPOINT-UPLOAD": {Title: "Data sent to an external URL"}}
		}},
		{"setting with an empty floor", func(c *config.Config) {
			c.Layers.Static.Rules = map[string]config.RuleSetting{"REMOTE-ENDPOINT-UPLOAD": {Judge: &config.JudgeSetting{DowngradeFloor: &none}}}
		}},
		{"no model with every LLM layer off", func(c *config.Config) {
			c.LLM.Model = ""
			c.Layers.Honeypot.Enabled = false
			c.Layers.Discovery.Enabled = false
			c.Layers.Judge.Enabled = false
		}},
		{"subscription provider", func(c *config.Config) { c.LLM.Provider = config.ChatGPTSubscription }},
		{"compaction at the context size", func(c *config.Config) { c.LLM.CompactionAt = c.LLM.InputTokens }},
		{"prices set", func(c *config.Config) { c.LLM.Price = config.PriceConfig{Input: 3, CacheRead: 0.3, Output: 15} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.LLM.Model = "m"
			tc.mutate(&cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAPIKey(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyFile, []byte("  from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILL_VERDICT_TEST_KEY", " from-env ")
	cases := []struct {
		name    string
		env     string
		file    string
		want    string
		wantErr bool
	}{
		{"from env", "SKILL_VERDICT_TEST_KEY", "", "from-env", false},
		{"from file trimmed", "SKILL_VERDICT_UNSET_KEY", keyFile, "from-file", false},
		{"env before file", "SKILL_VERDICT_TEST_KEY", keyFile, "from-env", false},
		{"nothing configured", "SKILL_VERDICT_UNSET_KEY", "", "", false},
		{"missing file", "SKILL_VERDICT_UNSET_KEY", filepath.Join(t.TempDir(), "none"), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.LLM.KeyEnv = tc.env
			cfg.LLM.KeyFile = tc.file
			got, err := cfg.APIKey()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
