// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

const (
	maxHeadingLevel = 6
	maxMentions     = 3
)

type position struct {
	headings []string
	levels   []int
	marker   string
	language string
}

func spot(s *pipeline.State, a *inventory.Artifact, lines []string, line int) string {
	if a.ContentType != inventory.Markdown {
		return named(s, a)
	}
	if line < 1 || line > len(lines) {
		return ""
	}
	first := 0
	if strings.TrimSpace(lines[0]) == "---" {
		end := slices.IndexFunc(lines[1:], func(l string) bool { return strings.TrimSpace(l) == "---" })
		if end < 0 || line <= end+2 {
			return "inside: frontmatter"
		}
		first = end + 2
	}
	p := &position{}
	for _, text := range lines[first:line] {
		p.read(text)
	}
	return p.String()
}

func named(s *pipeline.State, a *inventory.Artifact) string {
	base := path.Base(a.Path)
	var out []string
	for i := range s.Skill.Artifacts {
		other := &s.Skill.Artifacts[i]
		if other.Path == a.Path || other.Kind != inventory.Content || !strings.Contains(other.Content, base) {
			continue
		}
		for n, text := range textutil.Lines(other.Content) {
			if len(out) < maxMentions && strings.Contains(text, base) {
				out = append(out, fmt.Sprintf("%s L%d: %s", other.Path, n+1, textutil.Cut(strings.TrimSpace(text), maxLineChars)))
			}
		}
	}
	if len(out) == 0 {
		return ""
	}
	return "other files of the skill name this file:\n" + strings.Join(out, "\n")
}

func (p *position) read(text string) {
	run, info := code.Fence(text)
	switch {
	case p.marker == "" && run != "":
		p.marker = run
		p.language, _, _ = strings.Cut(strings.TrimSpace(info), " ")
	case p.marker != "" && strings.HasPrefix(run, p.marker) && strings.TrimSpace(info) == "":
		p.marker = ""
	case p.marker == "":
		p.heading(text)
	}
}

func (p *position) heading(text string) {
	level := len(text) - len(strings.TrimLeft(text, "#"))
	if level == 0 || level > maxHeadingLevel || !strings.HasPrefix(text[level:], " ") {
		return
	}
	for len(p.levels) > 0 && p.levels[len(p.levels)-1] >= level {
		p.levels = p.levels[:len(p.levels)-1]
		p.headings = p.headings[:len(p.headings)-1]
	}
	p.levels = append(p.levels, level)
	p.headings = append(p.headings, strings.TrimSpace(text[level:]))
}

func (p *position) String() string {
	out := "inside: prose"
	switch {
	case p.marker != "" && p.language != "":
		out = "inside: fenced code block, language " + p.language
	case p.marker != "":
		out = "inside: fenced code block"
	}
	if len(p.headings) > 0 {
		out += "\nsection: " + strings.Join(p.headings, " > ")
	}
	return out
}
