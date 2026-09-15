// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package discovery

import (
	"context"
	"embed"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/llm/agent"
	"github.com/bickus/skill-verdict/pkg/rules"
)

//go:embed *.md *.json
var files embed.FS

const name = "discovery"

type Layer struct {
	llmcall.Runner
}

func (Layer) Name() string {
	return name
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	st := &store{skill: s.Skill, allowed: s.Rules.ForLayer(name)}
	entry := llmcall.Entry(s.Skill)
	rep, err := l.Runner.Run(ctx, llmcall.Session{
		System: llmcall.System(files, "system.md", map[string]string{"rules": ruleList(s.Rules)}),
		User: llmcall.Render(files, "user.md", map[string]string{
			"origins":   llmcall.Origins(s.Skill),
			"artifacts": llmcall.Artifacts(s.Skill),
			"entry":     entry.Path,
			"content":   llmcall.Inline(entry, l.LLM.MaxInputChars),
		}),
		Reminder: llmcall.Text(files, "reminder.md"),
		Tools:    []agent.Tool{llmcall.Read{Skill: s.Skill, MaxChars: l.LLM.MaxInputChars}, recorder{st}, submit{st}},
	})
	s.Record(finding.Dedup(convert(s, st.allowed, st.items))...)
	return rep, err
}

func ruleList(set *rules.Set) string {
	var b strings.Builder
	for _, id := range slices.Sorted(maps.Keys(set.ForLayer(name))) {
		r := set.ByID[id]
		fmt.Fprintf(&b, "rule_id: %s\ntitle: %s\ndescription: %s\n\n", id, r.Title, r.Description)
	}
	return strings.TrimRight(b.String(), "\n")
}

func convert(s *pipeline.State, allowed map[string]bool, items []item) []finding.Finding {
	var out []finding.Finding
	for _, it := range items {
		if !allowed[it.RuleID] {
			continue
		}
		f := s.Rules.ByID[it.RuleID].Finding(finding.Location{File: it.File, Line: it.StartLine}, textutil.Clip(it.Message))
		if it.EndLine > it.StartLine {
			f.Location.EndLine = it.EndLine
		}
		out = append(out, f)
	}
	return out
}
