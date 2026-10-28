// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const (
	around       = 5
	maxLineChars = 400
	anchorChars  = 40
)

type usagePolicy struct {
	Question string            `json:"question"`
	Focus    string            `json:"focus"`
	Options  map[string]string `json:"options"`
}

func loadUsagePolicy() usagePolicy {
	var p usagePolicy
	if err := json.Unmarshal(llmcall.Schema(files, "usage.json"), &p); err != nil {
		panic(err)
	}
	return p
}

func (p usagePolicy) question(rep *refs.Report, id string) decisions.Question {
	return decisions.Question{Options: p.Options, Instructions: map[string]string{
		"question":  p.Question,
		"reference": rep.Ref.Value,
		"place":     "place " + id,
		"focus":     p.Focus,
		"warning":   llmcall.Text(files, "warning.md"),
	}}
}

type block struct {
	id     string
	text   string
	places []int
}

func blocks(s *pipeline.State, rep *refs.Report) []*block {
	var out []*block
	byBody := make(map[string]*block)
	for j, loc := range rep.Ref.Locations {
		values := placeValues(s, rep, loc)
		if values == nil {
			continue
		}
		body := values["file"] + "\n" + values["line"] + "\n" + values["lines"] + "\n" + values["trace"]
		if b, ok := byBody[body]; ok {
			b.places = append(b.places, j)
			continue
		}
		values["id"] = rep.LocationID(j)
		b := &block{id: values["id"], text: llmcall.Render(files, "place.md", values), places: []int{j}}
		byBody[body] = b
		out = append(out, b)
	}
	return out
}

func referenceState(rep *refs.Report, list []*block) string {
	var urls, places []string
	for _, u := range rep.Ref.URLs {
		urls = append(urls, "url: "+u)
	}
	for _, b := range list {
		places = append(places, b.text)
	}
	return llmcall.Render(files, "reference.md", map[string]string{
		"reference": rep.Ref.Value,
		"urls":      strings.Join(urls, "\n"),
		"places":    strings.Join(places, "\n"),
	})
}

func placeValues(s *pipeline.State, rep *refs.Report, loc finding.Location) map[string]string {
	values, a := located(s, loc, rep.Ref.Value)
	if values != nil {
		values["trace"] = placeTrace(s, a, rep, loc.Line)
	}
	return values
}

func located(s *pipeline.State, loc finding.Location, anchor string) (map[string]string, *inventory.Artifact) {
	a := s.Skill.Artifact(loc.File)
	if a == nil {
		return nil, nil
	}
	lines := textutil.Lines(a.Content)
	if loc.Line < 1 || loc.Line > len(lines) {
		return nil, nil
	}
	return map[string]string{
		"file":  loc.File,
		"line":  strconv.Itoa(loc.Line),
		"spot":  spot(s, a, lines, loc.Line),
		"lines": window(lines, loc.Line, anchor),
	}, a
}

func window(lines []string, line int, anchor string) string {
	var out []string
	for n := max(line-around, 1); n <= min(line+around, len(lines)); n++ {
		text := textutil.Cut(lines[n-1], maxLineChars)
		if n == line {
			text = centered(lines[n-1], anchor)
		}
		out = append(out, fmt.Sprintf("L%d: %s", n, text))
	}
	return strings.Join(out, "\n")
}

func centered(line, anchor string) string {
	key := textutil.Cut(strings.TrimSpace(anchor), anchorChars)
	at := -1
	if key != "" && len(line) > maxLineChars {
		at = strings.Index(line, key)
	}
	if at < 0 {
		return textutil.Cut(line, maxLineChars)
	}
	start := min(max(at+len(key)/2-maxLineChars/2, 0), len(line)-maxLineChars)
	for start > 0 && !utf8.RuneStart(line[start]) {
		start--
	}
	return textutil.Cut(line[start:], maxLineChars)
}

func placeTrace(s *pipeline.State, a *inventory.Artifact, rep *refs.Report, line int) string {
	if s.Code == nil {
		return ""
	}
	span, ok := mention(a.Content, rep, line)
	if !ok {
		return ""
	}
	sub := s.Code.At(a.Path, span, false)
	if sub == nil {
		return ""
	}
	t := s.Code.Trace(sub)
	if len(t.Feeds) == 0 {
		return ""
	}
	return llmcall.Render(files, "place-trace.md", map[string]string{"trace": strings.TrimSpace(t.Text())})
}

func mention(content string, rep *refs.Report, line int) (code.Span, bool) {
	starts := textutil.LineStarts(content)
	start := starts[line-1]
	end := len(content)
	if line < len(starts) {
		end = starts[line]
	}
	for _, needle := range append([]string{rep.Ref.Value}, rep.Ref.URLs...) {
		if at := strings.Index(content[start:end], needle); at >= 0 {
			return code.Span{Start: start + at, End: start + at + len(needle)}, true
		}
	}
	return code.Span{}, false
}
