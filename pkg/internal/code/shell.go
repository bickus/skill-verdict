// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

var shellParams = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "@", "*"}

type shell struct {
	unit *Unit
	base int
}

func lowerShell(unit *Unit, source string, base int) {
	variant := syntax.LangBash
	if first, _, _ := strings.Cut(source, "\n"); strings.HasPrefix(first, "#!") && strings.Contains(first, "zsh") {
		variant = syntax.LangZsh
	}
	file, err := syntax.NewParser(syntax.KeepComments(true), syntax.Variant(variant)).Parse(strings.NewReader(source), unit.Path)
	if err != nil {
		unit.Broken = append(unit.Broken, Span{Start: base, End: base + len(source)})
		return
	}
	s := &shell{unit: unit, base: base}
	syntax.Walk(file, func(n syntax.Node) bool {
		if c, ok := n.(*syntax.Comment); ok {
			unit.Comments = append(unit.Comments, s.span(c))
		}
		return true
	})
	s.stmts(file.Stmts, nil)
}

func (s *shell) span(n syntax.Node) Span {
	return Span{Start: s.base + int(n.Pos().Offset()), End: s.base + int(n.End().Offset())}
}

func (s *shell) stmts(list []*syntax.Stmt, scope *Func) {
	for _, st := range list {
		s.stmt(st, scope)
	}
}

func (s *shell) stmt(st *syntax.Stmt, scope *Func) *Stmt {
	switch cmd := st.Cmd.(type) {
	case *syntax.FuncDecl:
		s.function(cmd, scope)
	case *syntax.BinaryCmd:
		return s.binary(cmd, scope)
	case *syntax.Block:
		s.stmts(cmd.Stmts, scope)
	case *syntax.Subshell:
		s.stmts(cmd.Stmts, scope)
	case *syntax.IfClause:
		s.conditional(cmd, scope)
	case *syntax.WhileClause:
		s.stmts(cmd.Cond, scope)
		s.stmts(cmd.Do, scope)
	case *syntax.ForClause:
		s.loop(cmd, scope)
	case *syntax.CaseClause:
		for _, item := range cmd.Items {
			s.stmts(item.Stmts, scope)
		}
	default:
		return s.leaf(st, scope)
	}
	return nil
}

func (s *shell) function(decl *syntax.FuncDecl, scope *Func) {
	if decl.Name == nil || decl.Body == nil {
		return
	}
	f := &Func{Name: decl.Name.Value, Params: shellParams, Span: s.span(decl), Parent: scope}
	s.unit.Funcs = append(s.unit.Funcs, f)
	s.stmt(decl.Body, f)
}

func (s *shell) binary(cmd *syntax.BinaryCmd, scope *Func) *Stmt {
	left := s.stmt(cmd.X, scope)
	right := s.stmt(cmd.Y, scope)
	piped := cmd.Op == syntax.Pipe || cmd.Op == syntax.PipeAll
	if piped && left != nil && right != nil && len(right.Calls) > 0 {
		call := right.Calls[0]
		call.Args = append(call.Args, Arg{Name: "stdin", Expr: left.Value})
		right.Value.merge(left.Value)
	}
	return right
}

func (s *shell) conditional(cmd *syntax.IfClause, scope *Func) {
	for at := cmd; at != nil; at = at.Else {
		s.stmts(at.Cond, scope)
		s.stmts(at.Then, scope)
	}
}

func (s *shell) loop(cmd *syntax.ForClause, scope *Func) {
	if iter, ok := cmd.Loop.(*syntax.WordIter); ok && iter.Name != nil {
		st := &Stmt{Unit: s.unit, Span: s.span(iter), Scope: scope, Binds: []string{iter.Name.Value}}
		st.Value = Expr{Span: st.Span}
		for _, w := range iter.Items {
			st.Value.merge(s.word(w, st))
		}
		s.unit.Stmts = append(s.unit.Stmts, st)
	}
	s.stmts(cmd.Do, scope)
}

func (s *shell) leaf(src *syntax.Stmt, scope *Func) *Stmt {
	st := &Stmt{Unit: s.unit, Span: s.span(src), Scope: scope}
	st.Value = Expr{Span: st.Span}
	s.unit.Stmts = append(s.unit.Stmts, st)
	s.command(src, st, &st.Value)
	return st
}

func (s *shell) command(src *syntax.Stmt, st *Stmt, e *Expr) {
	switch cmd := src.Cmd.(type) {
	case *syntax.CallExpr:
		s.call(cmd, st, e)
	case *syntax.DeclClause:
		s.assigns(cmd.Args, st, e, true)
	case *syntax.BinaryCmd:
		s.command(cmd.X, st, e)
		s.command(cmd.Y, st, e)
	case nil:
	default:
		e.merge(s.refs(cmd, st))
	}
	for _, r := range src.Redirs {
		s.redirect(r, st, e)
	}
}

func (s *shell) redirect(r *syntax.Redirect, st *Stmt, e *Expr) {
	if r.Word != nil {
		e.merge(s.word(r.Word, st))
	}
	if r.Hdoc == nil {
		return
	}
	body := s.word(r.Hdoc, st)
	e.merge(body)
	if len(st.Calls) > 0 {
		call := st.Calls[len(st.Calls)-1]
		call.Args = append(call.Args, Arg{Name: "heredoc", Expr: body})
	}
}

func (s *shell) assigns(list []*syntax.Assign, st *Stmt, e *Expr, bind bool) []Arg {
	var args []Arg
	for _, a := range list {
		if a.Name == nil {
			continue
		}
		value := Expr{Span: s.span(a)}
		if a.Value != nil {
			value = s.word(a.Value, st)
		}
		if a.Array != nil {
			value.merge(s.refs(a.Array, st))
		}
		e.merge(value)
		if bind && !a.Naked {
			st.Binds = append(st.Binds, a.Name.Value)
		}
		args = append(args, Arg{Name: a.Name.Value, Expr: value})
	}
	return args
}

func (s *shell) call(cmd *syntax.CallExpr, st *Stmt, e *Expr) {
	if len(cmd.Args) == 0 {
		s.assigns(cmd.Assigns, st, e, true)
		return
	}
	c := &Call{Stmt: st, Span: s.span(cmd), Callee: cmd.Args[0].Lit()}
	st.Calls = append(st.Calls, c)
	c.Args = s.assigns(cmd.Assigns, st, e, false)
	if c.Callee == "" {
		e.merge(s.word(cmd.Args[0], st))
	}
	flag := ""
	for _, w := range cmd.Args[1:] {
		arg := Arg{Expr: s.word(w, st)}
		lit := w.Lit()
		if flag != "" && !strings.HasPrefix(lit, "-") {
			arg.Name = flag
		}
		flag = ""
		if strings.HasPrefix(lit, "-") {
			flag = lit
		}
		c.Args = append(c.Args, arg)
		e.merge(arg.Expr)
		if c.Callee == "read" && lit != "" && !strings.HasPrefix(lit, "-") {
			st.Binds = append(st.Binds, lit)
		}
	}
}

func (s *shell) word(w *syntax.Word, st *Stmt) Expr {
	return s.refs(w, st)
}

func (s *shell) refs(n syntax.Node, st *Stmt) Expr {
	e := Expr{Span: s.span(n)}
	syntax.Walk(n, func(x syntax.Node) bool {
		switch v := x.(type) {
		case *syntax.ParamExp:
			if v.Param != nil {
				e.Refs = append(e.Refs, Ref{Name: v.Param.Value, Span: s.span(v)})
			}
		case *syntax.CmdSubst:
			s.nested(v.Stmts, st, &e)
			return false
		case *syntax.ProcSubst:
			s.nested(v.Stmts, st, &e)
			return false
		}
		return true
	})
	return e
}

func (s *shell) nested(list []*syntax.Stmt, st *Stmt, e *Expr) {
	for _, inner := range list {
		s.command(inner, st, e)
	}
}
