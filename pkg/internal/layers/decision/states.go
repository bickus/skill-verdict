// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/internal/trace"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	entryReserve  = 2000
	maxFactPlaces = 5
	noLine        = "the line was not found"
	noCall        = "the trace has no call that takes the value"
	noEntry       = "the skill has no SKILL.md"
	noReference   = "the finding names no checked reference"
)

func (r *run) stateOf(subject rules.Subject, f finding.Finding) (string, string) {
	switch subject {
	case rules.Entry:
		return r.entry(f)
	case rules.Facts:
		return r.facts(f)
	case rules.Frame, rules.Places, rules.Trace:
	}
	text, ok := r.frame(f.Location, anchor(f))
	if !ok {
		return "", noLine
	}
	return llmcall.Render(files, "block.md", map[string]string{"frame": text}), ""
}

func anchor(f finding.Finding) string {
	if f.Reference != "" {
		return f.Reference
	}
	return f.Evidence
}

func (r *run) traceState(sub *trace.Subject) (string, string) {
	t := r.state.Code.Trace(sub)
	if len(t.Calls) == 0 {
		return "", noCall
	}
	frame := ""
	if a := r.state.Skill.Artifact(sub.Unit.Path); a != nil {
		line := textutil.LineAt(textutil.LineStarts(a.Content), sub.Span.Start)
		frame, _ = r.frame(finding.Location{File: a.Path, Line: line}, sub.Match)
	}
	return t.State(frame), ""
}

func traceLabel(i int) string {
	return fmt.Sprintf("T%d", i+1)
}

func (r *run) entry(f finding.Finding) (string, string) {
	a := llmcall.Entry(r.state.Skill)
	if a == nil {
		return "", noEntry
	}
	lines := textutil.Lines(strings.TrimSuffix(a.Content, "\n"))
	limit := r.layer.Config.MaxStateChars - entryReserve
	var b strings.Builder
	shown := 0
	for i, line := range lines {
		next := fmt.Sprintf("L%d: %s\n", i+1, line)
		if b.Len()+len(next) > limit {
			break
		}
		b.WriteString(next)
		shown++
	}
	if shown < len(lines) {
		fmt.Fprintf(&b, "[cut: %d more lines]\n", len(lines)-shown)
	}
	return llmcall.Render(files, "entry.md", map[string]string{
		"about":   fmt.Sprintf("%s:%d %s", f.Location.File, f.Location.Line, f.Evidence),
		"content": strings.TrimSuffix(b.String(), "\n"),
	}), ""
}

func (r *run) facts(f finding.Finding) (string, string) {
	rep := r.reference(f.Reference)
	if rep == nil {
		return "", noReference
	}
	var places []string
	for _, loc := range rep.Ref.Locations {
		if len(places) == maxFactPlaces {
			break
		}
		if text, ok := r.frame(loc, rep.Ref.Value); ok {
			places = append(places, text)
		}
	}
	return llmcall.Render(files, "facts.md", map[string]string{
		"brief":  refcheck.Section(r.state.Skill, *rep),
		"places": strings.Join(places, "\n\n"),
	}), ""
}

func (r *run) reference(value string) *refs.Report {
	for i := range r.state.Refs {
		if rep := &r.state.Refs[i]; value != "" && rep.Ref.Value == value {
			return rep
		}
	}
	return nil
}
