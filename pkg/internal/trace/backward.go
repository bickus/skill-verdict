// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package trace

import (
	"fmt"
	"slices"

	"github.com/bickus/skill-verdict/pkg/internal/code"
)

type Source struct {
	Param string
	Name  string
	Code  string
	FedBy []*Source
}

func (t *tracer) fedBy(c *code.Call, st *code.Stmt) []*Source {
	var out []*Source
	if c.Receiver != "" {
		if s := t.origin("receiver", c.Receiver, st, 0); s != nil {
			out = append(out, s)
		}
	}
	for i, a := range c.Args {
		param := a.Name
		if param == "" {
			param = fmt.Sprintf("arg%d", i+1)
		}
		out = append(out, t.sources(param, a.Expr, st, 0)...)
	}
	return out
}

func (t *tracer) sources(param string, e code.Expr, at *code.Stmt, depth int) []*Source {
	var out []*Source
	var names []string
	for _, r := range e.Refs {
		if slices.Contains(names, r.Name) {
			continue
		}
		names = append(names, r.Name)
		if s := t.origin(param, r.Name, at, depth); s != nil {
			out = append(out, s)
		}
	}
	return out
}

func (t *tracer) origin(param, name string, at *code.Stmt, depth int) *Source {
	if at.Unit.Imports[name] || len(t.prog.funcs[name]) > 0 || depth > maxDepth || !t.spend() {
		return nil
	}
	s := &Source{Param: param, Name: name}
	if def := t.prog.definition(name, at); def != nil {
		s.Code = t.text(def.Unit, def.Span)
		s.FedBy = t.sources("", def.Value, def, depth+1)
		return s
	}
	f, index := t.prog.param(name, at)
	if f == nil {
		return s
	}
	for _, site := range t.prog.callSites(f) {
		s.FedBy = append(s.FedBy, t.passed(site, f, index, depth+1))
	}
	return s
}

func (t *tracer) passed(site *code.Call, f *code.Func, index, depth int) *Source {
	out := &Source{Code: t.text(site.Stmt.Unit, site.Stmt.Span)}
	arg := argument(site, f, index)
	if arg == nil {
		arg = spread(site)
	}
	if arg != nil {
		out.FedBy = t.sources("", arg.Expr, site.Stmt, depth)
	}
	return out
}

func spread(site *code.Call) *code.Arg {
	for i := range site.Args {
		if e := site.Args[i].Expr; refers(e, "@") || refers(e, "*") {
			return &site.Args[i]
		}
	}
	return nil
}
