// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
)

type Scope string

const (
	Any   Scope = "any"
	Code  Scope = "code"
	Prose Scope = "prose"
	Link  Scope = "link"
)

type JudgePolicy struct {
	Downgrade      bool             `json:"downgrade"`
	DowngradeFloor finding.Severity `json:"downgradeFloor,omitempty"`
}

type Installer struct {
	Files    []string `json:"files"`
	Patterns []string `json:"patterns"`
}

type Rule struct {
	ID          string           `json:"id"`
	Category    string           `json:"category"`
	Severity    finding.Severity `json:"severity"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Kind        finding.Kind     `json:"kind"`
	Layer       string           `json:"layer"`
	Scope       Scope            `json:"scope,omitempty"`
	Files       []string         `json:"files,omitempty"`
	Patterns    []string         `json:"patterns,omitempty"`
	Exclude     []string         `json:"exclude,omitempty"`
	ReadOnly    []string         `json:"readOnly,omitempty"`
	Installers  []Installer      `json:"installers,omitempty"`
	Judge       JudgePolicy      `json:"judge"`
	Usages      []string         `json:"usages,omitempty"`
	Decision    *Decision        `json:"decision,omitempty"`
	Fields      map[string]Field `json:"fields,omitempty"`
	Enabled     bool             `json:"enabled"`
	Interrupt   bool             `json:"interrupt"`
	Attribution json.RawMessage  `json:"-"`
}

func (r Rule) Finding(loc finding.Location, evidence string) finding.Finding {
	return finding.Finding{
		Rule:     r.ID,
		Severity: r.Severity,
		Location: loc,
		Evidence: evidence,
	}
}

func (r Rule) validate() error {
	if r.ID == "" {
		return errors.New("missing id")
	}
	if r.Layer == "" {
		return errors.New("missing layer")
	}
	if r.Category == "" {
		return errors.New("missing category")
	}
	if _, err := finding.ParseSeverity(string(r.Severity)); err != nil {
		return err
	}
	if r.Judge.DowngradeFloor != "" {
		if _, err := finding.ParseSeverity(string(r.Judge.DowngradeFloor)); err != nil {
			return fmt.Errorf("floor: %w", err)
		}
	}
	switch r.Scope {
	case Any, Code, Prose, Link, "":
	default:
		return fmt.Errorf("unknown scope %q", r.Scope)
	}
	if err := r.validateGlobs(); err != nil {
		return err
	}
	if err := r.Decision.validate(r.Severity); err != nil {
		return err
	}
	if err := r.validateFields(); err != nil {
		return err
	}
	return r.validateKind()
}

func (r Rule) validateGlobs() error {
	if err := config.ValidGlobs("files", r.Files); err != nil {
		return err
	}
	for _, in := range r.Installers {
		if err := config.ValidGlobs("installers.files", in.Files); err != nil {
			return err
		}
	}
	return nil
}

func (r Rule) validateKind() error {
	switch r.Kind {
	case finding.KindHeuristic:
		if len(r.Patterns) == 0 && r.Layer == config.StaticLayer {
			return errors.New("heuristic rule without patterns")
		}
	case finding.KindLLM:
		if len(r.Patterns) > 0 || len(r.Exclude) > 0 {
			return errors.New("llm rule with patterns")
		}
	case finding.KindFact:
		if len(r.Patterns) > 0 || len(r.Exclude) > 0 {
			return errors.New("fact rule with patterns")
		}
	default:
		return fmt.Errorf("unknown kind %q", r.Kind)
	}
	return nil
}

type Compiled struct {
	Rule
	Patterns   []*regexp.Regexp
	Exclude    []*regexp.Regexp
	ReadOnly   []*regexp.Regexp
	Installers []CompiledInstaller
}

type CompiledInstaller struct {
	Files    []string
	Patterns []*regexp.Regexp
}

func compile(r Rule) (Compiled, error) {
	if err := r.prepare(); err != nil {
		return Compiled{}, err
	}
	if r.Scope == "" {
		r.Scope = Any
	}
	c := Compiled{Rule: r}
	var err error
	if c.Patterns, err = expressions("pattern", r.Patterns, "%s"); err != nil {
		return Compiled{}, err
	}
	if c.Exclude, err = expressions("exclude", r.Exclude, "%s"); err != nil {
		return Compiled{}, err
	}
	if c.ReadOnly, err = expressions("readOnly", r.ReadOnly, "^(?:%s)$"); err != nil {
		return Compiled{}, err
	}
	for _, in := range r.Installers {
		patterns, err := expressions("installers", in.Patterns, "%s")
		if err != nil {
			return Compiled{}, err
		}
		c.Installers = append(c.Installers, CompiledInstaller{Files: in.Files, Patterns: patterns})
	}
	return c, nil
}

func expressions(field string, sources []string, form string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	for _, p := range sources {
		re, err := regexp.Compile(fmt.Sprintf(form, p))
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", field, p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func (r *Rule) prepare() error {
	decision, err := r.decision()
	if err != nil {
		return err
	}
	r.Decision = decision
	return r.validate()
}

type Set struct {
	Static []Compiled
	ByID   map[string]Rule
}

func (s *Set) ForLayer(layer string) map[string]bool {
	allowed := make(map[string]bool)
	for id, r := range s.ByID {
		if r.Enabled && r.Layer == layer {
			allowed[id] = true
		}
	}
	return allowed
}
