// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package honeypot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/agent"
)

const (
	SkillTool  = "Skill"
	ReadTool   = "Read"
	SubmitTool = "submit_result"
)

type store struct {
	skill    *inventory.Skill
	base     string
	loaded   string
	maxChars int
	evidence string
}

func (s *store) tools() []agent.Tool {
	return []agent.Tool{
		trap{s, "Agent"},
		trap{s, "Bash"},
		trap{s, "Edit"},
		reader{s},
		loader{s},
		trap{s, "WebFetch"},
		trap{s, "WebSearch"},
		trap{s, "Write"},
		submit{},
	}
}

func (s *store) Final() bool {
	return s.evidence != ""
}

func (s *store) catch(tool string, raw json.RawMessage) {
	if s.evidence != "" {
		return
	}
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		b.Reset()
		b.Write(raw)
	}
	s.evidence = tool + " " + b.String()
}

func spec(tool string) llm.Tool {
	file := strings.ToLower(tool)
	return llm.Tool{Name: tool, Description: llmcall.Text(files, file+".md"), Parameters: llmcall.Schema(files, file+".json")}
}

type trap struct {
	*store
	name string
}

func (t trap) Spec() llm.Tool {
	return spec(t.name)
}

func (t trap) Call(_ context.Context, raw json.RawMessage) (string, error) {
	t.catch(t.name, raw)
	return "", nil
}

type loader struct {
	*store
}

type skillArgs struct {
	Skill string `json:"skill"`
}

func (loader) Spec() llm.Tool {
	return spec(SkillTool)
}

func (l loader) Call(_ context.Context, raw json.RawMessage) (string, error) {
	var args skillArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimPrefix(args.Skill, "/") != l.skill.Name {
		return "", fmt.Errorf("unknown skill: %s", args.Skill)
	}
	return l.loaded, nil
}

type submit struct{}

func (submit) Final() bool {
	return true
}

func (submit) Spec() llm.Tool {
	return llm.Tool{Name: SubmitTool, Description: llmcall.Text(files, "submit.md"), Parameters: llmcall.Schema(files, "submit.json")}
}

func (submit) Call(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
