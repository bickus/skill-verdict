// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	"context"
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name       = "reveal"
	maxPerRule = 50
)

type Layer struct{}

func (Layer) Name() string {
	return name
}

func (Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	allowed := s.Rules.ForLayer(name)
	for i := range s.Skill.Artifacts {
		if err := ctx.Err(); err != nil {
			return pipeline.Report{}, err
		}
		if a := &s.Skill.Artifacts[i]; wanted(a) && needsNormalize(a.Content) {
			artifact(s, a, allowed)
		}
	}
	return pipeline.Report{}, nil
}

func wanted(a *inventory.Artifact) bool {
	if a.Kind != inventory.Content || a.Origin != inventory.Original {
		return false
	}
	switch a.ContentType {
	case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data:
		return true
	case inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special, inventory.Directory:
	}
	return false
}

func artifact(s *pipeline.State, a *inventory.Artifact, allowed map[string]bool) {
	p := normalize(a.Content)
	if text := p.out.String(); text != a.Content {
		a.Clean, a.Shifts = text, p.shifts
	}
	found, capped := limit(finding.Dedup(findings(s.Rules, allowed, a, p.hits)))
	s.Record(found...)
	for _, id := range capped {
		a.Note(name + ": " + id + " capped at 50")
	}
}

func findings(set *rules.Set, allowed map[string]bool, a *inventory.Artifact, hits []hit) []finding.Finding {
	starts := textutil.LineStarts(a.Content)
	var out []finding.Finding
	for _, h := range hits {
		if !allowed[h.rule] {
			continue
		}
		loc := finding.Location{File: a.Path, Line: textutil.LineAt(starts, h.start)}
		out = append(out, set.ByID[h.rule].Finding(loc, evidence(a.Content, h)))
	}
	return out
}

func evidence(text string, h hit) string {
	var written, read strings.Builder
	for _, r := range text[wordStart(text, h.start):wordEnd(text, h.end)] {
		switch {
		case dropped(r):
			fmt.Fprintf(&written, "<U+%04X>", r)
		case concealed(r):
			fmt.Fprintf(&written, "<U+%04X>", r)
			_, mapped := fold(r)
			read.WriteString(mapped)
		default:
			written.WriteRune(r)
			read.WriteRune(r)
		}
	}
	if h.rule == lookAlike {
		return textutil.Clip(written.String() + " reads as " + read.String())
	}
	return textutil.Clip(written.String())
}

func limit(fs []finding.Finding) ([]finding.Finding, []string) {
	counts := make(map[string]int)
	var kept []finding.Finding
	var capped []string
	for _, f := range fs {
		counts[f.Rule]++
		switch {
		case counts[f.Rule] <= maxPerRule:
			kept = append(kept, f)
		case counts[f.Rule] == maxPerRule+1:
			capped = append(capped, f.Rule)
		}
	}
	return kept, capped
}
