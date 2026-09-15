// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/llm"
)

const (
	FindingTool = "finding"
	SubmitTool  = "submit_result"
)

type item struct {
	Action     string `json:"action"`
	File       string `json:"file"`
	RuleID     string `json:"rule_id"`
	Message    string `json:"message"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Confidence int    `json:"confidence"`
}

type list struct {
	Findings []item `json:"findings"`
}

type store struct {
	skill   *inventory.Skill
	allowed map[string]bool
	items   []item
}

type recorder struct {
	*store
}

func (recorder) Final() bool {
	return false
}

func (recorder) Spec() llm.Tool {
	action := map[string]any{"type": "string", "enum": []string{"add", "remove"}, "description": llmcall.Text(files, "action.md")}
	return llm.Tool{
		Name:        FindingTool,
		Description: llmcall.Text(files, "finding.md"),
		Parameters:  llmcall.Property(llmcall.Schema(files, "item.json"), "action", action),
	}
}

func (r recorder) Call(_ context.Context, raw json.RawMessage) (string, error) {
	it, err := llmcall.Args[item](raw)
	if err != nil {
		return "", err
	}
	switch it.Action {
	case "add":
		if err = r.check(it); err != nil {
			return "", fmt.Errorf("not recorded: %w", err)
		}
		return r.add(it), nil
	case "remove":
		i := r.index(it)
		if i < 0 {
			return "", errors.New("nothing on record with this identity")
		}
		r.items = slices.Delete(r.items, i, i+1)
		return r.status("removed"), nil
	}
	return "", fmt.Errorf("bad arguments: action %q, want add or remove", it.Action)
}

type submit struct {
	*store
}

func (submit) Final() bool {
	return true
}

func (submit) Spec() llm.Tool {
	return llm.Tool{
		Name:        SubmitTool,
		Description: llmcall.Text(files, "submit.md"),
		Parameters:  llmcall.List(llmcall.Schema(files, "item.json"), "findings"),
	}
}

func (s submit) Call(_ context.Context, raw json.RawMessage) (string, error) {
	args, err := llmcall.Args[list](raw)
	if err != nil {
		return "", err
	}
	for i, it := range args.Findings {
		if err = s.check(it); err != nil {
			return "", fmt.Errorf("findings[%d] rejected, nothing submitted: %w", i, err)
		}
	}
	for _, it := range args.Findings {
		s.add(it)
	}
	return "", nil
}

func (s *store) check(it item) error {
	switch {
	case s.skill.Artifact(it.File) == nil:
		return fmt.Errorf("no artifact at %q; use a path from the listing", it.File)
	case !s.allowed[it.RuleID]:
		return fmt.Errorf("unknown rule_id %q", it.RuleID)
	case it.Message == "":
		return errors.New("message is empty")
	case it.StartLine < 1:
		return errors.New("start_line must be 1 or more")
	case it.EndLine != 0 && it.EndLine < it.StartLine:
		return errors.New("end_line is before start_line")
	case it.Confidence < 1 || it.Confidence > 4:
		return errors.New("confidence must be 1 to 4")
	}
	return nil
}

func (s *store) index(it item) int {
	return slices.IndexFunc(s.items, func(o item) bool {
		return o.File == it.File && o.RuleID == it.RuleID && o.StartLine == it.StartLine
	})
}

func (s *store) add(it item) string {
	i := s.index(it)
	low := fmt.Sprintf("confidence %d is below %d", it.Confidence, llmcall.MinConfident)
	switch {
	case !llmcall.Confident(it.Confidence) && i >= 0:
		s.items = slices.Delete(s.items, i, i+1)
		return s.status("withdrawn, " + low)
	case !llmcall.Confident(it.Confidence):
		return s.status("not recorded, " + low)
	case i >= 0:
		s.items[i] = it
		return s.status("replaced")
	}
	s.items = append(s.items, it)
	return s.status("recorded")
}

func (s *store) status(verb string) string {
	return fmt.Sprintf("%s, %d on record", verb, len(s.items))
}
