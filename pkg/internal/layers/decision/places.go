// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const notAsked = "the question was not asked at this place"

var (
	extras    = regexp.MustCompile(`\[[^\]"'\s]*\]`)
	nameToken = regexp.MustCompile(`[@\w][\w@./:+-]*`)
)

type spotted struct {
	rule string
	file string
	line int
}

func at(f finding.Finding) spotted {
	return spotted{f.Rule, f.Location.File, f.Location.Line}
}

type place struct {
	id        string
	text      string
	locations []int
	replies   map[string]reply
	request   int
}

type groupQuestion struct {
	id     string
	origin string
	named  named
}

type group struct {
	subject   string
	state     func(places string) string
	questions []groupQuestion
	places    []*place
	guards    []reply
	byFinding map[spotted]*place
}

func (g *group) want(n named) string {
	for _, q := range g.questions {
		if q.origin == n.name && same(q.named.question, n.question) {
			return q.id
		}
	}
	id := n.name
	if slices.ContainsFunc(g.questions, func(q groupQuestion) bool { return q.id == id }) {
		id = fmt.Sprintf("%s-%d", n.name, len(g.questions)+1)
	}
	g.questions = append(g.questions, groupQuestion{id: id, origin: n.name, named: named{name: id, question: n.question}})
	return id
}

func same(a, b rules.Question) bool {
	return a.Question == b.Question && a.Focus == b.Focus && maps.Equal(a.Options, b.Options)
}

func (g *group) asked() []named {
	out := make([]named, 0, len(g.questions))
	for _, q := range g.questions {
		out = append(out, q.named)
	}
	return out
}

func (r *run) places(ctx context.Context, rule rules.Rule, f finding.Finding, questions []named, e *evidence) {
	var g *group
	var places []*place
	if rule.Layer == config.ReferencesLayer {
		if g = r.refGroups[f.Reference]; g != nil {
			places = g.places
		}
	} else {
		g = r.ruleGroup(ctx, rule, f, questions)
		if p := g.byFinding[at(f)]; p != nil {
			places = []*place{p}
		}
	}
	if len(places) == 0 {
		e.add(rules.Places, instance{replies: skip(questions, noLine)})
		return
	}
	requests := make(map[int]bool)
	for _, p := range places {
		in := instance{label: p.id, replies: make(map[string]reply, len(questions))}
		for _, n := range questions {
			in.replies[n.name] = p.reply(g.want(n))
		}
		e.add(rules.Places, in)
		if !requests[p.request] {
			requests[p.request] = true
			e.guards = append(e.guards, g.guards[p.request])
		}
	}
}

func (p *place) reply(id string) reply {
	if got, ok := p.replies[id]; ok {
		return got
	}
	return reply{skipped: notAsked}
}

func (r *run) ruleGroup(ctx context.Context, rule rules.Rule, f finding.Finding, questions []named) *group {
	name := target(rule, f)
	key := rule.ID + "\x00" + name
	if g, ok := r.ruleGroups[key]; ok {
		return g
	}
	g := &group{subject: name, byFinding: make(map[spotted]*place)}
	g.state = func(places string) string {
		return llmcall.Render(files, "places.md", map[string]string{"subject": name, "places": places})
	}
	for _, n := range questions {
		g.want(n)
	}
	for _, other := range r.state.Findings {
		if other.Rule != rule.ID || other.Decision != nil || other.Judgement != nil || target(rule, other) != name {
			continue
		}
		text, ok := r.frame(other.Location, anchor(other))
		if !ok {
			continue
		}
		p := &place{id: fmt.Sprintf("P%d", len(g.places)+1)}
		p.text = llmcall.Render(files, "place.md", map[string]string{"id": p.id, "frame": text, "trace": ""})
		g.places = append(g.places, p)
		g.byFinding[at(other)] = p
	}
	r.askAll(ctx, g)
	r.ruleGroups[key] = g
	return g
}

func target(rule rules.Rule, f finding.Finding) string {
	if rule.Kind == finding.KindLLM {
		return rule.Title
	}
	fields := strings.Fields(extras.ReplaceAllString(f.Evidence, ""))
	if len(fields) == 0 {
		return ""
	}
	tokens := nameToken.FindAllString(fields[len(fields)-1], -1)
	if len(tokens) == 0 {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(tokens[len(tokens)-1]), "@latest")
}

func (r *run) askAll(ctx context.Context, g *group) {
	cfg := r.layer.Config
	fixed := len(g.state("")) + chars(guard().question)
	var batch []*place
	text, total := fixed, fixed
	for _, p := range g.places {
		cost := 0
		for _, q := range queries(g.asked(), g.subject, p.id) {
			cost += chars(q.question)
		}
		over := text+len(p.text) > cfg.MaxStateChars || total+len(p.text)+cost > cfg.MaxRequestChars
		if over && len(batch) > 0 {
			r.askPlaces(ctx, g, batch)
			batch, text, total = nil, fixed, fixed
		}
		batch = append(batch, p)
		text += len(p.text) + 1
		total += len(p.text) + 1 + cost
	}
	if len(batch) > 0 {
		r.askPlaces(ctx, g, batch)
	}
}

func (r *run) askPlaces(ctx context.Context, g *group, batch []*place) {
	texts := make([]string, 0, len(batch))
	var qs []query
	for _, p := range batch {
		texts = append(texts, strings.TrimRight(p.text, "\n"))
		qs = append(qs, queries(g.asked(), g.subject, p.id)...)
	}
	replies, guarded := r.ask(ctx, g.state(strings.Join(texts, "\n\n")), qs)
	for _, p := range batch {
		p.replies = make(map[string]reply, len(g.questions))
		for _, q := range g.questions {
			p.replies[q.id] = replies[p.id+"."+q.id]
		}
		p.request = len(g.guards)
	}
	g.guards = append(g.guards, guarded)
}
