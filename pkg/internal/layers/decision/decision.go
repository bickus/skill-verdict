// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/internal/trace"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name       = "decision"
	questionID = "trace"
	confident  = 3
	certain    = 4
	certainty  = 0.95
)

type Layer struct {
	Client *decisions.Client
	Config config.DecisionConfig
	Late   bool
}

func (Layer) Name() string {
	return name
}

type asked struct {
	rule    string
	subject *trace.Subject
}

type outcome struct {
	pass        bool
	confirmed   bool
	choices     []string
	probability float64
	dismiss     float64
	confirm     float64
	skipped     string
}

type run struct {
	layer  Layer
	state  *pipeline.State
	report pipeline.Report
	done   map[asked]outcome
	placed map[spotted]outcome
	self   refcheck.SelfInstall
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	r := &run{layer: l, state: s, done: make(map[asked]outcome), placed: make(map[spotted]outcome), self: refcheck.NewSelfInstall(s.Skill.Folder())}
	if err := r.findings(ctx); err != nil || l.Late {
		return r.report, err
	}
	if err := r.usages(ctx); err != nil {
		return r.report, err
	}
	return r.report, r.findings(ctx)
}

func (r *run) findings(ctx context.Context) error {
	s := r.state
	for i := range s.Findings {
		f := &s.Findings[i]
		rule := s.Rules.ByID[f.Rule]
		if f.Judgement != nil || f.Decision != nil || rule.Decision == nil || len(rule.Decision.Dismiss) == 0 {
			continue
		}
		total, err := r.answer(ctx, rule, *f)
		if err != nil {
			return err
		}
		f.Decision = &finding.Decision{Choice: strings.Join(total.choices, ", "), Probability: total.probability, Dismiss: total.dismiss, Skipped: total.skipped}
		switch {
		case total.pass:
			f.Judgement = &finding.Judgement{State: finding.Dismissed, From: f.Severity, Confidence: level(total.dismiss), Reason: reason(rule.Decision.Subject, total)}
		case total.confirmed && !r.selfLine(*f):
			f.Judgement = &finding.Judgement{State: finding.Confirmed, From: f.Severity, Confidence: level(total.confirm), Reason: reason(rule.Decision.Subject, total)}
		}
	}
	return nil
}

func (r *run) selfLine(f finding.Finding) bool {
	return len(r.self.Spans(refcheck.Line(r.state.Skill, f.Location))) > 0
}

func (r *run) answer(ctx context.Context, rule rules.Rule, f finding.Finding) (outcome, error) {
	switch rule.Decision.Subject {
	case rules.Lines:
		return r.lines(ctx, rule, f)
	case rules.Places:
		return r.places(ctx, rule, f)
	case rules.EnvRead, rules.Request:
	}
	subjects := r.state.Linked(f)
	if len(subjects) == 0 {
		return r.lines(ctx, rule, f)
	}
	return r.all(ctx, rule, subjects)
}

func (r *run) all(ctx context.Context, rule rules.Rule, subjects []*trace.Subject) (outcome, error) {
	total := outcome{pass: true, probability: 1, dismiss: 1}
	for _, sub := range subjects {
		key := asked{rule.ID, sub}
		o, seen := r.done[key]
		if !seen {
			var err error
			if o, err = r.decide(ctx, rule.Decision, r.state.Code.Trace(sub)); err != nil {
				return outcome{}, err
			}
			r.done[key] = o
		}
		if !o.pass {
			return o, nil
		}
		total = merge(total, o)
	}
	return total, nil
}

func merge(a, b outcome) outcome {
	a.probability = min(a.probability, b.probability)
	a.dismiss = min(a.dismiss, b.dismiss)
	for _, c := range b.choices {
		if !slices.Contains(a.choices, c) {
			a.choices = append(a.choices, c)
		}
	}
	return a
}

func (r *run) decide(ctx context.Context, d *rules.DecisionPolicy, t *trace.Trace) (outcome, error) {
	state, err := t.State(r.where(t.Subject))
	if err != nil {
		return outcome{}, err
	}
	if len(t.Calls) == 0 {
		return outcome{skipped: "the trace has no call that takes the value"}, nil
	}
	return r.send(ctx, d, questionID, state, question(d, t.Subject.Match))
}

func (r *run) send(ctx context.Context, d *rules.DecisionPolicy, id string, state json.RawMessage, asked decisions.Question) (outcome, error) {
	cfg := r.layer.Config
	if len(state) > cfg.MaxStateChars || len(state)+chars(asked) > cfg.MaxRequestChars {
		return outcome{skipped: oversized}, nil
	}
	answers, usage, err := r.layer.Client.Decide(ctx, state, map[string]decisions.Question{id: asked})
	r.report.Requests++
	r.report.Usage = r.report.Usage.Add(usage)
	if err != nil {
		return outcome{}, err
	}
	return r.judge(d, answers[id]), nil
}

func (r *run) judge(d *rules.DecisionPolicy, a decisions.Answer) outcome {
	bar := max(d.MinProbability, r.layer.Config.MinProbability)
	dismiss, confirm := sum(a, d.Dismiss), sum(a, d.Confirm)
	return outcome{
		pass:        slices.Contains(d.Dismiss, a.Choice) && dismiss >= bar,
		confirmed:   slices.Contains(d.Confirm, a.Choice) && confirm >= bar,
		choices:     []string{a.Choice},
		probability: a.Probabilities[a.Choice],
		dismiss:     dismiss,
		confirm:     confirm,
	}
}

func sum(a decisions.Answer, options []string) float64 {
	total := 0.0
	for _, option := range options {
		total += a.Probabilities[option]
	}
	return total
}

func (r *run) where(s *trace.Subject) string {
	a := r.state.Skill.Artifact(s.Unit.Path)
	if a == nil {
		return ""
	}
	line := textutil.LineAt(textutil.LineStarts(a.Content), s.Span.Start)
	return spot(r.state, a, textutil.Lines(a.Content), line)
}

func level(probability float64) int {
	if probability >= certainty {
		return certain
	}
	return confident
}

func reason(subject rules.Subject, o outcome) string {
	slices.Sort(o.choices)
	read := "the trace of the code"
	if subject == rules.Lines || subject == rules.Places {
		read = "the lines around the finding"
	}
	if o.confirmed {
		return fmt.Sprintf("The decision model read %s and answered %s, probability of a confirming answer %.2f.", read, strings.Join(o.choices, ", "), o.confirm)
	}
	return fmt.Sprintf("The decision model read %s and answered %s, lowest probability of a dismissing answer %.2f.", read, strings.Join(o.choices, ", "), o.dismiss)
}
