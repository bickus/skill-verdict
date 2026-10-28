// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static

import (
	"slices"

	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/trace"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const maxTraces = 30

var traced = inventory.RegisterOrigin("trace", "Paths of the values findings point at through the skill's code, read from the syntax trees. Created by the scanner itself")

func (src *source) link(r rules.Compiled, match code.Span, line int) {
	s := src.state
	from, ok := r.Decision.TraceFrom()
	if s.Code == nil || !ok {
		return
	}
	var subjects []*trace.Subject
	if sub := s.Code.At(src.artifact.Path, match, from == rules.FromCall); sub != nil {
		subjects = append(subjects, sub)
	}
	s.Link(r.ID, src.artifact.Path, line, subjects)
}

func brief(s *pipeline.State) {
	var traces []*trace.Trace
	var seen []*trace.Subject
	for _, f := range s.Findings {
		for _, sub := range s.Linked(f) {
			if slices.Contains(seen, sub) {
				continue
			}
			seen = append(seen, sub)
			if len(traces) < maxTraces {
				traces = append(traces, s.Code.Trace(sub))
			}
		}
	}
	if len(traces) == 0 {
		return
	}
	text := trace.Brief(traces, len(seen)-len(traces))
	s.Skill.Artifacts = append(s.Skill.Artifacts, inventory.Artifact{
		Path: trace.BriefPath, Kind: inventory.Brief, Origin: traced, ContentType: inventory.Markdown, Size: int64(len(text)), Content: text,
	})
}
