// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	unknown    = "unknown"
	toolServer = "tool-server"
	untrusted  = "TOOL-SERVER-UNTRUSTED"
	rootError  = "DOMAIN-HTTP-ERROR"
)

var usageRules = map[string]string{
	"agent-instructions": "REMOTE-INSTRUCTIONS-FOLLOWED",
	"agent-install":      "REMOTE-CODE-AT-INSTALL",
	"agent-upload":       "REMOTE-ENDPOINT-UPLOAD-HIDDEN",
}

type placed struct {
	choice      string
	probability float64
}

type classifier struct {
	run    *run
	policy usagePolicy
}

func (r *run) usages(ctx context.Context) error {
	c := &classifier{run: r, policy: loadUsagePolicy()}
	s := r.state
	var raised []finding.Finding
	classified := false
	for i := range s.Refs {
		rep := &s.Refs[i]
		if !refcheck.Categorized(*rep) || len(rep.Ref.Locations) == 0 {
			continue
		}
		answers, err := c.report(ctx, rep)
		if err != nil {
			return err
		}
		classified = true
		raised = append(raised, c.raise(rep, answers)...)
		raised = append(raised, c.server(rep, answers)...)
		c.floor(rep, answers)
	}
	s.Record(finding.Dedup(raised)...)
	if a := s.Skill.Artifact(refcheck.BriefPath); a != nil && classified {
		a.Content = refcheck.Brief(s.Skill, s.Refs)
		a.Size = int64(len(a.Content))
	}
	return nil
}

func (c *classifier) report(ctx context.Context, rep *refs.Report) ([]placed, error) {
	rep.Usages = make([]string, len(rep.Ref.Locations))
	answers := make([]placed, len(rep.Ref.Locations))
	for j := range answers {
		answers[j] = placed{choice: unknown}
	}
	list := blocks(c.run.state, rep)
	text := referenceState(rep, list)
	if len(list) == 0 || len(text) > c.run.layer.Config.MaxStateChars {
		return answers, nil
	}
	replies, err := c.ask(ctx, rep, list, text)
	if err != nil {
		return nil, err
	}
	for _, b := range list {
		a := c.read(replies[b.id])
		for _, j := range b.places {
			answers[j] = a
			if c.confident(a) {
				rep.Usages[j] = a.choice
			}
		}
	}
	return answers, nil
}

func (c *classifier) ask(ctx context.Context, rep *refs.Report, list []*block, text string) (map[string]decisions.Answer, error) {
	r := c.run
	state, err := json.Marshal(text)
	if err != nil {
		return nil, err
	}
	questions := make(map[string]decisions.Question, len(list))
	for _, b := range list {
		questions[b.id] = c.policy.question(rep, b.id)
	}
	answers, usage, err := r.layer.Client.Decide(ctx, state, questions)
	r.report.Requests++
	r.report.Usage = r.report.Usage.Add(usage)
	return answers, err
}

func (c *classifier) read(a decisions.Answer) placed {
	out := placed{choice: a.Choice, probability: a.Probabilities[a.Choice]}
	if _, ok := c.policy.Options[out.choice]; !ok {
		return placed{choice: unknown}
	}
	return out
}

func (c *classifier) confident(a placed) bool {
	return a.choice != unknown && a.probability >= c.run.layer.Config.MinProbability
}

func (c *classifier) raise(rep *refs.Report, answers []placed) []finding.Finding {
	s := c.run.state
	var out []finding.Finding
	raised := make(map[string]bool)
	for j, loc := range rep.Ref.Locations {
		rule, ok := s.Rules.ByID[usageRules[answers[j].choice]]
		if !ok || !rule.Enabled {
			continue
		}
		s.Cover(rule.ID, loc.File, loc.Line-around, loc.Line+around)
		if raised[rule.ID] {
			continue
		}
		raised[rule.ID] = true
		evidence := refcheck.Line(s.Skill, loc)
		if evidence == "" {
			evidence = rep.Ref.Value
		}
		f := rule.Finding(loc, textutil.Clip(evidence))
		f.Reference = rep.Ref.Value
		if rule.Decision == nil || len(rule.Decision.Dismiss) == 0 {
			f.Decision = &finding.Decision{Choice: answers[j].choice, Probability: answers[j].probability}
		}
		out = append(out, f)
	}
	return out
}

func (c *classifier) server(rep *refs.Report, answers []placed) []finding.Finding {
	s := c.run.state
	rule, ok := s.Rules.ByID[untrusted]
	at := slices.IndexFunc(answers, func(a placed) bool { return a.choice == toolServer })
	if !ok || !rule.Enabled || at < 0 {
		return nil
	}
	var facts []string
	for _, f := range s.Findings {
		if f.Reference != rep.Ref.Value || s.Rules.ByID[f.Rule].Layer != config.ReferencesLayer || slices.Contains(facts, f.Rule) {
			continue
		}
		if f.Rule != rootError || !parentServes(rep) {
			facts = append(facts, f.Rule)
		}
	}
	if len(facts) == 0 {
		return nil
	}
	f := rule.Finding(rep.Ref.Locations[at], textutil.Clip(rep.Ref.Value+": "+strings.Join(facts, ", ")))
	f.Reference = rep.Ref.Value
	return []finding.Finding{f}
}

func parentServes(rep *refs.Report) bool {
	for _, f := range rep.Facts {
		if f.Key == refcheck.ParentStatus {
			code, err := strconv.Atoi(f.Value)
			return err == nil && code < http.StatusBadRequest
		}
	}
	return false
}

func (c *classifier) floor(rep *refs.Report, answers []placed) {
	s := c.run.state
	lowest := 1.0
	for _, a := range answers {
		if !c.confident(a) {
			return
		}
		lowest = min(lowest, a.probability)
	}
	for i := range s.Findings {
		f := &s.Findings[i]
		rule := s.Rules.ByID[f.Rule]
		if f.Judgement != nil || f.Reference != rep.Ref.Value || len(rule.Usages) == 0 || applies(rule, rep.Usages) {
			continue
		}
		lower(f, rule, level(lowest), usageReason(rep.Usages, lowest))
	}
}

func applies(rule rules.Rule, usages []string) bool {
	return slices.ContainsFunc(usages, func(u string) bool {
		return slices.Contains(rule.Usages, u)
	})
}

func lower(f *finding.Finding, rule rules.Rule, confidence int, reason string) {
	floor := rule.Judge.DowngradeFloor
	switch {
	case !rule.Judge.Downgrade:
	case floor == "":
		f.Judgement = &finding.Judgement{State: finding.Dismissed, From: f.Severity, Confidence: confidence, Reason: reason}
	case floor.Rank() < f.Severity.Rank():
		f.Judgement = &finding.Judgement{State: finding.Downgraded, From: f.Severity, Confidence: confidence, Reason: reason}
		f.Severity = floor
	}
}

func usageReason(usages []string, lowest float64) string {
	seen := slices.Clone(usages)
	slices.Sort(seen)
	return fmt.Sprintf("The decision model read every place the skill uses this reference and answered %s, lowest probability %.2f. The rule does not apply to these usages.",
		strings.Join(slices.Compact(seen), ", "), lowest)
}
