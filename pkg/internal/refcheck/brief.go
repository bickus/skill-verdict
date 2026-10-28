// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
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
	var kind refs.Kind
	for _, rep := range sorted {
		if rep.Ref.Kind != kind {
			kind = rep.Ref.Kind
			fmt.Fprintf(&b, "\n## %s\n", headings[kind])
		}
		describe(&b, skill, rep)
	}
	return b.String()
}

func Categorized(rep refs.Report) bool {
	return !rep.Ref.Derived && rep.Ref.Kind != refs.Package
}

func Section(skill *inventory.Skill, rep refs.Report) string {
	var b strings.Builder
	describe(&b, skill, rep)
	return strings.TrimSpace(b.String())
}

func describe(b *strings.Builder, skill *inventory.Skill, rep refs.Report) {
	fmt.Fprintf(b, "\n### %s %s\n", rep.ID, rep.Ref.Value)
	fmt.Fprintf(b, "- status: %s\n", rep.Status)
	if rep.Detail != "" {
		fmt.Fprintf(b, "- detail: %s\n", rep.Detail)
	}
	if rep.Ref.Derived {
		b.WriteString("- derived: true\n")
	}
	for i := range rep.Ref.Locations {
		place(b, skill, rep, i)
	}
	for _, u := range rep.Ref.URLs {
		fmt.Fprintf(b, "- url: %s\n", u)
	}
	for _, f := range rep.Facts {
		fmt.Fprintf(b, "- %s: %s\n", f.Key, f.Value)
	}
}

func place(b *strings.Builder, skill *inventory.Skill, rep refs.Report, i int) {
	loc := rep.Ref.Locations[i]
	text := Line(skill, loc)
	if text == "" {
		fmt.Fprintf(b, "- %s used at %s:%d\n", rep.LocationID(i), loc.File, loc.Line)
	} else {
		fmt.Fprintf(b, "- %s used at %s:%d `%s`\n", rep.LocationID(i), loc.File, loc.Line, textutil.Clip(text))
	}
	if registry := installRegistry(text); rep.Ref.Kind == refs.Package && registry != "" {
		def, _, _ := strings.Cut(rep.Ref.Value, ":")
		fmt.Fprintf(b, "  - note: the facts below are from %s, this command installs from %s\n", def, registry)
	}
	if usage := rep.Usage(i); usage != "" {
		fmt.Fprintf(b, "  - usage: %s\n", usage)
	}
}

func Line(skill *inventory.Skill, loc finding.Location) string {
	a := skill.Artifact(loc.File)
	if a == nil {
		return ""
	}
	lines := textutil.Lines(a.Content)
	if loc.Line < 1 || loc.Line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[loc.Line-1])
}
