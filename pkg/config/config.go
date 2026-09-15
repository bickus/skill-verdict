// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package config

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a string such as \"30s\": %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

type Config struct {
	Layers LayersConfig `json:"layers"`
	LLM    LLMConfig    `json:"llm"`
	Gate   GateConfig   `json:"gate"`
	Rules  RulesConfig  `json:"rules"`
}

const (
	WalkLayer       = "walk"
	RevealLayer     = "reveal"
	StaticLayer     = "static"
	ReferencesLayer = "references"
	HoneypotLayer   = "honeypot"
	DiscoveryLayer  = "discovery"
	JudgeLayer      = "judge"
)

type LayersConfig struct {
	Walk       WalkConfig       `json:"walk"`
	Reveal     LayerConfig      `json:"reveal"`
	Static     LayerConfig      `json:"static"`
	References ReferencesConfig `json:"references"`
	Honeypot   ModelLayerConfig `json:"honeypot"`
	Discovery  ModelLayerConfig `json:"discovery"`
	Judge      ModelLayerConfig `json:"judge"`
}

func (l LayersConfig) Rules() map[string]map[string]RuleSetting {
	return map[string]map[string]RuleSetting{
		WalkLayer:       l.Walk.Rules,
		RevealLayer:     l.Reveal.Rules,
		StaticLayer:     l.Static.Rules,
		ReferencesLayer: l.References.Rules,
		HoneypotLayer:   l.Honeypot.Rules,
		DiscoveryLayer:  l.Discovery.Rules,
		JudgeLayer:      l.Judge.Rules,
	}
}

func (l *LayersConfig) SetRules(byLayer map[string]map[string]RuleSetting) {
	l.Walk.Rules = byLayer[WalkLayer]
	l.Reveal.Rules = byLayer[RevealLayer]
	l.Static.Rules = byLayer[StaticLayer]
	l.References.Rules = byLayer[ReferencesLayer]
	l.Honeypot.Rules = byLayer[HoneypotLayer]
	l.Discovery.Rules = byLayer[DiscoveryLayer]
	l.Judge.Rules = byLayer[JudgeLayer]
}

func (l LayersConfig) ModelEnabled() bool {
	return l.Honeypot.Enabled || l.Discovery.Enabled || l.Judge.Enabled
}

type LayerConfig struct {
	Enabled bool                   `json:"enabled"`
	Rules   map[string]RuleSetting `json:"rules,omitempty"`
}

type ModelLayerConfig struct {
	Enabled bool                   `json:"enabled"`
	Calls   int                    `json:"calls"`
	Rules   map[string]RuleSetting `json:"rules,omitempty"`
}

type RulesConfig struct {
	Dir string `json:"dir"`
}

type RuleSetting struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Kind        finding.Kind      `json:"kind,omitempty"`
	Category    *string           `json:"category,omitempty"`
	Severity    *finding.Severity `json:"severity,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Interrupt   *bool             `json:"interrupt,omitempty"`
	Judge       *JudgeSetting     `json:"judge,omitempty"`
	Attribution json.RawMessage   `json:"attribution,omitempty"`
}

type JudgeSetting struct {
	Downgrade      *bool             `json:"downgrade,omitempty"`
	DowngradeFloor *finding.Severity `json:"downgradeFloor,omitempty"`
	Instructions   *string           `json:"instructions,omitempty"`
}

type WalkConfig struct {
	MaxFileBytes   int64                  `json:"maxFileBytes"`
	MaxBundleBytes int64                  `json:"maxBundleBytes"`
	MaxTextBytes   int64                  `json:"maxTextBytes"`
	MaxFiles       int                    `json:"maxFiles"`
	MaxDepth       int                    `json:"maxDepth"`
	Rules          map[string]RuleSetting `json:"rules,omitempty"`
}

type ReferencesConfig struct {
	Enabled         bool                   `json:"enabled"`
	Collectors      map[string]bool        `json:"collectors"`
	Timeout         Duration               `json:"timeout"`
	Budget          Duration               `json:"budget"`
	Concurrency     int                    `json:"concurrency"`
	RequestDelay    Duration               `json:"requestDelay"`
	RetryDelays     []Duration             `json:"retryDelays"`
	MaxReferences   int                    `json:"maxReferences"`
	MaxBodyBytes    int64                  `json:"maxBodyBytes"`
	MaxRedirects    int                    `json:"maxRedirects"`
	YoungDomainDays int                    `json:"youngDomainDays"`
	ExpiryDays      int                    `json:"expiryDays"`
	YoungOwnerDays  int                    `json:"youngOwnerDays"`
	NewPackageDays  int                    `json:"newPackageDays"`
	LowDownloads    int                    `json:"lowDownloads"`
	GitHubTokenEnv  string                 `json:"githubTokenEnv"`
	Rules           map[string]RuleSetting `json:"rules,omitempty"`
}

type LLMConfig struct {
	Provider        string      `json:"provider"`
	BaseURL         string      `json:"baseUrl"`
	Model           string      `json:"model"`
	KeyEnv          string      `json:"keyEnv"`
	KeyFile         string      `json:"keyFile"`
	Timeout         Duration    `json:"timeout"`
	ReasoningEffort string      `json:"reasoningEffort"`
	InputTokens     int         `json:"inputTokens"`
	CompactionAt    int         `json:"compactionAt"`
	MaxInputChars   int         `json:"maxInputChars"`
	MaxOutputTokens int         `json:"maxOutputTokens"`
	Price           PriceConfig `json:"price"`
}

type PriceConfig struct {
	Input      float64 `json:"input"`
	CacheWrite float64 `json:"cacheWrite"`
	CacheRead  float64 `json:"cacheRead"`
	Output     float64 `json:"output"`
}

const perMillion = 1_000_000

func (p PriceConfig) Cost(input, cached, output int) float64 {
	uncached, out := float64(input-cached), float64(output)
	micro := float64(cached)*p.CacheRead + out*p.Output
	if p.CacheWrite > 0 {
		micro += (uncached + out) * p.CacheWrite
	} else {
		micro += uncached * p.Input
	}
	return math.Round(micro) / perMillion
}

const (
	OpenAICompatible    = "openai-compatible"
	ChatGPTSubscription = "chatgpt-subscription"
)

func Providers() []string {
	return []string{OpenAICompatible, ChatGPTSubscription}
}

type GateConfig struct {
	Severity finding.Severity `json:"severity"`
}

const githubEnvVar = "GITHUB_TOKEN"

func kindNames() []string {
	names := make([]string, 0, len(refs.Kinds()))
	for _, kind := range refs.Kinds() {
		names = append(names, string(kind))
	}
	return names
}

func Default() Config {
	collectors := make(map[string]bool)
	for _, name := range kindNames() {
		collectors[name] = true
	}
	return Config{
		Layers: LayersConfig{
			Walk: WalkConfig{
				MaxFileBytes:   5 << 20,
				MaxBundleBytes: 64 << 20,
				MaxTextBytes:   400 << 10,
				MaxFiles:       50,
				MaxDepth:       5,
			},
			Reveal: LayerConfig{Enabled: true},
			Static: LayerConfig{Enabled: true},
			References: ReferencesConfig{
				Enabled:         true,
				Collectors:      collectors,
				Timeout:         Duration(10 * time.Second),
				Budget:          Duration(120 * time.Second),
				Concurrency:     8,
				RequestDelay:    Duration(time.Second),
				RetryDelays:     []Duration{Duration(2 * time.Second), Duration(5 * time.Second)},
				MaxReferences:   100,
				MaxBodyBytes:    1 << 20,
				MaxRedirects:    5,
				YoungDomainDays: 365,
				ExpiryDays:      14,
				YoungOwnerDays:  90,
				NewPackageDays:  90,
				LowDownloads:    1000,
				GitHubTokenEnv:  githubEnvVar,
			},
			Honeypot:  ModelLayerConfig{Enabled: true, Calls: 10},
			Discovery: ModelLayerConfig{Enabled: true, Calls: 24},
			Judge:     ModelLayerConfig{Enabled: true, Calls: 15},
		},
		LLM: LLMConfig{
			Provider:        OpenAICompatible,
			BaseURL:         "https://openrouter.ai/api/v1",
			KeyEnv:          "SKILL_VERDICT_API_KEY",
			Timeout:         Duration(120 * time.Second),
			InputTokens:     200000,
			CompactionAt:    160000,
			MaxInputChars:   200000,
			MaxOutputTokens: 8192,
		},
		Gate: GateConfig{
			Severity: finding.High,
		},
	}
}
