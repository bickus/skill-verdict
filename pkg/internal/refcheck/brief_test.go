// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

var briefReports = []refs.Report{
	{ID: "R2", Ref: refs.Ref{Kind: refs.Package, Value: "npm:left-pad", Locations: []finding.Location{{File: "SKILL.md", Line: 6}}, Derived: true}, Status: refs.NotChecked, Detail: "budget",
		Findings: []string{"PACKAGE-MISSING"}},
	{ID: "R1", Ref: refs.Ref{Kind: refs.Domain, Value: "example.com", URLs: []string{"https://example.com/docs", "https://example.com/other"},
		Locations: []finding.Location{{File: "SKILL.md", Line: 6}, {File: "missing.sh", Line: 1}}}, Status: refs.Checked,
		Facts: []refs.Fact{{Key: "age_days", Value: "12"}}, Usages: []string{"agent-fetch"}, Findings: []string{"DOMAIN-NEW"}},
}

func TestBrief(t *testing.T) {
	got := refcheck.Brief(walked(t, "brief"), briefReports)
	body := got[strings.Index(got, "\n## "):]
	want := strings.Join([]string{
		"",
		"## Domains",
		"",
		"### R1 example.com",
		"- status: checked",
		"- R1.1 used at SKILL.md:6 `See https://example.com/docs now.`",
		"  - usage: agent-fetch",
		"- R1.2 used at missing.sh:1",
		"- url: https://example.com/docs",
		"- url: https://example.com/other",
		"- age_days: 12",
		"",
		"## Packages",
		"",
		"### R2 npm:left-pad",
		"- status: not-checked",
		"- detail: budget",
		"- derived: true",
		"- R2.1 used at SKILL.md:6 `See https://example.com/docs now.`",
		"",
	}, "\n")
	if body != want {
		t.Errorf("brief body:\n%s\nwant:\n%s", body, want)
	}
	if strings.Contains(got, "REF-") {
		t.Errorf("brief carries rule IDs:\n%s", got)
	}
}
