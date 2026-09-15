// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package honeypot

import (
	"context"
	"embed"
	"path"
	"path/filepath"
	"time"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

//go:embed *.md *.json
var files embed.FS

const (
	name  = "honeypot"
	rule  = "HONEYPOT-TRIGGERED"
	entry = "SKILL.md"
	home  = "/home/dev"
)

type Layer struct {
	llmcall.Runner
}

func (Layer) Name() string {
	return name
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	st := &store{skill: s.Skill, base: path.Join(home, ".claude", "skills", filepath.Base(s.Skill.Root)), maxChars: l.LLM.MaxInputChars}
	st.loaded = llmcall.Render(files, "loaded.md", map[string]string{"base": st.base, "body": body(s.Skill)})
	rep, err := l.Runner.Run(ctx, llmcall.Session{
		System: llmcall.Render(files, "system.md", map[string]string{
			"cwd":  path.Join(home, "projects", "app"),
			"date": time.Now().Format(time.DateOnly),
		}),
		User:     llmcall.Render(files, "user.md", map[string]string{"skills": listing(s.Skill), "name": s.Skill.Name}),
		Reminder: llmcall.Text(files, "reminder.md"),
		Tools:    st.tools(),
	})
	if st.evidence != "" && s.Rules.ForLayer(name)[rule] {
		s.Record(s.Rules.ByID[rule].Finding(finding.Location{File: entry}, textutil.Evidence(st.evidence)))
	}
	return rep, err
}

func body(s *inventory.Skill) string {
	a := s.Artifact(entry)
	if a == nil {
		return ""
	}
	return inventory.Body(a.Content)
}

func listing(s *inventory.Skill) string {
	if s.Description == "" {
		return "- " + s.Name
	}
	return "- " + s.Name + ": " + s.Description
}
