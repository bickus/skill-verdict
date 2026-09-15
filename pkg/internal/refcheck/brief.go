// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const BriefPath = "references.brief.md"

//go:embed brief.md
var briefHeader string

var headings = map[refs.Kind]string{refs.Address: "Addresses", refs.Domain: "Domains", refs.GitHub: "GitHub", refs.Package: "Packages"}

func Brief(skill *inventory.Skill, reports []refs.Report) string {
	sorted := slices.Clone(reports)
	refs.Sort(sorted)
	var b strings.Builder
	b.WriteString(briefHeader)
	src := sources{skill: skill, lines: make(map[string][]string)}
	var kind refs.Kind
	for _, rep := range sorted {
		if rep.Ref.Kind != kind {
			kind = rep.Ref.Kind
			fmt.Fprintf(&b, "\n## %s\n", headings[kind])
		}
		describe(&b, rep, src)
	}
	return b.String()
}

func describe(b *strings.Builder, rep refs.Report, src sources) {
	fmt.Fprintf(b, "\n### %s\n", rep.Ref.Value)
	fmt.Fprintf(b, "- status: %s\n", rep.Status)
	if rep.Detail != "" {
		fmt.Fprintf(b, "- detail: %s\n", rep.Detail)
	}
	if rep.Ref.Derived {
		b.WriteString("- derived: true\n")
	}
	for _, loc := range rep.Ref.Locations {
		text := src.line(loc.File, loc.Line)
		if text == "" {
			fmt.Fprintf(b, "- used at: %s:%d\n", loc.File, loc.Line)
			continue
		}
		fmt.Fprintf(b, "- used at: %s:%d `%s`\n", loc.File, loc.Line, textutil.Clip(text))
		if registry := installRegistry(text); rep.Ref.Kind == refs.Package && registry != "" {
			def, _, _ := strings.Cut(rep.Ref.Value, ":")
			fmt.Fprintf(b, "  - note: the facts below are from %s, this command installs from %s\n", def, registry)
		}
	}
	for _, u := range rep.Ref.URLs {
		fmt.Fprintf(b, "- url: %s\n", u)
	}
	for _, f := range rep.Facts {
		fmt.Fprintf(b, "- %s: %s\n", f.Key, f.Value)
	}
}

type sources struct {
	skill *inventory.Skill
	lines map[string][]string
}

func (s sources) line(file string, n int) string {
	lines, seen := s.lines[file]
	if !seen {
		if a := s.skill.Artifact(file); a != nil {
			lines = textutil.Lines(a.Content)
		}
		s.lines[file] = lines
	}
	if n < 1 || n > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[n-1])
}
