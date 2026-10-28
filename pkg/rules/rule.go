// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"

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
	Instructions   string           `json:"instructions"`
}

var Usages = []string{"citation", "documentation", "user-action", "tool-server", "agent-fetch", "agent-instructions", "agent-install", "agent-upload"}

type Subject string

const (
	EnvRead Subject = "env-read"
	Request Subject = "request"
	Lines   Subject = "lines"
	Places  Subject = "places"
)

type DecisionPolicy struct {
	Subject        Subject           `json:"subject"`
	Question       string            `json:"question"`
	Options        map[string]string `json:"options"`
	Dismiss        []string          `json:"dismiss"`
	Confirm        []string          `json:"confirm,omitempty"`
	MinProbability float64           `json:"minProbability,omitempty"`
}

func (d *DecisionPolicy) validate() error {
	if d == nil {
		return nil
	}
	if d.Subject != EnvRead && d.Subject != Request && d.Subject != Lines && d.Subject != Places {
		return fmt.Errorf("decision.subject: unknown subject %q", d.Subject)
	}
	if d.Question == "" || len(d.Options) < 2 {
		return errors.New("decision.options: a decision needs a question and two options")
	}
	if d.MinProbability < 0 || d.MinProbability > 1 {
		return errors.New("decision.minProbability: must be between 0 and 1")
	}
	if err := d.known("dismiss", d.Dismiss); err != nil {
		return err
	}
	if err := d.known("confirm", d.Confirm); err != nil {
		return err
	}
	for _, option := range d.Confirm {
		if slices.Contains(d.Dismiss, option) {
			return fmt.Errorf("decision.confirm: option %q is also in dismiss", option)
		}
	}
	return nil
}

func (d *DecisionPolicy) known(field string, options []string) error {
	for _, option := range options {
		if _, ok := d.Options[option]; !ok {
			return fmt.Errorf("decision.%s: unknown option %q", field, option)
		}
	}
	return nil
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
	Judge       JudgePolicy      `json:"judge"`
	Usages      []string         `json:"usages,omitempty"`
	Decision    *DecisionPolicy  `json:"decision,omitempty"`
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
	if err := config.ValidGlobs("files", r.Files); err != nil {
		return err
	}
	if err := r.Decision.validate(); err != nil {
		return err
	}
	for _, u := range r.Usages {
		if !slices.Contains(Usages, u) {
			return fmt.Errorf("usages: unknown usage %q", u)
		}
	}
	return r.validateKind()
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
	Patterns []*regexp.Regexp
	Exclude  []*regexp.Regexp
}

func compile(r Rule) (Compiled, error) {
	if err := r.validate(); err != nil {
		return Compiled{}, err
	}
	if r.Scope == "" {
		r.Scope = Any
	}
	c := Compiled{Rule: r}
	for _, p := range r.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return Compiled{}, fmt.Errorf("pattern %q: %w", p, err)
		}
		c.Patterns = append(c.Patterns, re)
	}
	for _, p := range r.Exclude {
		re, err := regexp.Compile(p)
		if err != nil {
			return Compiled{}, fmt.Errorf("exclude %q: %w", p, err)
		}
		c.Exclude = append(c.Exclude, re)
	}
	return c, nil
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
