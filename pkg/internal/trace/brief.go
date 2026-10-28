// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package trace

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

const BriefPath = "code.brief.md"

//go:embed brief.md
var briefHeader string

//go:embed note.md
var stateNote string

type writer struct {
	text strings.Builder
	seen map[string]bool
}

func (t *Trace) State(where string) (json.RawMessage, error) {
	if where != "" {
		where += "\n"
	}
	return json.Marshal(strings.TrimSpace(stateNote) + "\n" + where + t.Text())
}

func Brief(traces []*Trace, omitted int) string {
	var b strings.Builder
	b.WriteString(briefHeader)
	for _, t := range traces {
		b.WriteString("\n" + t.Text())
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "\n%d more traces are left out of this brief.\n", omitted)
	}
	return b.String()
}

func (t *Trace) Text() string {
	s := t.Subject
	w := &writer{seen: make(map[string]bool)}
	fmt.Fprintf(&w.text, "file: %s\nsubject: %s\n", s.Unit.Path, s.Match)
	w.line(0, "", s.Unit.Text(s.Stmt.Span))
	w.nodes(t.Feeds, 0)
	if t.Truncated {
		w.text.WriteString("trace truncated\n")
	}
	return w.text.String()
}

func (w *writer) line(depth int, label, code string) {
	pad := strings.Repeat(" ", depth)
	first, _, _ := strings.Cut(code, ":")
	switch {
	case code == "":
		w.text.WriteString(pad + label + "\n")
		return
	case w.seen[code]:
		w.text.WriteString(pad + label + " (shown above at " + first + ")\n")
		return
	case label != "":
		label += ": "
	}
	w.seen[code] = true
	w.text.WriteString(pad + label + strings.ReplaceAll(code, "\n", "\n"+pad+"  ") + "\n")
}

func (w *writer) nodes(nodes []*Node, depth int) {
	for _, n := range nodes {
		label := "feeds saved as " + n.Name
		if n.Call != "" {
			label = "feeds call " + n.Call
		}
		w.line(depth, label, n.Code)
		w.sources(n.FedBy, depth+1)
		w.nodes(n.Feeds, depth+1)
	}
}

func (w *writer) sources(sources []*Source, depth int) {
	for _, s := range sources {
		known := w.seen[s.Code]
		w.line(depth, strings.TrimSpace(fmt.Sprintf("fed by %s %s", s.Param, s.Name)), s.Code)
		if !known {
			w.sources(s.FedBy, depth+1)
		}
	}
}
