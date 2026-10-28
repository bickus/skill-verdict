// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const oversized = "the state or the request is over the configured size"

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

func (r *run) places(ctx context.Context, rule rules.Rule, f finding.Finding) (outcome, error) {
	if _, ok := r.placed[at(f)]; !ok {
		if err := r.group(ctx, rule, target(rule, f)); err != nil {
			return outcome{}, err
		}
	}
	return r.placed[at(f)], nil
}

func (r *run) group(ctx context.Context, rule rules.Rule, name string) error {
	var keys []spotted
	var texts []string
	questions := make(map[string]decisions.Question)
	size := 0
	for _, f := range r.state.Findings {
		if f.Rule != rule.ID || f.Decision != nil || target(rule, f) != name {
			continue
		}
		anchor := f.Evidence
		if f.Reference != "" {
			anchor = f.Reference
		}
		values, _ := located(r.state, f.Location, anchor)
		if values == nil {
			r.placed[at(f)] = outcome{skipped: "the line was not found"}
			continue
		}
		values["id"] = placeID(len(keys))
		values["trace"] = ""
		q := placeQuestion(rule.Decision, name, values["id"])
		questions[values["id"]] = q
		size += chars(q)
		keys = append(keys, at(f))
		texts = append(texts, llmcall.Render(files, "place.md", values))
	}
	state, err := json.Marshal(llmcall.Render(files, "places.md", map[string]string{"subject": name, "places": strings.Join(texts, "\n")}))
	if err != nil || len(keys) == 0 {
		return err
	}
	if cfg := r.layer.Config; len(state) > cfg.MaxStateChars || len(state)+size > cfg.MaxRequestChars {
		for _, key := range keys {
			r.placed[key] = outcome{skipped: oversized}
		}
		return nil
	}
	answers, usage, err := r.layer.Client.Decide(ctx, state, questions)
	r.report.Requests++
	r.report.Usage = r.report.Usage.Add(usage)
	for i, key := range keys {
		r.placed[key] = r.judge(rule.Decision, answers[placeID(i)])
	}
	return err
}

func placeID(index int) string {
	return fmt.Sprintf("P%d", index+1)
}

func placeQuestion(d *rules.DecisionPolicy, name, id string) decisions.Question {
	q := question(d, name)
	q.Instructions["place"] = "place " + id
	return q
}
