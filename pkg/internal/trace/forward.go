// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package trace

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/code"
)

const (
	maxDepth = 6
	maxNodes = 80
)

type Node struct {
	Name  string
	Call  string
	Code  string
	FedBy []*Source
	Feeds []*Node
}

type Trace struct {
	Subject   *Subject
	Feeds     []*Node
	Calls     []*Node
	Truncated bool
}

type visit struct {
	st   *code.Stmt
	name string
}

type tracer struct {
	prog   *Program
	trace  *Trace
	budget int
	seen   map[visit]bool
}

func (p *Program) Trace(s *Subject) *Trace {
	if t, ok := p.traces[s]; ok {
		return t
	}
	t := &tracer{prog: p, trace: &Trace{Subject: s}, budget: maxNodes, seen: make(map[visit]bool)}
	if s.Call != nil {
		t.trace.Feeds = []*Node{t.callNode(s.Call, s.Stmt)}
	} else {
		t.trace.Feeds = t.flow(s.Stmt, []code.Span{s.Span}, 0)
	}
	if len(t.trace.Feeds) == 0 {
		t.trace.Feeds = t.elsewhere(s)
	}
	p.traces[s] = t.trace
	return t.trace
}

func (t *tracer) elsewhere(s *Subject) []*Node {
	var out []*Node
	for _, r := range s.Stmt.Value.Refs {
		if constant(r.Name) && r.Span.Start < s.Span.End && r.Span.End > s.Span.Start {
			out = append(out, t.follow(r.Name, s.Stmt, 0)...)
		}
	}
	return out
}

func (t *tracer) spend() bool {
	if t.budget <= 0 {
		t.trace.Truncated = true
		return false
	}
	t.budget--
	return true
}

func (t *tracer) text(u *code.Unit, s code.Span) string {
	if u.Path != t.trace.Subject.Unit.Path {
		return u.Path + " " + u.Text(s)
	}
	return u.Text(s)
}

func (t *tracer) deep(depth int) bool {
	if depth > maxDepth {
		t.trace.Truncated = true
		return true
	}
	return false
}

func (t *tracer) flow(st *code.Stmt, spans []code.Span, depth int) []*Node {
	var out []*Node
	free := false
	done := make(map[*code.Call]bool)
	for _, sp := range spans {
		chain := consumers(st, sp)
		if len(chain) == 0 {
			free = true
			continue
		}
		if !done[chain[0]] {
			done[chain[0]] = true
			out = append(out, t.chain(chain, st, sp, depth)...)
		}
	}
	if free {
		out = append(out, t.bound(st, depth)...)
	}
	return out
}

func (t *tracer) chain(calls []*code.Call, st *code.Stmt, sp code.Span, depth int) []*Node {
	var first, last *Node
	for _, c := range calls {
		if !t.spend() {
			break
		}
		n := t.callNode(c, st)
		if f := t.prog.target(c.Callee); f != nil {
			n.Feeds = t.into(f, c, sp, depth)
		}
		if first == nil {
			first = n
		} else {
			last.Feeds = append(last.Feeds, n)
		}
		last = n
	}
	if first == nil {
		return nil
	}
	last.Feeds = append(last.Feeds, t.bound(st, depth)...)
	return []*Node{first}
}

func (t *tracer) callNode(c *code.Call, st *code.Stmt) *Node {
	n := &Node{Call: fmt.Sprintf("c%d", len(t.trace.Calls)+1), Code: t.text(st.Unit, c.Span)}
	t.trace.Calls = append(t.trace.Calls, n)
	n.FedBy = t.fedBy(c, st)
	return n
}

func (t *tracer) bound(st *code.Stmt, depth int) []*Node {
	var out []*Node
	if st.Return && st.Scope != nil {
		out = append(out, t.returned(st, depth)...)
	}
	for _, name := range st.Binds {
		if t.spend() {
			out = append(out, &Node{Name: name, Code: t.text(st.Unit, st.Span), Feeds: t.follow(name, st, depth+1)})
		}
	}
	return out
}

func (t *tracer) returned(st *code.Stmt, depth int) []*Node {
	var out []*Node
	for _, site := range t.prog.callSites(st.Scope) {
		if key := (visit{st: site.Stmt, name: st.Scope.Name}); !t.seen[key] {
			t.seen[key] = true
			out = append(out, t.flow(site.Stmt, []code.Span{site.Span}, depth+1)...)
		}
	}
	return out
}

func (t *tracer) follow(name string, def *code.Stmt, depth int) []*Node {
	if t.deep(depth) {
		return nil
	}
	var out []*Node
	for _, st := range t.prog.users(name, def) {
		key := visit{st: st, name: name}
		if t.seen[key] {
			continue
		}
		t.seen[key] = true
		out = append(out, t.flow(st, occurrences(st, name), depth)...)
	}
	return out
}

func occurrences(st *code.Stmt, names ...string) []code.Span {
	var out []code.Span
	for _, r := range st.Value.Refs {
		if slices.Contains(names, r.Name) {
			out = append(out, r.Span)
		}
	}
	return out
}

func (t *tracer) into(f *code.Func, c *code.Call, sp code.Span, depth int) []*Node {
	var names []string
	for i := range f.Params {
		if a := argument(c, f, i); a != nil && holds(a.Expr.Span, sp.Start) {
			names = append(names, f.Params[i])
		}
	}
	if len(names) == 0 || t.deep(depth) || !t.spend() {
		return nil
	}
	unit := t.prog.owners[f]
	if unit.Lang == code.Shell {
		names = append(names, "@", "*")
	}
	n := &Node{Name: names[0], Code: t.text(unit, signature(unit, f))}
	for _, inner := range unit.Stmts {
		key := visit{st: inner, name: names[0]}
		spans := occurrences(inner, names...)
		if inner.Scope == nil || !inner.Scope.Within(f) || len(spans) == 0 || t.seen[key] {
			continue
		}
		t.seen[key] = true
		n.Feeds = append(n.Feeds, t.flow(inner, spans, depth+1)...)
	}
	return []*Node{n}
}

func signature(u *code.Unit, f *code.Func) code.Span {
	end := f.Span.End
	if i := strings.IndexByte(u.Source[f.Span.Start:end], '\n'); i >= 0 {
		end = f.Span.Start + i
	}
	return code.Span{Start: f.Span.Start, End: end}
}
