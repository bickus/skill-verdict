// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package judge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	VerdictTool = "verdict"
	SubmitTool  = "submit_result"
	dismiss     = "dismiss"
)

type record struct {
	Finding     string `json:"finding"`
	Verdict     string `json:"verdict"`
	Confidence  int    `json:"confidence"`
	Explanation string `json:"explanation"`
}

type batch struct {
	Verdicts []record `json:"verdicts"`
}

type target struct {
	index  int
	rule   string
	from   finding.Severity
	policy rules.JudgePolicy
}

type outcome struct {
	index      int
	severity   finding.Severity
	state      finding.State
	confidence int
	reason     string
}

type store struct {
	targets map[string]target
	byID    map[string]outcome
	order   []string
}

func newStore(s *pipeline.State, groups ...[]labelled) *store {
	st := &store{targets: make(map[string]target), byID: make(map[string]outcome)}
	for _, group := range groups {
		for _, e := range group {
			f := s.Findings[e.index]
			st.targets[e.id] = target{index: e.index, rule: f.Rule, from: f.Severity, policy: s.Rules.ByID[f.Rule].Judge}
		}
	}
	return st
}

func (s *store) results() []outcome {
	out := make([]outcome, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.byID[id])
	}
	return out
}

func (s *store) take(rs []record) (string, error) {
	ids := make([]string, len(rs))
	for i, r := range rs {
		id, err := s.check(r)
		if err != nil {
			return "", fmt.Errorf("verdicts[%d] rejected, nothing recorded: %w", i, err)
		}
		ids[i] = id
	}
	var b strings.Builder
	for i, r := range rs {
		fmt.Fprintf(&b, "%s\n", s.put(ids[i], r))
	}
	fmt.Fprintf(&b, "%d on record", len(s.order))
	return b.String(), nil
}

func (s *store) check(r record) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(r.Finding))
	_, known := s.targets[id]
	switch {
	case !known:
		return "", fmt.Errorf("no finding %q in the list", r.Finding)
	case r.Verdict != dismiss && finding.Severity(r.Verdict).Rank() == 0:
		return "", fmt.Errorf("verdict %q is neither a severity nor %s", r.Verdict, dismiss)
	case r.Explanation == "":
		return "", errors.New("explanation is empty")
	case r.Confidence < 1 || r.Confidence > 4:
		return "", errors.New("confidence must be 1 to 4")
	}
	return id, nil
}

func (s *store) put(id string, r record) string {
	t := s.targets[id]
	severity, state := resolve(t, r.Verdict)
	if _, seen := s.byID[id]; !seen {
		s.order = append(s.order, id)
	}
	s.byID[id] = outcome{index: t.index, severity: severity, state: state, confidence: r.Confidence, reason: r.Explanation}
	return report(id, t, r, severity, state)
}

func resolve(t target, want string) (finding.Severity, finding.State) {
	if want == dismiss {
		return dismissal(t)
	}
	end := finding.Severity(want)
	if end.Rank() > t.from.Rank() {
		return t.from, finding.Confirmed
	}
	if bottom := lowest(t); end.Rank() < bottom.Rank() {
		end = bottom
	}
	switch {
	case end.Rank() < t.from.Rank():
		return end, finding.Downgraded
	case want == string(t.from):
		return t.from, finding.Confirmed
	}
	return t.from, finding.Upheld
}

func dismissal(t target) (finding.Severity, finding.State) {
	bottom := lowest(t)
	switch {
	case t.policy.Downgrade && t.policy.DowngradeFloor == "":
		return t.from, finding.Dismissed
	case bottom.Rank() < t.from.Rank():
		return bottom, finding.Downgraded
	}
	return t.from, finding.Upheld
}

func lowest(t target) finding.Severity {
	switch {
	case !t.policy.Downgrade:
		return t.from
	case t.policy.DowngradeFloor == "":
		return finding.Low
	}
	return t.policy.DowngradeFloor
}

func report(id string, t target, r record, severity finding.Severity, state finding.State) string {
	text := fmt.Sprintf("%s: %s", id, severity)
	if state == finding.Dismissed {
		text = id + ": dismissed"
	}
	var notes []string
	if r.Verdict != string(severity) && state != finding.Dismissed {
		notes = append(notes, limit(t, r.Verdict))
	}
	if !llmcall.Confident(r.Confidence) {
		notes = append(notes, fmt.Sprintf("not applied below confidence %d.", llmcall.MinConfident))
	}
	if len(notes) == 0 {
		return text
	}
	return text + " - " + strings.Join(notes, " ")
}

func limit(t target, want string) string {
	switch {
	case want != dismiss && finding.Severity(want).Rank() > t.from.Rank():
		return "rule gives no higher than this."
	case !t.policy.Downgrade:
		return "rule cannot be lowered at all."
	}
	return "rule goes no lower than this."
}

func params() json.RawMessage {
	values := make([]string, 0, len(finding.Severities())+1)
	for _, s := range finding.Severities() {
		values = append(values, string(s))
	}
	item := llmcall.Property(llmcall.Schema(files, "item.json"), "verdict", map[string]any{"enum": append(values, dismiss)})
	return llmcall.List(item, "verdicts")
}

type verdicts struct {
	*store
}

func (verdicts) Final() bool {
	return false
}

func (verdicts) Spec() llm.Tool {
	return llm.Tool{Name: VerdictTool, Description: llmcall.Text(files, "verdict.md"), Parameters: params()}
}

func (v verdicts) Call(_ context.Context, raw json.RawMessage) (string, error) {
	args, err := llmcall.Args[batch](raw)
	if err != nil {
		return "", err
	}
	return v.take(args.Verdicts)
}

type submit struct {
	*store
}

func (submit) Final() bool {
	return true
}

func (submit) Spec() llm.Tool {
	return llm.Tool{Name: SubmitTool, Description: llmcall.Text(files, "submit.md"), Parameters: params()}
}

func (s submit) Call(_ context.Context, raw json.RawMessage) (string, error) {
	args, err := llmcall.Args[batch](raw)
	if err != nil {
		return "", err
	}
	_, err = s.take(args.Verdicts)
	return "", err
}
