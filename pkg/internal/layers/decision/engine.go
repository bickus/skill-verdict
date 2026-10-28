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
	yes       = "yes"
	confident = 3
	certain   = 4
	certainty = 0.95
)

type limits struct {
	confirm float64
	dismiss float64
	guard   float64
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
	guards    []reply
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

func judge(rule rules.Rule, f finding.Finding, e evidence, t limits) *verdict {
	d := rule.Decision
	warned, guard := e.guarded(t.guard)
	if m, ok := e.first(d, d.Outcome.Confirm, t.confirm, false); ok {
		return &verdict{state: finding.Confirmed, severity: f.Severity, confidence: m.confidence(), reason: join(m.words(), warned)}
	}
	if warned != "" {
		return &verdict{state: finding.Upheld, severity: f.Severity, confidence: level(guard), reason: warned}
	}
	if !rule.Judge.Downgrade {
		return nil
	}
	if v, ok := e.lowered(d, rule.Judge.DowngradeFloor, f.Severity, t.dismiss); ok {
		return v
	}
	return e.dismissed(d, rule.Judge.DowngradeFloor, f.Severity, t.dismiss)
}

func (e evidence) guarded(bar float64) (string, float64) {
	best := 0.0
	for _, g := range e.guards {
		best = max(best, g.probabilities[yes])
	}
	if best < bar {
		return "", 0
	}
	return fmt.Sprintf("%s: %s (%.2f), the text speaks to a reviewer", rules.Guard, yes, best), best
}

func (e evidence) lowered(d *rules.Decision, floor, current finding.Severity, bar float64) (*verdict, bool) {
	for _, severity := range finding.Severities() {
		m, ok := e.first(d, d.Outcome.Severity[severity], bar, true)
		if !ok {
			continue
		}
		target := severity
		if floor.Rank() > target.Rank() {
			target = floor
		}
		if target.Rank() >= current.Rank() {
			return nil, true
		}
		return &verdict{state: finding.Downgraded, severity: target, confidence: m.confidence(), reason: join(m.words(), "")}, true
	}
	return nil, false
}

func (e evidence) dismissed(d *rules.Decision, floor, current finding.Severity, bar float64) *verdict {
	var first matched
	for _, row := range d.Outcome.Dismiss {
		m, ok := e.match(d, row, bar, true)
		if !ok {
			continue
		}
		if alone, misfire := misfired(row); misfire {
			if n, hit := e.match(d, alone, bar, true); hit {
				return &verdict{state: finding.Dismissed, severity: current, confidence: n.confidence(), reason: join(n.words(), "")}
			}
		}
		if first == nil {
			first = m
		}
	}
	switch {
	case first == nil:
		return nil
	case floor == "":
		return &verdict{state: finding.Dismissed, severity: current, confidence: first.confidence(), reason: join(first.words(), "")}
	case floor.Rank() < current.Rank():
		return &verdict{state: finding.Downgraded, severity: floor, confidence: first.confidence(), reason: join(first.words(), "")}
	}
	return nil
}

func misfired(row rules.Row) (rules.Row, bool) {
	if len(row) != 1 {
		return nil, false
	}
	for question, options := range row {
		if slices.Contains(options, rules.NotAMatch) {
			return rules.Row{question: {rules.NotAMatch}}, true
		}
	}
	return nil, false
}

func (e evidence) first(d *rules.Decision, rows []rules.Row, bar float64, every bool) (matched, bool) {
	for _, row := range rows {
		if m, ok := e.match(d, row, bar, every); ok {
			return m, true
		}
	}
	return nil, false
}

func (e evidence) match(d *rules.Decision, row rules.Row, bar float64, every bool) (matched, bool) {
	bySubject := make(map[rules.Subject][]string)
	for _, question := range slices.Sorted(maps.Keys(row)) {
		subject := d.Questions[question].Subject
		bySubject[subject] = append(bySubject[subject], question)
	}
	out := matched{}
	for subject, questions := range bySubject {
		m, ok := holds(e.instances[subject], questions, row, bar, every)
		if !ok {
			return nil, false
		}
		maps.Copy(out, m)
	}
	return out, true
}

func holds(instances []instance, questions []string, row rules.Row, bar float64, every bool) (matched, bool) {
	out := matched{}
	for _, in := range instances {
		m, ok := in.holds(questions, row, bar, len(instances) > 1 && !every)
		switch {
		case ok && !every:
			return m, true
		case !ok && every:
			return nil, false
		case ok:
			out.lowest(m)
		}
	}
	return out, every && len(instances) > 0
}

func (in instance) holds(questions []string, row rules.Row, bar float64, placed bool) (matched, bool) {
	out := make(matched, len(questions))
	for _, question := range questions {
		p := in.replies[question]
		mass := p.mass(row[question])
		if mass < bar {
			return nil, false
		}
		text := p.words(question, row[question], mass)
		if placed && in.label != "" {
			text += " at " + in.label
		}
		out[question] = term{text: text, mass: mass}
	}
	return out, true
}

func (m matched) lowest(o matched) {
	for question, t := range o {
		if have, ok := m[question]; !ok || t.mass < have.mass {
			m[question] = t
		}
	}
}

func (m matched) words() []string {
	out := make([]string, 0, len(m))
	for _, question := range slices.Sorted(maps.Keys(m)) {
		out = append(out, m[question].text)
	}
	return out
}

func (m matched) confidence() int {
	lowest := 1.0
	for _, t := range m {
		lowest = min(lowest, t.mass)
	}
	return level(lowest)
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

func join(words []string, warned string) string {
	if warned != "" {
		words = append(words, warned)
	}
	return strings.Join(words, ", ")
}

func level(probability float64) int {
	if probability >= certainty {
		return certain
	}
	return confident
}
