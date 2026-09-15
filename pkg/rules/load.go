// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
)

//go:embed data/*.json
var embedded embed.FS

type file struct {
	Attribution json.RawMessage `json:"attribution"`
	Rules       []Rule          `json:"rules"`
}

func Load(dir string, byLayer map[string]map[string]config.RuleSetting) (*Set, error) {
	merged, err := loadEmbedded()
	if err != nil {
		return nil, err
	}
	if dir != "" {
		if merged, err = loadDir(merged, dir); err != nil {
			return nil, err
		}
	}
	if merged, err = apply(merged, byLayer); err != nil {
		return nil, err
	}
	return build(merged), nil
}

func loadEmbedded() ([]Compiled, error) {
	entries, err := fs.ReadDir(embedded, "data")
	if err != nil {
		return nil, err
	}
	var merged []Compiled
	for _, e := range entries {
		data, err := embedded.ReadFile(path.Join("data", e.Name()))
		if err != nil {
			return nil, err
		}
		if merged, err = mergeFile(merged, e.Name(), data); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func loadDir(merged []Compiled, dir string) ([]Compiled, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if merged, err = mergeFile(merged, name, data); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func mergeFile(into []Compiled, name string, data []byte) ([]Compiled, error) {
	var f file
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	for _, r := range f.Rules {
		r.Attribution = f.Attribution
		c, err := compile(r)
		if err != nil {
			return nil, fmt.Errorf("%s: rule %s: %w", name, r.ID, err)
		}
		i := slices.IndexFunc(into, func(e Compiled) bool { return e.ID == c.ID })
		if i < 0 {
			into = append(into, c)
			continue
		}
		into[i] = c
	}
	return into, nil
}

func apply(rules []Compiled, byLayer map[string]map[string]config.RuleSetting) ([]Compiled, error) {
	for i, c := range rules {
		settings, ok := byLayer[c.Layer]
		if !ok {
			return nil, fmt.Errorf("rule %s: unknown layer %q", c.ID, c.Layer)
		}
		if s, ok := settings[c.ID]; ok {
			rules[i].Rule = override(c.Rule, s)
		}
	}
	return rules, unknownSettings(rules, byLayer)
}

func unknownSettings(rules []Compiled, byLayer map[string]map[string]config.RuleSetting) error {
	for _, layer := range slices.Sorted(maps.Keys(byLayer)) {
		for _, id := range slices.Sorted(maps.Keys(byLayer[layer])) {
			known := slices.ContainsFunc(rules, func(c Compiled) bool { return c.ID == id && c.Layer == layer })
			if !known {
				return fmt.Errorf("layers.%s.rules.%s: no such rule in that layer", layer, id)
			}
		}
	}
	return nil
}

func override(r Rule, s config.RuleSetting) Rule {
	if s.Category != nil {
		r.Category = *s.Category
	}
	if s.Severity != nil {
		r.Severity = *s.Severity
	}
	if s.Enabled != nil {
		r.Enabled = *s.Enabled
	}
	if s.Interrupt != nil {
		r.Interrupt = *s.Interrupt
	}
	if s.Judge == nil {
		return r
	}
	if s.Judge.Downgrade != nil {
		r.Judge.Downgrade = *s.Judge.Downgrade
	}
	if s.Judge.DowngradeFloor != nil {
		r.Judge.DowngradeFloor = *s.Judge.DowngradeFloor
	}
	if s.Judge.Instructions != nil {
		r.Judge.Instructions = *s.Judge.Instructions
	}
	return r
}

func build(rules []Compiled) *Set {
	s := &Set{ByID: make(map[string]Rule, len(rules))}
	for _, c := range rules {
		s.ByID[c.ID] = c.Rule
		if c.Enabled && c.Layer == config.StaticLayer {
			s.Static = append(s.Static, c)
		}
	}
	return s
}
