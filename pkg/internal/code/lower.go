// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"github.com/bickus/skill-verdict/pkg/internal/syntax"
)

type member struct {
	object   string
	property string
	index    bool
}

type spec struct {
	language      syntax.Language
	comprehension string
	funcs         map[string]bool
	blocks        map[string]bool
	simple        map[string]bool
	assigns       map[string][2]string
	loops         map[string]bool
	calls         map[string]string
	members       map[string]member
	idents        map[string]bool
	strings       map[string]bool
	fragments     map[string]bool
	delimiters    map[string]bool
	objects       map[string]bool
	skipped       map[string]bool
	imports       map[string]bool
	selves        map[string]bool
	comment       string
	keyword       string
	pair          string
	shorthand     string
	returns       string
	alias         string
	docstrings    bool
}

type lowerer struct {
	spec *spec
	tree *syntax.Tree
	unit *Unit
	base int
}

func (l *lowerer) span(n *syntax.Node) Span {
	return Span{Start: n.Start + l.base, End: n.End + l.base}
}

func (l *lowerer) text(n *syntax.Node) string {
	return l.tree.Text(n)
}

func (l *lowerer) broken(n *syntax.Node) bool {
	if !n.Error && !n.Missing {
		return false
	}
	l.unit.Broken = append(l.unit.Broken, l.span(n))
	return true
}

func (l *lowerer) block(n *syntax.Node, scope *Func) {
	for i, c := range n.NamedChildren() {
		if i == 0 && l.spec.docstrings && l.docstring(c) {
			l.unit.Comments = append(l.unit.Comments, l.span(c))
			continue
		}
		l.statement(c, scope)
	}
}

func (l *lowerer) docstring(n *syntax.Node) bool {
	if l.spec.strings[n.Kind] {
		return true
	}
	named := n.NamedChildren()
	return l.spec.simple[n.Kind] && len(named) == 1 && l.spec.strings[named[0].Kind]
}

func (l *lowerer) statement(n *syntax.Node, scope *Func) {
	switch {
	case l.broken(n):
	case n.Kind == l.spec.comment:
		l.unit.Comments = append(l.unit.Comments, l.span(n))
	case l.spec.funcs[n.Kind]:
		l.function(n, scope, "")
	case l.spec.blocks[n.Kind]:
		l.block(n, scope)
	case l.spec.simple[n.Kind]:
		l.simple(n, scope)
	case l.assigns(n):
		l.assign(n, n, scope)
	case l.spec.loops[n.Kind]:
		l.loop(n, scope)
	case l.compound(n):
		for _, c := range n.NamedChildren() {
			l.statement(c, scope)
		}
	default:
		l.expression(n, n, scope)
	}
}

func (l *lowerer) compound(n *syntax.Node) bool {
	for _, c := range n.Children {
		if l.spec.blocks[c.Kind] || l.spec.funcs[c.Kind] || l.spec.simple[c.Kind] {
			return true
		}
	}
	return false
}

func (l *lowerer) assigns(n *syntax.Node) bool {
	_, ok := l.spec.assigns[n.Kind]
	return ok
}

func (l *lowerer) loop(n *syntax.Node, scope *Func) {
	left, right := n.Child("left"), n.Child("right")
	if left != nil && right != nil {
		st := l.add(n, scope)
		st.Span = Span{Start: l.span(left).Start, End: l.span(right).End}
		st.Binds = l.targets(left)
		st.Value = l.collect(right, st)
	}
	for _, c := range n.NamedChildren() {
		if c != left && c != right {
			l.statement(c, scope)
		}
	}
}

func (l *lowerer) simple(n *syntax.Node, scope *Func) {
	if l.spec.imports[n.Kind] {
		n.Walk(func(c *syntax.Node) bool {
			if l.spec.idents[c.Kind] {
				l.unit.Imports[l.text(c)] = true
			}
			return true
		})
		return
	}
	handled := false
	for _, c := range n.NamedChildren() {
		if _, ok := l.spec.assigns[c.Kind]; ok {
			l.assign(c, n, scope)
			handled = true
		}
	}
	if !handled {
		l.expression(n, n, scope)
	}
}

func (l *lowerer) expression(n, whole *syntax.Node, scope *Func) {
	st := l.add(whole, scope)
	st.Return = n.Kind == l.spec.returns
	if n.Kind == l.spec.alias {
		named := n.NamedChildren()
		if len(named) == 2 {
			st.Binds = l.targets(named[1])
			st.Value = l.collect(named[0], st)
			return
		}
	}
	st.Value = l.collect(n, st)
}

func (l *lowerer) add(n *syntax.Node, scope *Func) *Stmt {
	st := &Stmt{Unit: l.unit, Span: l.span(n), Scope: scope}
	l.unit.Stmts = append(l.unit.Stmts, st)
	return st
}

func (l *lowerer) assign(n, whole *syntax.Node, scope *Func) {
	fields := l.spec.assigns[n.Kind]
	left, right := n.Child(fields[0]), n.Child(fields[1])
	if left == nil || right == nil {
		return
	}
	if l.spec.funcs[right.Kind] {
		l.function(right, scope, l.text(left))
		return
	}
	st := l.add(whole, scope)
	st.Binds = l.targets(left)
	right = l.chained(st, right)
	if fields[0] == "left" && n.Child("operator") != nil {
		st.Value = l.collect(n, st)
	} else {
		st.Value = l.collect(right, st)
	}
	if len(st.Calls) > 0 && st.Calls[0].Callee == "require" {
		for _, name := range st.Binds {
			l.unit.Imports[name] = true
		}
	}
}

func (l *lowerer) chained(st *Stmt, right *syntax.Node) *syntax.Node {
	for {
		inner, chained := l.spec.assigns[right.Kind]
		if !chained || right.Child(inner[0]) == nil || right.Child(inner[1]) == nil {
			return right
		}
		st.Binds = append(st.Binds, l.targets(right.Child(inner[0]))...)
		right = right.Child(inner[1])
	}
}

func (l *lowerer) targets(left *syntax.Node) []string {
	if _, ok := l.spec.members[left.Kind]; ok {
		name, _ := l.chain(left)
		if name == "" {
			return nil
		}
		return []string{name}
	}
	var out []string
	left.Walk(func(c *syntax.Node) bool {
		if l.spec.skipped[c.Kind] {
			return false
		}
		if l.spec.idents[c.Kind] || c.Kind == l.spec.shorthand+"_pattern" {
			out = append(out, l.text(c))
		}
		return true
	})
	return out
}

func (l *lowerer) function(n *syntax.Node, scope *Func, name string) {
	if name == "" {
		name = l.text(n.Child("name"))
	}
	f := &Func{Name: name, Span: l.span(n), Parent: scope}
	params := n.Child("parameters")
	if params == nil {
		params = n.Child("parameter")
	}
	if params != nil {
		f.Params = l.params(params)
	}
	l.unit.Funcs = append(l.unit.Funcs, f)
	body := n.Child("body")
	switch {
	case body == nil:
	case l.spec.blocks[body.Kind]:
		l.block(body, f)
	default:
		st := l.add(body, f)
		st.Return = true
		st.Value = l.collect(body, st)
	}
}

func (l *lowerer) params(n *syntax.Node) []string {
	if l.spec.idents[n.Kind] {
		return []string{l.text(n)}
	}
	var out []string
	for _, p := range n.NamedChildren() {
		if p.Kind == l.spec.comment {
			continue
		}
		out = append(out, l.firstIdent(p))
	}
	return out
}

func (l *lowerer) firstIdent(n *syntax.Node) string {
	found := ""
	n.Walk(func(c *syntax.Node) bool {
		if found != "" || l.spec.skipped[c.Kind] || c.Field == "value" || c.Field == "right" {
			return false
		}
		if l.spec.idents[c.Kind] || c.Kind == l.spec.shorthand+"_pattern" {
			found = l.text(c)
		}
		return found == ""
	})
	return found
}
