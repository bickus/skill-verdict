// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

const (
	around          = 5
	before          = 3
	maxBlockLines   = 40
	keptLines       = 20
	maxLineChars    = 400
	anchorChars     = 40
	maxHeadingLevel = 6
	noFence         = "no"
	plainFence      = "yes"
)

var listItem = regexp.MustCompile(`^\s*(?:[-*+]|\d{1,9}[.)])(?:\s|$)`)

type lineKind int

const (
	blank lineKind = iota
	fenceLine
	heading
	table
	item
	text
)

type block struct {
	from        int
	first       int
	last        int
	fence       string
	section     string
	frontmatter bool
}

func (r *run) frame(loc finding.Location, anchor string) (string, bool) {
	a, lines, b, ok := r.blockAt(loc)
	if !ok {
		return "", false
	}
	values := map[string]string{
		"file":    a.Path,
		"role":    role(r.state.Skill, a, b.frontmatter),
		"fence":   b.fence,
		"section": "",
		"lines":   b.render(lines, loc.Line, anchor),
	}
	if b.section != "" {
		values["section"] = "section: " + b.section
	}
	return strings.TrimSuffix(llmcall.Render(files, "frame.md", values), "\n"), true
}

func (r *run) blockAt(loc finding.Location) (*inventory.Artifact, []string, block, bool) {
	a := r.state.Skill.Artifact(loc.File)
	if a == nil {
		return nil, nil, block{}, false
	}
	lines := textutil.Lines(a.Content)
	if loc.Line < 1 || loc.Line > len(lines) {
		return nil, nil, block{}, false
	}
	if a.ContentType == inventory.Code {
		return a, lines, r.function(a, lines, loc.Line), true
	}
	return a, lines, textBlock(lines, loc.Line), true
}

func role(skill *inventory.Skill, a *inventory.Artifact, frontmatter bool) string {
	entry := llmcall.Entry(skill)
	switch {
	case a == entry && frontmatter:
		return "frontmatter"
	case a == entry:
		return "entry"
	case a.ContentType == inventory.Code:
		return "script"
	case entry != nil && strings.Contains(entry.Content, a.Path):
		return "referenced-by-entry"
	case strings.HasPrefix(strings.ToLower(path.Base(a.Path)), "readme"):
		return "readme"
	}
	return "other"
}

func (r *run) function(a *inventory.Artifact, lines []string, n int) block {
	window := block{from: max(n-around, 1), first: max(n-around, 1), last: min(n+around, len(lines)), fence: noFence}
	if r.state.Code == nil {
		return window
	}
	starts := textutil.LineStarts(a.Content)
	line := lines[n-1]
	offset := starts[n-1] + len(line) - len(strings.TrimLeft(line, " \t"))
	var best *code.Func
	for _, u := range r.state.Code.Units {
		if u.Path != a.Path {
			continue
		}
		for _, f := range u.Funcs {
			inside := f.Span.Start <= offset && offset < f.Span.End
			if inside && (best == nil || f.Span.End-f.Span.Start < best.Span.End-best.Span.Start) {
				best = f
			}
		}
	}
	if best == nil {
		return window
	}
	first, last := textutil.LineAt(starts, best.Span.Start), textutil.LineAt(starts, best.Span.End-1)
	return block{from: first, first: first, last: last, fence: noFence}
}

func textBlock(lines []string, n int) block {
	first := body(lines)
	if n < first {
		return block{from: 1, first: 1, last: min(first-1, len(lines)), fence: noFence, frontmatter: true}
	}
	p := &position{}
	for i := first; i < n; i++ {
		p.read(i, lines[i-1])
	}
	b := block{fence: noFence, section: strings.Join(p.headings, " > ")}
	mark, info := code.Fence(lines[n-1])
	switch {
	case p.marker != "":
		b.first, b.last, b.fence = p.opened, closing(lines, n, p.marker), fenceName(p.language)
	case mark != "":
		b.first, b.last, b.fence = n, closing(lines, n+1, mark), fenceName(language(info))
	default:
		b.first, b.last = paragraph(lines, n, first)
	}
	b.from = max(b.first-before, 1)
	return b
}

func body(lines []string) int {
	if strings.TrimSpace(lines[0]) != "---" {
		return 1
	}
	end := slices.IndexFunc(lines[1:], func(l string) bool { return strings.TrimSpace(l) == "---" })
	if end < 0 {
		return len(lines) + 1
	}
	return end + 3
}

func closing(lines []string, from int, marker string) int {
	for i := from; i <= len(lines); i++ {
		if mark, info := code.Fence(lines[i-1]); mark != "" && strings.HasPrefix(mark, marker) && strings.TrimSpace(info) == "" {
			return i
		}
	}
	return len(lines)
}

func language(info string) string {
	lang, _, _ := strings.Cut(strings.TrimSpace(info), " ")
	return lang
}

func fenceName(lang string) string {
	if lang == "" {
		return plainFence
	}
	return lang
}

func paragraph(lines []string, n, first int) (int, int) {
	switch kind(lines[n-1]) {
	case blank, fenceLine, heading, table:
		return n, n
	case item, text:
	}
	start := n
	if kind(lines[n-1]) == text {
		for start > first && kind(lines[start-2]) == text {
			start--
		}
		if start > first && kind(lines[start-2]) == item {
			start--
		}
	}
	end := n
	for end < len(lines) && kind(lines[end]) == text {
		end++
	}
	return start, end
}

func kind(line string) lineKind {
	trimmed := strings.TrimSpace(line)
	mark, _ := code.Fence(line)
	switch {
	case trimmed == "":
		return blank
	case mark != "":
		return fenceLine
	case headingLevel(line) > 0:
		return heading
	case strings.HasPrefix(trimmed, "|"):
		return table
	case listItem.MatchString(line):
		return item
	}
	return text
}

func headingLevel(line string) int {
	level := len(line) - len(strings.TrimLeft(line, "#"))
	if level == 0 || level > maxHeadingLevel || !strings.HasPrefix(line[level:], " ") {
		return 0
	}
	return level
}

type position struct {
	headings []string
	levels   []int
	marker   string
	language string
	opened   int
}

func (p *position) read(i int, line string) {
	mark, info := code.Fence(line)
	switch {
	case p.marker == "" && mark != "":
		p.marker, p.opened, p.language = mark, i, language(info)
	case p.marker != "" && strings.HasPrefix(mark, p.marker) && strings.TrimSpace(info) == "":
		p.marker = ""
	case p.marker == "":
		p.heading(line)
	}
}

func (p *position) heading(line string) {
	level := headingLevel(line)
	if level == 0 {
		return
	}
	for len(p.levels) > 0 && p.levels[len(p.levels)-1] >= level {
		p.levels = p.levels[:len(p.levels)-1]
		p.headings = p.headings[:len(p.headings)-1]
	}
	p.levels = append(p.levels, level)
	p.headings = append(p.headings, strings.TrimSpace(line[level:]))
}

func (b block) render(lines []string, n int, anchor string) string {
	parts := [][2]int{}
	if b.from < b.first {
		parts = append(parts, [2]int{b.from, b.first - 1})
	}
	parts = append(parts, cut(b.first, b.last, n)...)
	var out []string
	last := 0
	for _, part := range parts {
		if last > 0 && part[0] > last+1 {
			out = append(out, "...")
		}
		for i := max(part[0], last+1); i <= part[1]; i++ {
			out = append(out, numbered(lines[i-1], i, n, anchor))
		}
		last = max(last, part[1])
	}
	return strings.Join(out, "\n")
}

func cut(first, last, n int) [][2]int {
	if last-first+1 <= maxBlockLines {
		return [][2]int{{first, last}}
	}
	head, tail := [2]int{first, first + keptLines - 1}, [2]int{last - keptLines + 1, last}
	if n <= head[1] || n >= tail[0] {
		return [][2]int{head, tail}
	}
	return [][2]int{head, {max(n-around, head[1]+1), min(n+around, tail[0]-1)}, tail}
}

func numbered(line string, i, n int, anchor string) string {
	if i == n {
		return fmt.Sprintf(">> L%d: %s", i, centered(line, anchor))
	}
	return fmt.Sprintf("L%d: %s", i, textutil.Cut(line, maxLineChars))
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
