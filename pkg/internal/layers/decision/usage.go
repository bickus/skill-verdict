// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	unknown    = "unknown"
	toolServer = "tool-server"
	untrusted  = "TOOL-SERVER-UNTRUSTED"
)

var networkKinds = []refs.Kind{refs.Domain, refs.Address}

var usageRules = map[string]string{
	"agent-instructions": "REMOTE-INSTRUCTIONS-FOLLOWED",
	"agent-install":      "REMOTE-CODE-AT-INSTALL",
	"agent-upload":       "REMOTE-ENDPOINT-UPLOAD-HIDDEN",
	toolServer:           "TOOL-SERVER-REMOTE-HIDDEN",
}

type placed struct {
	choice      string
	probability float64
}

func (r *run) usages(ctx context.Context) {
	s := r.state
	var raised []finding.Finding
	classified := false
	for i := range s.Refs {
		rep := &s.Refs[i]
		g := r.referenceGroup(ctx, rep)
		if g == nil || !refcheck.Categorized(*rep) {
			continue
		}
		answers := r.classify(rep, g)
		classified = true
		server := r.server(rep, answers)
		raised = append(raised, r.raise(rep, answers, len(server) > 0)...)
		raised = append(raised, server...)
	}
	s.Record(finding.Dedup(raised)...)
	if a := s.Skill.Artifact(refcheck.BriefPath); a != nil && classified {
		a.Content = refcheck.Brief(s.Skill, s.Refs)
		a.Size = int64(len(a.Content))
	}
}

func (r *run) referenceGroup(ctx context.Context, rep *refs.Report) *group {
	if rep.Ref.Derived || len(rep.Ref.Locations) == 0 {
		return nil
	}
	g := &group{subject: rep.Ref.Value, state: func(places string) string { return referenceState(rep, places) }}
	if refcheck.Categorized(*rep) {
		g.want(named{name: rules.Usage, question: rules.Template(rules.Usage)})
	}
	for _, f := range r.state.Findings {
		rule := r.state.Rules.ByID[f.Rule]
		if f.Reference != rep.Ref.Value || rule.Layer != config.ReferencesLayer || rule.Decision == nil || f.Judgement != nil || f.Decision != nil {
			continue
		}
		for _, n := range asked(rule.Decision, rules.Places) {
			g.want(n)
		}
	}
	if len(g.questions) == 0 {
		return nil
	}
	g.places = r.referencePlaces(rep)
	r.askAll(ctx, g)
	r.refGroups[rep.Ref.Value] = g
	return g
}

func referenceState(rep *refs.Report, places string) string {
	urls := make([]string, 0, len(rep.Ref.URLs))
	for _, u := range rep.Ref.URLs {
		urls = append(urls, "url: "+u)
	}
	return llmcall.Render(files, "reference.md", map[string]string{
		"reference": rep.Ref.Value,
		"urls":      strings.Join(urls, "\n"),
		"places":    places,
	})
}

func (r *run) classify(rep *refs.Report, g *group) []placed {
	usage := g.want(named{name: rules.Usage, question: rules.Template(rules.Usage)})
	options := rules.Template(rules.Usage).Options
	rep.Usages = make([]string, len(rep.Ref.Locations))
	answers := make([]placed, len(rep.Ref.Locations))
	for j := range rep.Ref.Locations {
		answers[j] = placed{choice: unknown}
		p := g.located(j)
		if p == nil {
			continue
		}
		got := p.reply(usage)
		if _, ok := options[got.choice]; !ok {
			continue
		}
		answers[j] = placed{choice: got.choice, probability: got.probabilities[got.choice]}
		if got.choice != unknown && answers[j].probability >= r.layer.Config.DismissProbability {
			rep.Usages[j] = got.choice
		}
	}
	return answers
}

func (g *group) located(j int) *place {
	for _, p := range g.places {
		if slices.Contains(p.locations, j) {
			return p
		}
	}
	return nil
}

func (r *run) referencePlaces(rep *refs.Report) []*place {
	byBody := make(map[string]*place)
	var out []*place
	for j, loc := range rep.Ref.Locations {
		frame, ok := r.frame(loc, rep.Ref.Value)
		if !ok {
			continue
		}
		trace := r.placeTrace(rep, loc)
		if p, seen := byBody[frame+"\n"+trace]; seen {
			p.locations = append(p.locations, j)
			continue
		}
		p := &place{id: rep.LocationID(j), locations: []int{j}}
		p.text = llmcall.Render(files, "place.md", map[string]string{"id": p.id, "frame": frame, "trace": trace})
		byBody[frame+"\n"+trace] = p
		out = append(out, p)
	}
	return out
}

func (r *run) placeTrace(rep *refs.Report, loc finding.Location) string {
	s := r.state
	a := s.Skill.Artifact(loc.File)
	if s.Code == nil || a == nil {
		return ""
	}
	span, ok := mention(a.Content, rep, loc.Line)
	if !ok {
		return ""
	}
	sub := s.Code.At(a.Path, span, false)
	if sub == nil {
		return ""
	}
	t := s.Code.Trace(sub)
	if len(t.Feeds) == 0 {
		return ""
	}
	return llmcall.Render(files, "place-trace.md", map[string]string{"trace": strings.TrimSpace(t.Text())})
}

func mention(content string, rep *refs.Report, line int) (code.Span, bool) {
	starts := textutil.LineStarts(content)
	if line < 1 || line > len(starts) {
		return code.Span{}, false
	}
	start, end := starts[line-1], len(content)
	if line < len(starts) {
		end = starts[line]
	}
	for _, needle := range append([]string{rep.Ref.Value}, rep.Ref.URLs...) {
		if found := strings.Index(content[start:end], needle); found >= 0 {
			return code.Span{Start: start + found, End: start + found + len(needle)}, true
		}
	}
	return code.Span{}, false
}

func (r *run) raise(rep *refs.Report, answers []placed, serverFound bool) []finding.Finding {
	s := r.state
	var out []finding.Finding
	raised := make(map[string]bool)
	for j, loc := range rep.Ref.Locations {
		rule, ok := s.Rules.ByID[usageRules[answers[j].choice]]
		if !ok || !rule.Enabled || answers[j].probability < r.layer.Config.DismissProbability {
			continue
		}
		r.cover(rule.ID, loc)
		if raised[rule.ID] || r.silent(rep, rule.ID, loc, serverFound) {
			continue
		}
		raised[rule.ID] = true
		evidence := refcheck.Line(s.Skill, loc)
		if evidence == "" {
			evidence = rep.Ref.Value
		}
		f := rule.Finding(loc, textutil.Clip(evidence))
		f.Reference = rep.Ref.Value
		if rule.Decision == nil {
			f.Decision = &finding.Decision{Answers: []finding.Answer{{Question: rules.Usage, Place: rep.LocationID(j), Choice: answers[j].choice, Probability: answers[j].probability}}}
		}
		out = append(out, f)
	}
	return out
}

func (r *run) cover(rule string, loc finding.Location) {
	first, last := loc.Line, loc.Line
	if _, _, b, ok := r.blockAt(loc); ok {
		first, last = b.first, b.last
	}
	r.state.Cover(rule, loc.File, first, last)
}

func (r *run) silent(rep *refs.Report, rule string, loc finding.Location, serverFound bool) bool {
	if rule == usageRules[toolServer] && (serverFound || !slices.Contains(networkKinds, rep.Ref.Kind)) {
		return true
	}
	return r.state.Restated(rule, loc.File, loc.Line)
}

func (r *run) server(rep *refs.Report, answers []placed) []finding.Finding {
	s := r.state
	rule, ok := s.Rules.ByID[untrusted]
	at := slices.IndexFunc(answers, func(a placed) bool {
		return a.choice == toolServer && a.probability >= r.layer.Config.DismissProbability
	})
	if !ok || !rule.Enabled || at < 0 {
		return nil
	}
	var facts []string
	for _, f := range s.Findings {
		fact := s.Rules.ByID[f.Rule]
		if f.Reference != rep.Ref.Value || fact.Layer != config.ReferencesLayer || fact.Severity.Rank() <= finding.Low.Rank() || slices.Contains(facts, f.Rule) {
			continue
		}
		facts = append(facts, f.Rule)
	}
	if len(facts) == 0 {
		return nil
	}
	f := rule.Finding(rep.Ref.Locations[at], textutil.Clip(rep.Ref.Value+": "+strings.Join(facts, ", ")))
	f.Reference = rep.Ref.Value
	return []finding.Finding{f}
}
