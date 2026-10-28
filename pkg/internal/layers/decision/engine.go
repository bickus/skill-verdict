// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	confident = 3
	certain   = 4
	certainty = 0.95
)

type limits struct {
	confirm float64
	dismiss float64
}

type named struct {
	name     string
	question rules.Question
}

type reply struct {
	choice        string
	probabilities map[string]float64
	skipped       string
}

type instance struct {
	label   string
	replies map[string]reply
}

type evidence struct {
	instances map[rules.Subject][]instance
}

type verdict struct {
	state      finding.State
	severity   finding.Severity
	confidence int
	reason     string
}

type term struct {
	text string
	mass float64
}

type matched map[string]term

type result int

const (
	misfired result = iota
	dismissed
	lowered
	kept
	confirmed
)

type outcome struct {
	result   result
	severity finding.Severity
	match    matched
}

type given struct {
	reply
	label string
}

type combination map[string]given

func judge(rule rules.Rule, f finding.Finding, e evidence, t limits) *verdict {
	o := e.decide(rule.Decision.Outcome, t, rule.Judge.Downgrade)
	floor := rule.Judge.DowngradeFloor
	switch o.result {
	case confirmed:
		return o.verdict(finding.Confirmed, f.Severity)
	case misfired:
		return o.verdict(finding.Dismissed, f.Severity)
	case lowered:
		return o.lower(higher(o.severity, floor), f.Severity)
	case dismissed:
		if floor == "" {
			return o.verdict(finding.Dismissed, f.Severity)
		}
		return o.lower(floor, f.Severity)
	case kept:
	}
	return nil
}

func higher(a, b finding.Severity) finding.Severity {
	if b.Rank() > a.Rank() {
		return b
	}
	return a
}

func (o outcome) lower(target, current finding.Severity) *verdict {
	if target.Rank() >= current.Rank() {
		return nil
	}
	return o.verdict(finding.Downgraded, target)
}

func (o outcome) verdict(state finding.State, severity finding.Severity) *verdict {
	return &verdict{state: state, severity: severity, confidence: level(o.match.weakest()), reason: strings.Join(o.match.words(), ", ")}
}

func (e evidence) decide(table rules.Outcome, t limits, lowering bool) outcome {
	all := e.combinations()
	if len(all) == 0 {
		return outcome{result: kept}
	}
	worst := all[0].decide(table, t, lowering)
	for _, c := range all[1:] {
		o := c.decide(table, t, lowering)
		tied := o.rank() == worst.rank() && o.result != confirmed && o.match.weakest() < worst.match.weakest()
		if o.rank() > worst.rank() || tied {
			worst = o
		}
	}
	return worst
}

func (o outcome) rank() int {
	switch o.result {
	case lowered:
		return int(dismissed) + o.severity.Rank()
	case kept, confirmed:
		return int(o.result) + finding.Critical.Rank()
	case misfired, dismissed:
	}
	return int(o.result)
}

func (e evidence) combinations() []combination {
	all := []combination{{}}
	for _, subject := range slices.Sorted(maps.Keys(e.instances)) {
		instances := e.instances[subject]
		next := make([]combination, 0, len(all)*len(instances))
		for _, c := range all {
			for _, in := range instances {
				next = append(next, c.with(in, len(instances) > 1))
			}
		}
		all = next
	}
	return all
}

func (c combination) with(in instance, labelled bool) combination {
	out := make(combination, len(c)+len(in.replies))
	maps.Copy(out, c)
	for question, r := range in.replies {
		g := given{reply: r}
		if labelled {
			g.label = in.label
		}
		out[question] = g
	}
	return out
}

func (c combination) decide(table rules.Outcome, t limits, lowering bool) outcome {
	if m, ok := c.first(table.Misfire, t.dismiss); ok && lowering {
		return outcome{result: misfired, match: m}
	}
	if m, ok := c.first(table.Confirm, t.confirm); ok {
		return outcome{result: confirmed, match: m}
	}
	if !lowering {
		return outcome{result: kept}
	}
	for _, severity := range finding.Severities() {
		if m, ok := c.first(table.Severity[severity], t.dismiss); ok {
			return outcome{result: lowered, severity: severity, match: m}
		}
	}
	if m, ok := c.first(table.Dismiss, t.dismiss); ok {
		return outcome{result: dismissed, match: m}
	}
	return outcome{result: kept}
}

func (c combination) first(rows []rules.Row, bar float64) (matched, bool) {
	for _, row := range rows {
		if m, ok := c.match(row, bar); ok {
			return m, true
		}
	}
	return nil, false
}

func (c combination) match(row rules.Row, bar float64) (matched, bool) {
	out := make(matched, len(row))
	for question, options := range row {
		g := c[question]
		mass := g.mass(options)
		if mass < bar {
			return nil, false
		}
		text := g.words(question, options, mass)
		if g.label != "" {
			text += " at " + g.label
		}
		out[question] = term{text: text, mass: mass}
	}
	return out, true
}

func (m matched) words() []string {
	out := make([]string, 0, len(m))
	for _, question := range slices.Sorted(maps.Keys(m)) {
		out = append(out, m[question].text)
	}
	return out
}

func (m matched) weakest() float64 {
	lowest := 1.0
	for _, t := range m {
		lowest = min(lowest, t.mass)
	}
	return lowest
}

func (p reply) mass(options rules.Options) float64 {
	total := 0.0
	for _, option := range options {
		total += p.probabilities[option]
	}
	return total
}

func (p reply) words(question string, options rules.Options, mass float64) string {
	var hit []string
	for _, option := range options {
		if p.probabilities[option] > 0 {
			hit = append(hit, option)
		}
	}
	return fmt.Sprintf("%s: %s (%.2f)", question, strings.Join(hit, " or "), mass)
}

func (p reply) answer(question, place string) finding.Answer {
	return finding.Answer{Question: question, Place: place, Choice: p.choice, Probability: p.probabilities[p.choice], Skipped: p.skipped}
}

func level(probability float64) int {
	if probability >= certainty {
		return certain
	}
	return confident
}
