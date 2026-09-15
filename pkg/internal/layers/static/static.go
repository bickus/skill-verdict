// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static

import (
	"context"
	"path"
	"regexp"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name       = "static"
	maxPerRule = 50
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
			scanArtifact(s, a, seen)
		case inventory.Symlink:
			scanLink(s, a)
		case inventory.Media, inventory.Unknown, inventory.Special, inventory.Directory:
		}
	}
	return pipeline.Report{}, nil
}

func applies(r rules.Compiled, a *inventory.Artifact) bool {
	if len(r.Files) > 0 && !config.MatchAny(r.Files, path.Base(inventory.OriginalPath(a.Path))) {
		return false
	}
	return (r.Scope == rules.Link) == (a.ContentType == inventory.Symlink)
}

type source struct {
	artifact *inventory.Artifact
	starts   []int
	lines    []string
	fenced   []bool
}

func scanArtifact(s *pipeline.State, a *inventory.Artifact, seen map[key]bool) {
	src := &source{artifact: a, starts: textutil.LineStarts(a.Content), lines: textutil.Lines(a.Content)}
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
		k := key{f.Rule, inventory.OriginalPath(f.Location.File), f.Location.Line}
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
		for _, m := range re.FindAllStringIndex(src.artifact.Content, -1) {
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
	line := textutil.LineAt(src.starts, m[0])
	if !allowed(a.ContentType, r.Scope, src.fenced != nil && src.fenced[line-1]) || matchesAny(r.Exclude, src.lines[line-1]) {
		return finding.Finding{}, false
	}
	f := r.Finding(finding.Location{File: a.Path, Line: line}, textutil.Evidence(a.Content[m[0]:m[1]]))
	if endLine := textutil.LineAt(src.starts, m[1]-1); endLine > line {
		f.Location.EndLine = endLine
	}
	return f, true
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, re := range patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
