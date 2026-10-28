// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"maps"
	"slices"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const name = "decision"

type Layer struct {
	Client *decisions.Client
	Config config.DecisionConfig
	Late   bool
}

func (Layer) Name() string {
	return name
}

type run struct {
	layer      Layer
	state      *pipeline.State
	report     pipeline.Report
	err        error
	ruleGroups map[string]*group
	refGroups  map[string]*group
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	r := &run{layer: l, state: s, ruleGroups: make(map[string]*group), refGroups: make(map[string]*group)}
	if !l.Late {
		r.usages(ctx)
	}
	r.findings(ctx)
	return r.report, r.err
}

func (r *run) findings(ctx context.Context) {
	s := r.state
	for i := range s.Findings {
		f := &s.Findings[i]
		rule := s.Rules.ByID[f.Rule]
		if f.Judgement != nil || f.Decision != nil || rule.Decision == nil {
			continue
		}
		e := r.evidence(ctx, rule, *f)
		f.Decision = e.record()
		if v := judge(rule, *f, e, r.limits()); v != nil {
			f.Judgement = &finding.Judgement{State: v.state, From: f.Severity, Confidence: v.confidence, Reason: v.reason}
			f.Severity = v.severity
		}
	}
}

func (r *run) limits() limits {
	c := r.layer.Config
	return limits{confirm: c.ConfirmProbability, dismiss: c.DismissProbability}
}

func (r *run) evidence(ctx context.Context, rule rules.Rule, f finding.Finding) evidence {
	e := evidence{instances: make(map[rules.Subject][]instance)}
	for _, subject := range subjects(rule.Decision) {
		questions := asked(rule.Decision, subject)
		switch subject {
		case rules.Places:
			r.places(ctx, rule, f, questions, &e)
		case rules.Trace:
			r.traced(ctx, f, questions, &e)
		case rules.Frame, rules.Entry, rules.Facts:
			state, skipped := r.stateOf(subject, f)
			r.single(ctx, subject, state, skipped, f.Evidence, questions, &e)
		}
	}
	return e
}

func (r *run) single(ctx context.Context, subject rules.Subject, state, skipped, about string, questions []named, e *evidence) {
	if skipped != "" {
		e.add(subject, instance{replies: skip(questions, skipped)})
		return
	}
	e.add(subject, instance{replies: r.ask(ctx, state, queries(questions, about, ""))})
}

func (r *run) traced(ctx context.Context, f finding.Finding, questions []named, e *evidence) {
	linked := r.state.Linked(f)
	if len(linked) == 0 {
		state, skipped := r.stateOf(rules.Frame, f)
		r.single(ctx, rules.Trace, state, skipped, f.Evidence, questions, e)
		return
	}
	for i, sub := range linked {
		state, skipped := r.traceState(sub)
		before := len(e.instances[rules.Trace])
		r.single(ctx, rules.Trace, state, skipped, sub.Match, questions, e)
		if len(linked) > 1 {
			e.instances[rules.Trace][before].label = traceLabel(i)
		}
	}
}

func subjects(d *rules.Decision) []rules.Subject {
	var out []rules.Subject
	for _, q := range d.Questions {
		if !slices.Contains(out, q.Subject) {
			out = append(out, q.Subject)
		}
	}
	slices.Sort(out)
	return out
}

func asked(d *rules.Decision, subject rules.Subject) []named {
	var out []named
	for _, n := range slices.Sorted(maps.Keys(d.Questions)) {
		if q := d.Questions[n]; q.Subject == subject {
			out = append(out, named{name: n, question: q})
		}
	}
	return out
}

func (e *evidence) add(subject rules.Subject, in instance) {
	e.instances[subject] = append(e.instances[subject], in)
}

func (e evidence) record() *finding.Decision {
	d := &finding.Decision{}
	for _, subject := range slices.Sorted(maps.Keys(e.instances)) {
		for _, in := range e.instances[subject] {
			for _, q := range slices.Sorted(maps.Keys(in.replies)) {
				d.Answers = append(d.Answers, in.replies[q].answer(q, in.label))
			}
		}
	}
	return d
}
