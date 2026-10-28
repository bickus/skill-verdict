// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

type Lang string

const (
	Python     Lang = "python"
	JavaScript Lang = "javascript"
	TypeScript Lang = "typescript"
	TSX        Lang = "tsx"
	Shell      Lang = "shell"
	PowerShell Lang = "powershell"
)

type Span struct {
	Start int
	End   int
}

func (s Span) Overlaps(o Span) bool {
	return s.Start < o.End && o.Start < s.End
}

func (s Span) Inside(o Span) bool {
	return s.Start >= o.Start && s.End <= o.End
}

type Ref struct {
	Name string
	Span Span
}

type Expr struct {
	Span Span
	Refs []Ref
}

func (e *Expr) Literal() bool {
	return len(e.Refs) == 0
}

func (e *Expr) merge(o Expr) {
	e.Refs = append(e.Refs, o.Refs...)
}

type Arg struct {
	Name string
	Expr Expr
}

type Call struct {
	Stmt     *Stmt
	Span     Span
	Callee   string
	Receiver string
	Args     []Arg
}

type Stmt struct {
	Unit   *Unit
	Span   Span
	Scope  *Func
	Binds  []string
	Value  Expr
	Calls  []*Call
	Return bool
}

type Func struct {
	Name   string
	Params []string
	Span   Span
	Parent *Func
}

func (f *Func) Within(outer *Func) bool {
	for at := f; at != nil; at = at.Parent {
		if at == outer {
			return true
		}
	}
	return outer == nil
}

type Unit struct {
	Path     string
	Lang     Lang
	Source   string
	Funcs    []*Func
	Stmts    []*Stmt
	Comments []Span
	Broken   []Span
	Imports  map[string]bool
	starts   []int
}

func newUnit(path string, lang Lang, source string) *Unit {
	return &Unit{Path: path, Lang: lang, Source: source, Imports: make(map[string]bool), starts: textutil.LineStarts(source)}
}

func (u *Unit) Line(offset int) int {
	return textutil.LineAt(u.starts, offset)
}

func (u *Unit) BrokenAt(s Span) bool {
	for _, b := range u.Broken {
		if b.Overlaps(s) {
			return true
		}
	}
	return false
}

const (
	maxCodeLines = 8
	maxLineChars = 300
)

func (u *Unit) Text(s Span) string {
	return numbered(u.Plain(s), u.Line(s.Start))
}

func (u *Unit) Plain(s Span) string {
	var b strings.Builder
	at := s.Start
	for _, c := range u.Comments {
		if c.End <= at || c.Start >= s.End {
			continue
		}
		b.WriteString(u.Source[at:max(at, c.Start)])
		at = min(c.End, s.End)
	}
	b.WriteString(u.Source[at:max(at, s.End)])
	return b.String()
}

func numbered(text string, first int) string {
	var out []string
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(out) == maxCodeLines {
			out = append(out, "...")
			break
		}
		out = append(out, fmt.Sprintf("L%d: %s", first+i, textutil.Cut(line, maxLineChars)))
	}
	return strings.Join(out, "\n")
}
