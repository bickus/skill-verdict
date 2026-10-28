// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package trace

import (
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/code"
)

type Subject struct {
	Unit  *code.Unit
	Stmt  *code.Stmt
	Span  code.Span
	Call  *code.Call
	Match string
}

type place struct {
	path  string
	start int
	call  bool
}

type Program struct {
	Units    []*code.Unit
	subjects map[place]*Subject
	funcs    map[string][]*code.Func
	owners   map[*code.Func]*code.Unit
	traces   map[*Subject]*Trace
}

const (
	maxCallSites = 3
	constantMin  = 4
)

func NewProgram(units []*code.Unit) *Program {
	p := &Program{Units: units, subjects: make(map[place]*Subject), funcs: make(map[string][]*code.Func),
		owners: make(map[*code.Func]*code.Unit), traces: make(map[*Subject]*Trace)}
	for _, u := range units {
		for _, f := range u.Funcs {
			p.owners[f] = u
			if f.Name != "" {
				p.funcs[f.Name] = append(p.funcs[f.Name], f)
			}
		}
	}
	return p
}

func (p *Program) At(path string, match code.Span, call bool) *Subject {
	st := p.statement(path, match.Start)
	if st == nil || st.Unit.BrokenAt(st.Span) {
		return nil
	}
	if c := smallest(st, match.Start); call && c != nil && !takes(c, match) {
		return p.callSubject(c)
	}
	return p.valueSubject(st, code.Span{Start: match.Start, End: min(match.End, st.Span.End)})
}

func (p *Program) valueSubject(st *code.Stmt, span code.Span) *Subject {
	key := place{st.Unit.Path, span.Start, false}
	if s, ok := p.subjects[key]; ok {
		return s
	}
	s := &Subject{Unit: st.Unit, Stmt: st, Span: span, Match: st.Unit.Source[span.Start:span.End]}
	p.subjects[key] = s
	return s
}

func (p *Program) callSubject(c *code.Call) *Subject {
	key := place{c.Stmt.Unit.Path, c.Span.Start, true}
	if s, ok := p.subjects[key]; ok {
		return s
	}
	s := &Subject{Unit: c.Stmt.Unit, Stmt: c.Stmt, Span: c.Span, Call: c, Match: c.Stmt.Unit.Plain(c.Span)}
	p.subjects[key] = s
	return s
}

func (p *Program) statement(path string, offset int) *code.Stmt {
	var best *code.Stmt
	for _, u := range p.Units {
		if u.Path != path {
			continue
		}
		for _, st := range u.Stmts {
			if holds(st.Span, offset) && (best == nil || size(st.Span) < size(best.Span)) {
				best = st
			}
		}
	}
	return best
}

func holds(s code.Span, offset int) bool {
	return offset >= s.Start && offset < s.End
}

func size(s code.Span) int {
	return s.End - s.Start
}

func smallest(st *code.Stmt, offset int) *code.Call {
	var best *code.Call
	for _, c := range st.Calls {
		if holds(c.Span, offset) && (best == nil || size(c.Span) < size(best.Span)) {
			best = c
		}
	}
	return best
}

func takes(c *code.Call, sp code.Span) bool {
	return slices.ContainsFunc(c.Args, func(a code.Arg) bool { return holds(a.Expr.Span, sp.Start) })
}

func consumers(st *code.Stmt, sp code.Span) []*code.Call {
	var out []*code.Call
	for _, c := range st.Calls {
		if takes(c, sp) {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b *code.Call) int { return size(a.Span) - size(b.Span) })
	return out
}

func constant(name string) bool {
	return len(name) >= constantMin && name == strings.ToUpper(name) && strings.ToLower(name) != name
}

func wide(name string, def *code.Stmt) bool {
	return def.Scope == nil || def.Unit.Lang == code.Shell || strings.HasPrefix(name, "self.") || strings.HasPrefix(name, "this.")
}

func refers(e code.Expr, name string) bool {
	return slices.ContainsFunc(e.Refs, func(r code.Ref) bool { return r.Name == name })
}

func (p *Program) shares(other, home *code.Unit, name string) bool {
	return other.Path == home.Path || other.Imports[name] || constant(name)
}

func (p *Program) users(name string, def *code.Stmt) []*code.Stmt {
	var out []*code.Stmt
	global := wide(name, def)
	for _, u := range p.Units {
		if u != def.Unit && (!global || !p.shares(u, def.Unit, name) || p.binds(u, name)) {
			continue
		}
		for _, st := range u.Stmts {
			if st != def && refers(st.Value, name) && (global || after(st, def)) {
				out = append(out, st)
			}
		}
	}
	return out
}

func after(st, def *code.Stmt) bool {
	return st.Scope != nil && st.Scope.Within(def.Scope) && st.Span.Start > def.Span.Start
}

func (p *Program) binds(u *code.Unit, name string) bool {
	return slices.ContainsFunc(u.Stmts, func(st *code.Stmt) bool { return st.Scope == nil && slices.Contains(st.Binds, name) })
}

func (p *Program) definition(name string, at *code.Stmt) *code.Stmt {
	if best := local(name, at); best != nil {
		return best
	}
	for _, u := range p.Units {
		if u == at.Unit || !p.shares(u, at.Unit, name) {
			continue
		}
		for _, st := range u.Stmts {
			if st.Scope == nil && slices.Contains(st.Binds, name) {
				return st
			}
		}
	}
	return nil
}

func local(name string, at *code.Stmt) *code.Stmt {
	var best *code.Stmt
	for _, st := range at.Unit.Stmts {
		if st == at || !slices.Contains(st.Binds, name) {
			continue
		}
		before := st.Span.Start < at.Span.Start
		if visible(st, at) && (before || st.Scope == nil) && (best == nil || before) {
			best = st
		}
	}
	return best
}

func visible(st, at *code.Stmt) bool {
	return at.Unit.Lang == code.Shell || st.Scope == nil || (at.Scope != nil && at.Scope.Within(st.Scope))
}

func (p *Program) param(name string, at *code.Stmt) (*code.Func, int) {
	for f := at.Scope; f != nil; f = f.Parent {
		if i := slices.Index(f.Params, name); i >= 0 {
			return f, i
		}
	}
	return nil, -1
}

func (p *Program) target(callee string) *code.Func {
	name := callee[strings.LastIndex(callee, ".")+1:]
	if found := p.funcs[name]; len(found) > 0 {
		return found[0]
	}
	return nil
}

func (p *Program) callSites(f *code.Func) []*code.Call {
	var out []*code.Call
	if f.Name == "" {
		return nil
	}
	for _, u := range p.Units {
		for _, st := range u.Stmts {
			for _, c := range st.Calls {
				if p.target(c.Callee) == f && len(out) < maxCallSites {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

func argument(c *code.Call, f *code.Func, index int) *code.Arg {
	positional := 0
	for i := range c.Args {
		a := &c.Args[i]
		named := a.Name != "" && !strings.HasPrefix(a.Name, "-")
		if named && a.Name == f.Params[index] {
			return a
		}
		if named {
			continue
		}
		if positional == index-offset(f) {
			return a
		}
		positional++
	}
	return nil
}

func offset(f *code.Func) int {
	if len(f.Params) > 0 && (f.Params[0] == "self" || f.Params[0] == "cls") {
		return 1
	}
	return 0
}
