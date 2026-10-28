// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/syntax"
)

const envDrive = "env:"

var (
	powershellScopes = []string{"global:", "local:", "private:", "script:", "using:", "workflow:"}
	powershellValues = []string{"null", "true", "false"}
)

type powershell struct {
	tree *syntax.Tree
	unit *Unit
	base int
}

func (p *powershell) span(n *syntax.Node) Span {
	return Span{Start: n.Start + p.base, End: n.End + p.base}
}

func (p *powershell) lower(n *syntax.Node) string {
	return strings.ToLower(p.tree.Text(n))
}

func (p *powershell) broken(n *syntax.Node) bool {
	if !n.Error && !n.Missing {
		return false
	}
	p.unit.Broken = append(p.unit.Broken, p.span(n))
	return true
}

func (p *powershell) walk(n *syntax.Node, scope *Func, returns bool) {
	switch {
	case p.broken(n):
	case n.Kind == "comment":
		p.unit.Comments = append(p.unit.Comments, p.span(n))
	case n.Kind == "function_statement":
		p.function(n, scope)
	case n.Kind == "foreach_statement":
		p.foreach(n, scope)
	case n.Kind == "pipeline":
		p.statement(n, scope, returns)
	default:
		returns = n.Kind == "flow_control_statement" && strings.HasPrefix(p.lower(n), "return")
		for _, c := range n.Children {
			p.walk(c, scope, returns)
		}
	}
}

func (p *powershell) function(n *syntax.Node, scope *Func) {
	f := &Func{Span: p.span(n), Parent: scope}
	n.Walk(func(c *syntax.Node) bool {
		switch c.Kind {
		case "function_name":
			f.Name = p.lower(c)
		case "script_parameter":
			if v := kind(c, "variable"); v != nil {
				f.Params = append(f.Params, variableName(p.lower(v)))
			}
		}
		return c.Kind != "script_block_body"
	})
	p.unit.Funcs = append(p.unit.Funcs, f)
	for _, c := range n.Children {
		if c.Kind != "function_name" && c.Kind != "function_parameter_declaration" {
			p.walk(c, f, false)
		}
	}
}

func kind(n *syntax.Node, name string) *syntax.Node {
	for _, c := range n.Children {
		if c.Kind == name {
			return c
		}
	}
	return nil
}

func (p *powershell) foreach(n *syntax.Node, scope *Func) {
	item, list := kind(n, "variable"), kind(n, "pipeline")
	if item != nil && list != nil {
		st := p.add(Span{Start: p.span(item).Start, End: p.span(list).End}, scope)
		st.Binds = []string{variableName(p.lower(item))}
		p.gather(list, st, &st.Value)
	}
	if body := kind(n, "statement_block"); body != nil {
		p.walk(body, scope, false)
	}
}

func (p *powershell) add(s Span, scope *Func) *Stmt {
	st := &Stmt{Unit: p.unit, Span: s, Scope: scope, Value: Expr{Span: s}}
	p.unit.Stmts = append(p.unit.Stmts, st)
	return st
}

func (p *powershell) statement(n *syntax.Node, scope *Func, returns bool) {
	st := p.add(p.span(n), scope)
	st.Return = returns
	assign := kind(n, "assignment_expression")
	if assign == nil {
		p.gather(n, st, &st.Value)
		return
	}
	if left := kind(assign, "left_assignment_expression"); left != nil {
		left.Walk(func(c *syntax.Node) bool {
			if c.Kind == "variable" && len(st.Binds) == 0 {
				st.Binds = []string{variableName(p.lower(c))}
			}
			return true
		})
	}
	if value := assign.Child("value"); value != nil {
		p.gather(value, st, &st.Value)
	}
}

func variableName(text string) string {
	name := strings.Trim(strings.TrimLeft(text, "$@"), "{}")
	for _, scope := range powershellScopes {
		name = strings.TrimPrefix(name, scope)
	}
	if rest, ok := strings.CutPrefix(name, envDrive); ok {
		return envDrive + strings.ToUpper(rest)
	}
	return name
}

func (p *powershell) gather(n *syntax.Node, st *Stmt, e *Expr) {
	switch {
	case p.broken(n):
	case n.Kind == "comment":
		p.unit.Comments = append(p.unit.Comments, p.span(n))
	case n.Kind == "type_literal":
	case n.Kind == "variable":
		p.variable(n, e)
	case n.Kind == "command":
		p.command(n, st, e, nil)
	case n.Kind == "pipeline_chain":
		p.chain(n, st, e)
	case n.Kind == "invokation_expression" && kind(n, "argument_list") != nil:
		p.invoke(n, st, e)
	case n.Kind == "script_block_expression":
		p.walk(n, st.Scope, false)
	default:
		for _, c := range n.Children {
			p.gather(c, st, e)
		}
	}
}

func (p *powershell) variable(n *syntax.Node, e *Expr) {
	name := variableName(p.lower(n))
	if slices.Contains(powershellValues, name) {
		return
	}
	e.Refs = append(e.Refs, Ref{Name: name, Span: p.span(n)})
}

func (p *powershell) sub(n *syntax.Node, st *Stmt) Expr {
	e := Expr{Span: p.span(n)}
	p.gather(n, st, &e)
	return e
}

func (p *powershell) chain(n *syntax.Node, st *Stmt, e *Expr) {
	var piped *Expr
	for _, c := range n.Children {
		switch {
		case !c.Named:
		case c.Kind == "command":
			own := p.command(c, st, e, piped)
			piped = &own
		default:
			own := p.sub(c, st)
			e.merge(own)
			piped = &own
		}
	}
}

func (p *powershell) command(n *syntax.Node, st *Stmt, e *Expr, piped *Expr) Expr {
	own := Expr{Span: p.span(n)}
	c := &Call{Stmt: st, Span: own.Span}
	st.Calls = append(st.Calls, c)
	if name := n.Child("command_name"); name != nil {
		c.Callee = p.lower(name)
		own.merge(p.sub(name, st))
	}
	if elements := n.Child("command_elements"); elements != nil {
		p.elements(elements, st, c, &own)
	}
	if piped != nil {
		c.Args = append(c.Args, Arg{Name: "pipeline", Expr: *piped})
	}
	e.merge(own)
	return own
}

func (p *powershell) elements(n *syntax.Node, st *Stmt, c *Call, own *Expr) {
	flag := ""
	for _, el := range n.Children {
		switch {
		case !el.Named || el.Kind == "command_argument_sep":
		case el.Kind == "command_parameter":
			flag = strings.Trim(p.lower(el), "-:")
		case el.Kind == "redirection" || el.Kind == "stop_parsing":
			own.merge(p.sub(el, st))
		default:
			arg := Arg{Name: flag, Expr: p.sub(el, st)}
			flag = ""
			c.Args = append(c.Args, arg)
			own.merge(arg.Expr)
		}
	}
}

func (p *powershell) invoke(n *syntax.Node, st *Stmt, e *Expr) {
	list := kind(n, "argument_list")
	c := &Call{Stmt: st, Span: p.span(n)}
	c.Callee = strings.Join(strings.Fields(strings.ToLower(p.tree.Source[n.Start:list.Start])), "")
	st.Calls = append(st.Calls, c)
	for _, part := range n.Children {
		if part == list {
			break
		}
		if part.Kind == "variable" && c.Receiver == "" {
			c.Receiver = variableName(p.lower(part))
		}
		p.gather(part, st, e)
	}
	list.Walk(func(a *syntax.Node) bool {
		if a.Kind != "argument_expression" {
			return true
		}
		arg := Arg{Expr: p.sub(a, st)}
		c.Args = append(c.Args, arg)
		e.merge(arg.Expr)
		return false
	})
}
