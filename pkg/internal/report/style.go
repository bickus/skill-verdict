// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package report

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/scan"
)

const (
	MaxWidth = 120
	sep      = " • "
)

const (
	reset      = "\x1b[0m"
	bold       = "\x1b[1m"
	dim        = "\x1b[2m"
	red        = "\x1b[31m"
	green      = "\x1b[32m"
	yellow     = "\x1b[33m"
	cyan       = "\x1b[36m"
	boldRed    = "\x1b[1;31m"
	boldYellow = "\x1b[1;33m"
)

type Layout struct {
	Color bool
	Width int
}

type cell struct {
	shown string
	plain string
}

type piece struct {
	code string
	text string
}

func (p Layout) paint(code, s string) string {
	if !p.Color || code == "" || s == "" {
		return s
	}
	return code + s + reset
}

func verdictCode(v scan.Verdict) string {
	switch v {
	case scan.Block:
		return boldRed
	case scan.Incomplete:
		return boldYellow
	case scan.Review:
		return yellow
	case scan.Clean:
		return green
	}
	return ""
}

func severityCode(s finding.Severity) string {
	switch s {
	case finding.Critical:
		return boldRed
	case finding.High:
		return red
	case finding.Medium:
		return yellow
	case finding.Low:
		return ""
	}
	return ""
}

func statusCode(s refs.Status) string {
	switch s {
	case refs.Checked:
		return ""
	case refs.NotChecked:
		return yellow
	case refs.Failed:
		return red
	}
	return ""
}

func count(s string) int {
	return utf8.RuneCountInString(s)
}

func cut(s string, n int) string {
	if count(s) <= n {
		return s
	}
	i := 0
	for range n {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return s[:i]
}

func spaces(n int) string {
	if n < 1 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func safe(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s))
}

func ruled(p Layout, left cell) string {
	fill := p.Width - 1 - count(left.plain)
	if fill < 3 {
		fill = 3
	}
	return left.shown + " " + p.paint(dim, strings.Repeat("-", fill)) + "\n"
}

func wrapped(text string, first, rest int) []string {
	first, rest = max(first, 1), max(rest, 1)
	var out []string
	for n := first; count(text) > n; n = rest {
		at := len(cut(text, n))
		if i := strings.LastIndexByte(text[:at], ' '); i > 0 {
			at = i
		}
		out = append(out, strings.TrimRight(text[:at], " "))
		if text = strings.TrimLeft(text[at:], " "); text == "" {
			return out
		}
	}
	return append(out, text)
}

func pieces(b *strings.Builder, p Layout, ps ...piece) {
	for _, part := range ps {
		b.WriteString(p.paint(part.code, part.text))
	}
	b.WriteString("\n")
}

func wrap(b *strings.Builder, p Layout, indent int, ps ...piece) {
	col := 0
	for _, part := range ps {
		for i, text := range wrapped(part.text, p.Width-col, p.Width-indent) {
			if i > 0 {
				b.WriteString("\n")
				b.WriteString(spaces(indent))
				col = indent
			}
			b.WriteString(p.paint(part.code, text))
			col += count(text)
		}
	}
	b.WriteString("\n")
}

func line(b *strings.Builder, p Layout, code, text string) {
	if text == "" {
		return
	}
	pieces(b, p, piece{code, text})
}

func filled(ps []piece) bool {
	for _, part := range ps {
		if part.text != "" {
			return true
		}
	}
	return false
}

func keyed(b *strings.Builder, p Layout, label string, ps ...piece) {
	if !filled(ps) {
		return
	}
	pieces(b, p, append([]piece{{"", label + ": "}}, ps...)...)
}

func field(b *strings.Builder, p Layout, label string, ps ...piece) {
	if !filled(ps) {
		return
	}
	head := piece{"", label + ":" + spaces(labelWidth-count(label)+1)}
	wrap(b, p, labelWidth+2, append([]piece{head}, ps...)...)
}
