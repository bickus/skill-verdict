// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package config

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
)

func (c Config) Validate() error {
	if err := knownKeys("layers.references.collectors", c.Layers.References.Collectors, kindNames()); err != nil {
		return err
	}
	if err := c.validateGate(); err != nil {
		return err
	}
	if err := c.validatePositive(); err != nil {
		return err
	}
	if err := c.validateRules(); err != nil {
		return err
	}
	if err := c.validateModel(); err != nil {
		return err
	}
	if err := c.validatePrices(); err != nil {
		return err
	}
	return c.validateProvider()
}

func knownKeys(prefix string, m map[string]bool, known []string) error {
	for _, key := range slices.Sorted(maps.Keys(m)) {
		if !slices.Contains(known, key) {
			return fmt.Errorf("%s.%s: not one of %s", prefix, key, strings.Join(known, ", "))
		}
	}
	return nil
}

func (c Config) validateGate() error {
	if _, err := finding.ParseSeverity(string(c.Gate.Severity)); err != nil {
		return fmt.Errorf("gate.severity: %w", err)
	}
	return nil
}

type setting[T int64 | float64] struct {
	name  string
	value T
}

func (c Config) validatePositive() error {
	walk, refs := c.Layers.Walk, c.Layers.References
	settings := []setting[int64]{
		{"layers.walk.maxFileBytes", walk.MaxFileBytes},
		{"layers.walk.maxBundleBytes", walk.MaxBundleBytes},
		{"layers.walk.maxTextBytes", walk.MaxTextBytes},
		{"layers.walk.maxFiles", int64(walk.MaxFiles)},
		{"layers.walk.maxDepth", int64(walk.MaxDepth)},
		{"layers.references.timeout", int64(refs.Timeout)},
		{"layers.references.budget", int64(refs.Budget)},
		{"layers.references.concurrency", int64(refs.Concurrency)},
		{"layers.references.maxReferences", int64(refs.MaxReferences)},
		{"layers.references.maxBodyBytes", refs.MaxBodyBytes},
		{"layers.references.maxRedirects", int64(refs.MaxRedirects)},
		{"layers.references.youngDomainDays", int64(refs.YoungDomainDays)},
		{"layers.references.expiryDays", int64(refs.ExpiryDays)},
		{"layers.references.youngOwnerDays", int64(refs.YoungOwnerDays)},
		{"layers.references.newPackageDays", int64(refs.NewPackageDays)},
		{"layers.references.lowDownloads", int64(refs.LowDownloads)},
		{"layers.honeypot.calls", int64(c.Layers.Honeypot.Calls)},
		{"layers.discovery.calls", int64(c.Layers.Discovery.Calls)},
		{"layers.judge.calls", int64(c.Layers.Judge.Calls)},
		{"llm.timeout", int64(c.LLM.Timeout)},
		{"llm.inputTokens", int64(c.LLM.InputTokens)},
		{"llm.compactionAt", int64(c.LLM.CompactionAt)},
		{"llm.maxInputChars", int64(c.LLM.MaxInputChars)},
		{"llm.maxOutputTokens", int64(c.LLM.MaxOutputTokens)},
	}
	for _, s := range settings {
		if s.value <= 0 {
			return fmt.Errorf("%s: must be positive", s.name)
		}
	}
	if refs.RequestDelay < 0 {
		return fmt.Errorf("layers.references.requestDelay: must not be negative")
	}
	for i, d := range refs.RetryDelays {
		if d <= 0 {
			return fmt.Errorf("layers.references.retryDelays[%d]: must be positive", i)
		}
	}
	return nil
}

func (c Config) validateRules() error {
	byLayer := c.Layers.Rules()
	for _, layer := range slices.Sorted(maps.Keys(byLayer)) {
		for _, id := range slices.Sorted(maps.Keys(byLayer[layer])) {
			if err := byLayer[layer][id].validate("layers." + layer + ".rules." + id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s RuleSetting) validate(name string) error {
	if s.Category != nil && *s.Category == "" {
		return fmt.Errorf("%s.category: must not be empty", name)
	}
	if s.Severity != nil {
		if _, err := finding.ParseSeverity(string(*s.Severity)); err != nil {
			return fmt.Errorf("%s.severity: %w", name, err)
		}
	}
	if s.Judge == nil || s.Judge.DowngradeFloor == nil || *s.Judge.DowngradeFloor == "" {
		return nil
	}
	if _, err := finding.ParseSeverity(string(*s.Judge.DowngradeFloor)); err != nil {
		return fmt.Errorf("%s.judge.downgradeFloor: %w", name, err)
	}
	return nil
}

func ValidGlobs(name string, globs []string) error {
	for _, g := range globs {
		if _, err := path.Match(g, ""); err != nil {
			return fmt.Errorf("%s: bad glob %q", name, g)
		}
	}
	return nil
}

func (c Config) validateModel() error {
	if c.LLM.Model != "" {
		return nil
	}
	if c.Layers.Honeypot.Enabled {
		return fmt.Errorf("llm.model: required while layers.%s.enabled is true", HoneypotLayer)
	}
	if c.Layers.Discovery.Enabled {
		return fmt.Errorf("llm.model: required while layers.%s.enabled is true", DiscoveryLayer)
	}
	if c.Layers.Judge.Enabled {
		return fmt.Errorf("llm.model: required while layers.%s.enabled is true", JudgeLayer)
	}
	return nil
}

func (c Config) validatePrices() error {
	p := c.LLM.Price
	prices := []setting[float64]{
		{"llm.price.input", p.Input},
		{"llm.price.cacheWrite", p.CacheWrite},
		{"llm.price.cacheRead", p.CacheRead},
		{"llm.price.output", p.Output},
	}
	for _, s := range prices {
		if s.value < 0 {
			return fmt.Errorf("%s: must not be negative", s.name)
		}
	}
	return nil
}

func (c Config) validateProvider() error {
	if !slices.Contains(Providers(), c.LLM.Provider) {
		return fmt.Errorf("llm.provider: not one of %s", strings.Join(Providers(), ", "))
	}
	if c.LLM.CompactionAt > c.LLM.InputTokens {
		return fmt.Errorf("llm.compactionAt: must not exceed llm.inputTokens")
	}
	return nil
}

func MatchAny(globs []string, name string) bool {
	for _, g := range globs {
		if ok, err := path.Match(g, name); err == nil && ok {
			return true
		}
	}
	return false
}
