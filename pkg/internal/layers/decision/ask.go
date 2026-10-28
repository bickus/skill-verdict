// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"embed"
	"encoding/json"
	"slices"

	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

//go:embed *.md
var files embed.FS

const (
	oversized  = "the state or the request is over the configured size"
	failed     = "an earlier request to the decision model failed"
	unanswered = "the model gave no answer"
)

type query struct {
	id       string
	question decisions.Question
}

func queries(questions []named, about, place string) []query {
	out := make([]query, 0, len(questions))
	for _, n := range questions {
		id := n.name
		if place != "" {
			id = place + "." + n.name
		}
		out = append(out, query{id: id, question: decisions.Question{Options: n.question.Options, Instructions: instructions(n.question, about, place)}})
	}
	return out
}

func instructions(q rules.Question, about, place string) map[string]string {
	out := map[string]string{"question": q.Question}
	if q.Focus != "" {
		out["focus"] = q.Focus
	}
	if about != "" {
		out["subject"] = about
	}
	if place != "" {
		out["place"] = "place " + place
	}
	return out
}

func guard() query {
	q := rules.Template(rules.Guard)
	return query{id: rules.Guard, question: decisions.Question{Options: q.Options, Instructions: instructions(q, "", "")}}
}

func chars(q decisions.Question) int {
	n := 0
	for _, m := range []map[string]string{q.Instructions, q.Options} {
		for k, v := range m {
			n += len(k) + len(v)
		}
	}
	return n
}

func skip(questions []named, reason string) map[string]reply {
	out := make(map[string]reply, len(questions))
	for _, n := range questions {
		out[n.name] = reply{skipped: reason}
	}
	return out
}

func (r *run) ask(ctx context.Context, state string, qs []query) (map[string]reply, reply) {
	cfg := r.layer.Config
	state = textutil.Cut(state, cfg.MaxStateChars)
	all := append(slices.Clone(qs), guard())
	size := len(state)
	for _, q := range all {
		size += chars(q.question)
	}
	switch {
	case size > cfg.MaxRequestChars:
		return skipped(all, oversized)
	case r.err != nil:
		return skipped(all, failed)
	}
	body, err := json.Marshal(state)
	if err != nil {
		return skipped(all, err.Error())
	}
	questions := make(map[string]decisions.Question, len(all))
	for _, q := range all {
		questions[q.id] = q.question
	}
	answers, usage, err := r.layer.Client.Decide(ctx, body, questions)
	r.report.Requests++
	r.report.Usage = r.report.Usage.Add(usage)
	if err != nil {
		r.err = err
		return skipped(all, "the request failed: "+err.Error())
	}
	out := make(map[string]reply, len(all))
	for _, q := range all {
		a, ok := answers[q.id]
		if !ok {
			out[q.id] = reply{skipped: unanswered}
			continue
		}
		out[q.id] = reply{choice: a.Choice, probabilities: a.Probabilities}
	}
	return split(out)
}

func skipped(all []query, reason string) (map[string]reply, reply) {
	out := make(map[string]reply, len(all))
	for _, q := range all {
		out[q.id] = reply{skipped: reason}
	}
	return split(out)
}

func split(out map[string]reply) (map[string]reply, reply) {
	g := out[rules.Guard]
	delete(out, rules.Guard)
	return out, g
}
