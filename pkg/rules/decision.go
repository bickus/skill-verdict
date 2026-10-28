// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
)

type Subject string

const (
	Frame  Subject = "frame"
	Places Subject = "places"
	Trace  Subject = "trace"
	Entry  Subject = "entry"
	Facts  Subject = "facts"
)

const (
	Usage    = "usage"
	FromCall = "call"
)

const (
	templatesFile = "questions.json"
	confirmKind   = "confirm"
	severityKind  = "severity"
	dismissKind   = "dismiss"
	misfireKind   = "misfire"
)

type Question struct {
	Subject  Subject           `json:"subject"`
	Template string            `json:"template,omitempty"`
	From     string            `json:"from,omitempty"`
	Question string            `json:"question,omitempty"`
	Focus    string            `json:"focus,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type Options []string

func (o *Options) UnmarshalJSON(data []byte) error {
	var one string
	if json.Unmarshal(data, &one) == nil {
		*o = Options{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return errors.New("an outcome row maps a question to one option or a list of options")
	}
	*o = many
	return nil
}

type Row map[string]Options

type Outcome struct {
	Confirm  []Row                      `json:"confirm,omitempty"`
	Severity map[finding.Severity][]Row `json:"severity,omitempty"`
	Dismiss  []Row                      `json:"dismiss,omitempty"`
	Misfire  []Row                      `json:"misfire,omitempty"`
}

type Decision struct {
	Questions map[string]Question `json:"questions"`
	Outcome   Outcome             `json:"outcome"`
}

var templates = readTemplates()

func readTemplates() map[string]Question {
	data, err := embedded.ReadFile(path.Join("data", templatesFile))
	if err != nil {
		panic(err)
	}
	var out map[string]Question
	if err = json.Unmarshal(data, &out); err != nil {
		panic(fmt.Errorf("%s: %w", templatesFile, err))
	}
	return out
}

func Template(name string) Question {
	q := templates[name]
	q.Template = name
	return q
}

func (d *Decision) TraceFrom() (string, bool) {
	if d == nil {
		return "", false
	}
	for _, name := range slices.Sorted(maps.Keys(d.Questions)) {
		if q := d.Questions[name]; q.Subject == Trace {
			return q.From, true
		}
	}
	return "", false
}

func (r Rule) decision() (*Decision, error) {
	d := r.Decision
	if len(r.Usages) > 0 {
		if d != nil {
			return nil, errors.New("usages and decision: a rule carries one or the other")
		}
		d = usages(r.Usages, r.Severity)
	}
	if d == nil {
		return nil, nil
	}
	resolved := &Decision{Questions: make(map[string]Question, len(d.Questions)), Outcome: d.Outcome}
	for name, q := range d.Questions {
		full, err := q.resolve()
		if err != nil {
			return nil, fmt.Errorf("decision.questions.%s.%w", name, err)
		}
		resolved.Questions[name] = full
	}
	return resolved, nil
}

func usages(confirm []string, severity finding.Severity) *Decision {
	outcome := Outcome{
		Confirm: []Row{{Usage: confirm}},
		Dismiss: []Row{{Usage: Options{"citation", "documentation"}}},
	}
	if finding.Medium.Rank() < severity.Rank() {
		outcome.Severity = map[finding.Severity][]Row{finding.Medium: {{Usage: Options{"user-action"}}}}
	}
	return &Decision{Questions: map[string]Question{Usage: {Subject: Places, Template: Usage}}, Outcome: outcome}
}

func (q Question) resolve() (Question, error) {
	if q.Template == "" {
		return q, nil
	}
	t, ok := templates[q.Template]
	switch {
	case !ok:
		return q, fmt.Errorf("template: unknown template %q", q.Template)
	case q.Question != "" || len(q.Options) > 0:
		return q, errors.New("template: a question names a template or carries its own question and options")
	}
	q.Question, q.Options, q.Focus = t.Question, t.Options, strings.TrimSpace(t.Focus+" "+q.Focus)
	return q, nil
}

func (d *Decision) validate(severity finding.Severity) error {
	if d == nil {
		return nil
	}
	if len(d.Questions) == 0 {
		return errors.New("decision.questions: a decision needs a question")
	}
	from, traced := "", false
	for _, name := range slices.Sorted(maps.Keys(d.Questions)) {
		q := d.Questions[name]
		if err := q.validate(); err != nil {
			return fmt.Errorf("decision.questions.%s.%w", name, err)
		}
		if q.Subject != Trace {
			continue
		}
		if traced && q.From != from {
			return fmt.Errorf("decision.questions.%s.from: every trace question of a rule starts at the same place", name)
		}
		from, traced = q.From, true
	}
	return d.Outcome.validate(d.Questions, severity)
}

func (q Question) validate() error {
	switch q.Subject {
	case Frame, Places, Trace, Entry, Facts:
	default:
		return fmt.Errorf("subject: unknown subject %q", q.Subject)
	}
	switch {
	case q.From != "" && (q.From != FromCall || q.Subject != Trace):
		return fmt.Errorf("from: only a trace question starts from a %q", FromCall)
	case len(q.Options) < 2:
		return errors.New("options: a question needs at least two options")
	}
	return nil
}

type kinded struct {
	kind string
	row  Row
}

func (o Outcome) validate(questions map[string]Question, severity finding.Severity) error {
	all, err := o.rows(severity)
	if err != nil {
		return err
	}
	for i, a := range all {
		if err = a.check(questions); err != nil {
			return fmt.Errorf("decision.outcome.%s: %w", a.kind, err)
		}
		for _, b := range all[i+1:] {
			if conflicting(a.kind, b.kind) && overlap(a.row, b.row) {
				return fmt.Errorf("decision.outcome: a %s row and a %s row match the same answers", a.kind, b.kind)
			}
		}
	}
	return nil
}

func conflicting(a, b string) bool {
	switch {
	case a == b:
		return false
	case a == confirmKind || b == confirmKind:
		return true
	}
	return a == severityKind || b == severityKind
}

func (o Outcome) rows(severity finding.Severity) ([]kinded, error) {
	all := make([]kinded, 0, len(o.Confirm)+len(o.Dismiss)+len(o.Misfire))
	for _, row := range o.Confirm {
		all = append(all, kinded{confirmKind, row})
	}
	for _, level := range slices.Sorted(maps.Keys(o.Severity)) {
		if level.Rank() == 0 || level.Rank() >= severity.Rank() {
			return nil, fmt.Errorf("decision.outcome.severity.%s: must be a severity below the rule severity %s", level, severity)
		}
		for _, row := range o.Severity[level] {
			all = append(all, kinded{severityKind, row})
		}
	}
	for _, row := range o.Dismiss {
		all = append(all, kinded{dismissKind, row})
	}
	for _, row := range o.Misfire {
		all = append(all, kinded{misfireKind, row})
	}
	return all, nil
}

func (k kinded) check(questions map[string]Question) error {
	if len(k.row) == 0 {
		return errors.New("a row names no question")
	}
	for _, name := range slices.Sorted(maps.Keys(k.row)) {
		q, known := questions[name]
		if !known {
			return fmt.Errorf("unknown question %q", name)
		}
		if err := named(q, k.row[name]); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func named(q Question, options Options) error {
	if len(options) == 0 {
		return errors.New("a row names at least one option")
	}
	for _, option := range options {
		if _, ok := q.Options[option]; !ok {
			return fmt.Errorf("unknown option %q", option)
		}
	}
	return nil
}

func overlap(a, b Row) bool {
	if len(a) != len(b) {
		return false
	}
	for name, options := range a {
		other, ok := b[name]
		if !ok || !slices.ContainsFunc(options, func(o string) bool { return slices.Contains(other, o) }) {
			return false
		}
	}
	return true
}
