// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/syntax"
)

const constantMin = 4

func (l *lowerer) collect(n *syntax.Node, st *Stmt) Expr {
	e := Expr{Span: l.span(n)}
	var locals []string
	l.gather(n, st, &e, &locals)
	if len(locals) > 0 {
		e.Refs = slices.DeleteFunc(e.Refs, func(r Ref) bool { return slices.Contains(locals, r.Name) })
	}
	return e
}

func (l *lowerer) gather(n *syntax.Node, st *Stmt, e *Expr, locals *[]string) {
	if l.leaf(n, st, e, locals) {
		return
	}
	if field, ok := l.spec.calls[n.Kind]; ok {
		l.call(n, field, st, e, locals)
		return
	}
	if m, ok := l.spec.members[n.Kind]; ok {
		l.member(n, m, st, e, locals)
		return
	}
	if n.Kind == l.spec.keyword || n.Kind == l.spec.pair {
		if v := n.Child("value"); v != nil {
			l.gather(v, st, e, locals)
		}
		return
	}
	bound := n.Kind == l.spec.comprehension
	for _, c := range n.Children {
		if bound && c.Field == "left" {
			*locals = append(*locals, l.targets(c)...)
			continue
		}
		l.gather(c, st, e, locals)
	}
}

func (l *lowerer) leaf(n *syntax.Node, st *Stmt, e *Expr, locals *[]string) bool {
	switch {
	case l.broken(n):
		return false
	case n.Kind == l.spec.comment:
		l.unit.Comments = append(l.unit.Comments, l.span(n))
	case l.spec.skipped[n.Kind]:
	case l.spec.funcs[n.Kind]:
		l.function(n, st.Scope, "")
	case l.spec.idents[n.Kind], n.Kind == l.spec.shorthand:
		e.Refs = append(e.Refs, Ref{Name: l.text(n), Span: l.span(n)})
	case l.spec.strings[n.Kind]:
		l.str(n, st, e, locals)
	default:
		return false
	}
	return true
}

func (l *lowerer) str(n *syntax.Node, st *Stmt, e *Expr, locals *[]string) {
	for _, c := range n.NamedChildren() {
		if !l.spec.fragments[c.Kind] && !l.spec.delimiters[c.Kind] {
			l.gather(c, st, e, locals)
		}
	}
}

func (l *lowerer) stringValue(n *syntax.Node) (string, bool) {
	if n == nil || !l.spec.strings[n.Kind] {
		return "", false
	}
	var b strings.Builder
	for _, c := range n.NamedChildren() {
		if l.spec.delimiters[c.Kind] {
			continue
		}
		if !l.spec.fragments[c.Kind] {
			return "", false
		}
		b.WriteString(l.text(c))
	}
	return b.String(), b.Len() > 0
}

func (l *lowerer) chain(n *syntax.Node) (string, *syntax.Node) {
	var props []string
	at := n
	for {
		m, ok := l.spec.members[at.Kind]
		if !ok {
			break
		}
		object := at.Child(m.object)
		if object == nil {
			return "", at
		}
		if prop := at.Child(m.property); prop != nil && !m.index {
			props = append(props, l.text(prop))
		} else {
			props = append(props, "")
		}
		at = object
	}
	if !l.spec.idents[at.Kind] {
		return "", at
	}
	head := l.text(at)
	slices.Reverse(props)
	if l.spec.selves[head] && len(props) > 0 && props[0] != "" {
		return head + "." + props[0], nil
	}
	return head, nil
}

func (l *lowerer) member(n *syntax.Node, m member, st *Stmt, e *Expr, locals *[]string) {
	name, rest := l.chain(n)
	if name != "" {
		e.Refs = append(e.Refs, Ref{Name: name, Span: l.span(n)})
	}
	if rest != nil {
		l.gather(rest, st, e, locals)
	}
	if prop := n.Child(m.property); prop != nil {
		if m.index {
			l.gather(prop, st, e, locals)
		} else if constant(l.text(prop)) {
			e.Refs = append(e.Refs, Ref{Name: l.text(prop), Span: l.span(prop)})
		}
	}
	for at := n.Child(m.object); at != nil; {
		inner, ok := l.spec.members[at.Kind]
		if !ok {
			break
		}
		if idx := at.Child(inner.property); idx != nil && inner.index {
			l.gather(idx, st, e, locals)
		}
		at = at.Child(inner.object)
	}
}

func constant(name string) bool {
	return len(name) >= constantMin && name == strings.ToUpper(name) && strings.ToLower(name) != name
}

func (l *lowerer) callee(fn *syntax.Node) (callee, receiver string) {
	if fn == nil {
		return "", ""
	}
	if l.spec.idents[fn.Kind] {
		return l.text(fn), ""
	}
	m, ok := l.spec.members[fn.Kind]
	if !ok || m.index {
		return "", ""
	}
	name, rest := l.chain(fn)
	if rest != nil || name == "" {
		return l.text(fn.Child(m.property)), ""
	}
	return strings.Join(strings.Fields(l.text(fn)), ""), name
}

func (l *lowerer) call(n *syntax.Node, field string, st *Stmt, e *Expr, locals *[]string) {
	fn := n.Child(field)
	c := &Call{Stmt: st, Span: l.span(n)}
	c.Callee, c.Receiver = l.callee(fn)
	st.Calls = append(st.Calls, c)
	switch {
	case fn == nil:
	case c.Receiver != "":
		e.Refs = append(e.Refs, Ref{Name: c.Receiver, Span: l.span(fn)})
	case !l.spec.idents[fn.Kind]:
		l.gather(fn, st, e, locals)
	}
	args := n.Child("arguments")
	if args == nil {
		return
	}
	for _, a := range args.NamedChildren() {
		if a.Kind == l.spec.comment {
			continue
		}
		for _, arg := range l.args(a, st) {
			c.Args = append(c.Args, arg)
			e.merge(arg.Expr)
		}
	}
}

func (l *lowerer) args(a *syntax.Node, st *Stmt) []Arg {
	if a.Kind == l.spec.keyword {
		if v := a.Child("value"); v != nil {
			return []Arg{{Name: l.text(a.Child("name")), Expr: l.collect(v, st)}}
		}
	}
	if !l.spec.objects[a.Kind] {
		return []Arg{{Expr: l.collect(a, st)}}
	}
	var out []Arg
	for _, p := range a.NamedChildren() {
		key, value := p.Child("key"), p.Child("value")
		switch {
		case p.Kind == l.spec.comment:
		case key != nil && value != nil:
			name, ok := l.stringValue(key)
			if !ok {
				name = l.text(key)
			}
			out = append(out, Arg{Name: name, Expr: l.collect(value, st)})
		default:
			out = append(out, Arg{Name: l.text(p), Expr: l.collect(p, st)})
		}
	}
	return out
}
