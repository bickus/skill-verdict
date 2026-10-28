// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static

import (
	"context"
	"path"
	"regexp"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name       = "static"
	maxPerRule = 50
	edges      = " \t\r\n;&|()`\"',"
)

type Layer struct{}

func (Layer) Name() string {
	return name
}

type key struct {
	rule string
	file string
	line int
}

func (Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	seen := make(map[key]bool)
	self := refcheck.NewSelfInstall(s.Skill.Folder())
	for i := range s.Skill.Artifacts {
		if err := ctx.Err(); err != nil {
			return pipeline.Report{}, err
		}
		a := &s.Skill.Artifacts[i]
		if a.Kind != inventory.Content {
			continue
		}
		switch a.ContentType {
		case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data:
			scanArtifact(s, a, seen, self)
		case inventory.Symlink:
			scanLink(s, a)
		case inventory.Media, inventory.Unknown, inventory.Special, inventory.Directory:
		}
	}
	brief(s)
	return pipeline.Report{}, nil
}

func applies(r rules.Compiled, a *inventory.Artifact) bool {
	if len(r.Files) > 0 && !config.MatchAny(r.Files, path.Base(a.Path)) {
		return false
	}
	return (r.Scope == rules.Link) == (a.ContentType == inventory.Symlink)
}

type source struct {
	state    *pipeline.State
	artifact *inventory.Artifact
	text     string
	folder   string
	starts   []int
	scanned  []int
	lines    []string
	fenced   []bool
	self     refcheck.SelfInstall
}

func scanArtifact(s *pipeline.State, a *inventory.Artifact, seen map[key]bool, self refcheck.SelfInstall) {
	src := &source{state: s, artifact: a, text: a.Text(), folder: strings.ToLower(s.Skill.Folder()), starts: textutil.LineStarts(a.Content), self: self}
	src.scanned, src.lines = src.starts, textutil.Lines(src.text)
	if a.Clean != "" {
		src.scanned = textutil.LineStarts(a.Clean)
	}
	if a.ContentType == inventory.Markdown {
		src.fenced = fences(src.lines)
	}
	for _, r := range s.Rules.Static {
		if !applies(r, a) {
			continue
		}
		found := unique(src.matches(r), seen)
		if len(found) > maxPerRule {
			found = found[:maxPerRule]
			a.Note(name + ": " + r.ID + " capped at 50")
		}
		s.Record(found...)
	}
}

func unique(found []finding.Finding, seen map[key]bool) []finding.Finding {
	var out []finding.Finding
	for _, f := range found {
		k := key{f.Rule, f.Location.File, f.Location.Line}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, f)
	}
	return out
}

func (src *source) matches(r rules.Compiled) []finding.Finding {
	var out []finding.Finding
	for _, re := range r.Patterns {
		for _, m := range re.FindAllStringIndex(src.text, -1) {
			if m[0] == m[1] {
				continue
			}
			if f, ok := src.build(r, m); ok {
				out = append(out, f)
			}
		}
	}
	return out
}

func (src *source) build(r rules.Compiled, m []int) (finding.Finding, bool) {
	a := src.artifact
	if src.text[m[0]] == '\n' && m[1]-m[0] > 1 {
		m = []int{m[0] + 1, m[1]}
	}
	row := textutil.LineAt(src.scanned, m[0])
	if !allowed(a.ContentType, r.Scope, src.fenced != nil && src.fenced[row-1]) || matchesAny(r.Exclude, src.lines[row-1]) {
		return finding.Finding{}, false
	}
	if src.selfInstall(row, m) || src.selfPath(m) {
		return finding.Finding{}, false
	}
	start := a.Source(m[0])
	end := max(a.Source(m[1]), start+1)
	line := textutil.LineAt(src.starts, start)
	f := r.Finding(finding.Location{File: a.Path, Line: line}, textutil.Evidence(src.text[m[0]:m[1]]))
	if endLine := textutil.LineAt(src.starts, end-1); endLine > line {
		f.Location.EndLine = endLine
	}
	src.link(r, code.Span{Start: start, End: end}, line)
	return f, true
}

func (src *source) selfInstall(row int, m []int) bool {
	spans := src.self.Spans(src.lines[row-1])
	if len(spans) == 0 {
		return false
	}
	start, end := m[0], m[1]
	if strings.IndexByte(edges, src.text[start]) >= 0 {
		start++
	}
	if end > start && strings.IndexByte(edges, src.text[end-1]) >= 0 {
		end--
	}
	start, end = start-src.scanned[row-1], end-src.scanned[row-1]
	for _, span := range spans {
		if start >= span[0] && end <= span[1] {
			return true
		}
	}
	return false
}

func (src *source) selfPath(m []int) bool {
	text := strings.Trim(src.text[m[0]:m[1]], edges+"[]<>")
	if src.folder == "" || strings.Contains(text, "://") || strings.ContainsAny(text, " \t") {
		return false
	}
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(text, `\`, "/")), "/")
	for i := 1; i < len(parts); i++ {
		underSkills := parts[i] == src.folder && parts[i-1] == "skills"
		ownEntry := parts[i] == "skill.md" && parts[i-1] == src.folder
		if underSkills || ownEntry {
			return true
		}
	}
	return false
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, re := range patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
