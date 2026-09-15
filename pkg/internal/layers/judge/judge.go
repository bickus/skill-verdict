// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package judge

import (
	"cmp"
	"context"
	"embed"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/llm/agent"
	"github.com/bickus/skill-verdict/pkg/rules"
)

//go:embed *.md *.json
var files embed.FS

const name = "judge"

type Layer struct {
	llmcall.Runner
}

func (Layer) Name() string {
	return name
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	down, fixed := label(s, order(s.Findings))
	if len(down)+len(fixed) == 0 {
		return pipeline.Report{}, nil
	}
	st := newStore(s, down, fixed)
	rep, err := l.Runner.Run(ctx, llmcall.Session{
		System:   llmcall.System(files, "system.md", nil),
		User:     l.user(s, down, fixed),
		Reminder: llmcall.Text(files, "reminder.md"),
		Tools:    []agent.Tool{llmcall.Read{Skill: s.Skill, MaxChars: l.LLM.MaxInputChars}, verdicts{st}, submit{st}},
	})
	apply(s, st.results())
	return rep, err
}

func apply(s *pipeline.State, results []outcome) {
	for _, o := range results {
		f := &s.Findings[o.index]
		if !llmcall.Confident(o.confidence) {
			continue
		}
		f.Judgement = &finding.Judgement{State: o.state, From: f.Severity, Confidence: o.confidence, Reason: textutil.Clip(o.reason)}
		f.Severity = o.severity
	}
}

func order(findings []finding.Finding) []int {
	best := make(map[string]int)
	var pending []int
	for i, f := range findings {
		if f.Judgement != nil {
			continue
		}
		pending = append(pending, i)
		if rank, seen := best[f.Location.File]; !seen || f.Severity.Rank() > rank {
			best[f.Location.File] = f.Severity.Rank()
		}
	}
	slices.SortStableFunc(pending, func(a, b int) int {
		fa, fb := findings[a].Location.File, findings[b].Location.File
		return cmp.Or(cmp.Compare(best[fb], best[fa]), cmp.Compare(fa, fb))
	})
	return pending
}

type labelled struct {
	index int
	id    string
}

func label(s *pipeline.State, pending []int) (down, fixed []labelled) {
	for _, i := range pending {
		if s.Rules.ByID[s.Findings[i].Rule].Judge.Downgrade {
			down = append(down, labelled{index: i, id: fmt.Sprintf("D%d", len(down)+1)})
			continue
		}
		fixed = append(fixed, labelled{index: i, id: fmt.Sprintf("ND%d", len(fixed)+1)})
	}
	return down, fixed
}

func (l Layer) user(s *pipeline.State, down, fixed []labelled) string {
	entry := llmcall.Entry(s.Skill)
	return llmcall.Render(files, "user.md", map[string]string{
		"origins":                  llmcall.Origins(s.Skill),
		"artifacts":                llmcall.Artifacts(s.Skill),
		"rulesdowngradeable":       ruleBlock(s, down, "rules-downgradeable.md"),
		"rulesnondowngradeable":    ruleBlock(s, fixed, "rules-nondowngradeable.md"),
		"findingsdowngradeable":    findingBlock(s, down, "findings-downgradeable.md"),
		"findingsnondowngradeable": findingBlock(s, fixed, "findings-nondowngradeable.md"),
		"briefs":                   l.briefs(s.Skill),
		"entry":                    entry.Path,
		"content":                  llmcall.Inline(entry, l.LLM.MaxInputChars),
	})
}

func (l Layer) briefs(skill *inventory.Skill) string {
	var b strings.Builder
	for i := range skill.Artifacts {
		a := &skill.Artifacts[i]
		if a.Kind != inventory.Brief {
			continue
		}
		b.WriteString(llmcall.Render(files, "brief.md", map[string]string{
			"path":    a.Path,
			"content": shift(textutil.Cut(a.Content, l.LLM.MaxInputChars)),
		}))
	}
	if b.Len() == 0 {
		return ""
	}
	return llmcall.Render(files, "briefs.md", map[string]string{"briefs": strings.TrimSuffix(b.String(), "\n")})
}

const maxDepth = 6

var header = regexp.MustCompile(`(?m)^#{1,6} `)

func shift(text string) string {
	under := depth(llmcall.Text(files, "brief.md"))
	return header.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Repeat("#", min(depth(m)+under, maxDepth)) + " "
	})
}

func depth(line string) int {
	return len(line) - len(strings.TrimLeft(line, "#"))
}

func ruleBlock(s *pipeline.State, group []labelled, file string) string {
	if len(group) == 0 {
		return ""
	}
	var b strings.Builder
	seen := make(map[string]bool)
	for _, e := range group {
		id := s.Findings[e.index].Rule
		if seen[id] {
			continue
		}
		seen[id] = true
		r := s.Rules.ByID[id]
		fmt.Fprintf(&b, "[%s] - %s - %s - %s\n  %s\n", id, r.Title, r.Severity, floor(r), r.Description)
		if r.Judge.Instructions != "" {
			fmt.Fprintf(&b, "  Additional instructions for Judge agent: %s\n", r.Judge.Instructions)
		}
	}
	return llmcall.Render(files, file, map[string]string{"rules": strings.TrimSuffix(b.String(), "\n")})
}

func findingBlock(s *pipeline.State, group []labelled, file string) string {
	if len(group) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range group {
		f := s.Findings[e.index]
		fmt.Fprintf(&b, "%s: %s@%d:%s: %s\n", e.id, f.Location.File, f.Location.Line, f.Rule, f.Evidence)
	}
	return llmcall.Render(files, file, map[string]string{"findings": strings.TrimSuffix(b.String(), "\n")})
}

func floor(r rules.Rule) string {
	switch {
	case !r.Judge.Downgrade:
		return "n/a"
	case r.Judge.DowngradeFloor == "":
		return "none"
	}
	return string(r.Judge.DowngradeFloor)
}
