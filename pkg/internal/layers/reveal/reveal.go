// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

var revealed = inventory.RegisterOrigin(name, "Modified item from the skill bundle with concealment attempts removed. Created by the scanner itself")

type Layer struct{}

func (Layer) Name() string {
	return name
}

func (Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	allowed := s.Rules.ForLayer(name)
	count := len(s.Skill.Artifacts)
	for i := range count {
		if err := ctx.Err(); err != nil {
			return pipeline.Report{}, err
		}
		if !wanted(&s.Skill.Artifacts[i]) {
			continue
		}
		artifact(s, i, allowed)
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

func artifact(s *pipeline.State, i int, allowed map[string]bool) {
	a := s.Skill.Artifacts[i]
	text, hits := reveal(a.Content)
	if text == a.Content {
		return
	}
	path := inventory.RevealedPath(a.Path)
	s.Skill.Artifacts = append(s.Skill.Artifacts, inventory.Artifact{
		Path: path, Kind: inventory.Content, Origin: revealed, ContentType: a.ContentType, Size: int64(len(text)), Content: text,
	})
	found, capped := limit(finding.Dedup(findings(s.Rules, allowed, a, hits)))
	s.Record(found...)
	for _, id := range capped {
		s.Skill.Artifacts[i].Note(name + ": " + id + " capped at 50")
	}
}

func reveal(src string) (string, []hit) {
	text, offsets, hits := src, []int(nil), []hit(nil)
	if needsNormalize(src) {
		text, offsets, hits = normalize(src)
	}
	compacted, matched := compact(text)
	for _, sp := range matched {
		hits = append(hits, hit{spaced, source(offsets, sp.start), sourceEnd(src, offsets, sp.end)})
	}
	return compacted, hits
}

func source(offsets []int, i int) int {
	if offsets == nil {
		return i
	}
	return offsets[i]
}

func sourceEnd(src string, offsets []int, i int) int {
	if offsets == nil {
		return i
	}
	start := offsets[i-1]
	_, n := utf8.DecodeRuneInString(src[start:])
	return start + n
}

func findings(set *rules.Set, allowed map[string]bool, a inventory.Artifact, hits []hit) []finding.Finding {
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
	if h.rule == spaced {
		return textutil.Evidence(text[h.start:h.end])
	}
	var b strings.Builder
	for _, r := range text[wordStart(text, h.start):wordEnd(text, h.end)] {
		if dropped(r) || concealed(r) {
			fmt.Fprintf(&b, "<U+%04X>", r)
		} else {
			b.WriteRune(r)
		}
	}
	return textutil.Clip(b.String())
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
