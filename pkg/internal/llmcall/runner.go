// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall

import (
	"context"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/agent"
)

const (
	MinConfident = 3
	maxConfident = 4
)

func Confident(level int) bool {
	return level >= MinConfident && level <= maxConfident
}

type Runner struct {
	Provider llm.Provider
	LLM      config.LLMConfig
	Calls    int
}

type Session struct {
	System   string
	User     string
	Reminder string
	Tools    []agent.Tool
}

func (r Runner) Run(ctx context.Context, s Session) (pipeline.Report, error) {
	loop := agent.Loop{
		Provider:     r.Provider,
		System:       s.System,
		Tools:        s.Tools,
		Reminder:     s.Reminder,
		Calls:        r.Calls,
		CompactionAt: r.LLM.CompactionAt,
	}
	o, err := loop.Run(ctx, s.User)
	return pipeline.Report{Usage: o.Usage, Requests: o.Requests}, err
}
