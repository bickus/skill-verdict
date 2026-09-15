// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package finding_test

import (
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/finding"
)

func TestDedupKeepsFirstPerRuleFileAndLine(t *testing.T) {
	in := []finding.Finding{
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "first"},
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "second"},
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "SKILL.md", Line: 4}, Evidence: "other line"},
		{Rule: "SECRETS-ENV-DUMPED", Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "other rule"},
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "README.md", Line: 3}, Evidence: "other file"},
	}
	want := []finding.Finding{
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "first"},
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "SKILL.md", Line: 4}, Evidence: "other line"},
		{Rule: "SECRETS-ENV-DUMPED", Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "other rule"},
		{Rule: "REMOTE-ENDPOINT-UPLOAD", Location: finding.Location{File: "README.md", Line: 3}, Evidence: "other file"},
	}
	if got := finding.Dedup(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSortOrdersBySeverityFileLineRule(t *testing.T) {
	got := []finding.Finding{
		{Rule: "B", Severity: finding.Low, Location: finding.Location{File: "b.md", Line: 1}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "b.md", Line: 9}},
		{Rule: "Z", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 2}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 2}},
		{Rule: "C", Severity: finding.Critical, Location: finding.Location{File: "z.md", Line: 100}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 1}},
	}
	want := []finding.Finding{
		{Rule: "C", Severity: finding.Critical, Location: finding.Location{File: "z.md", Line: 100}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 1}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 2}},
		{Rule: "Z", Severity: finding.High, Location: finding.Location{File: "a.md", Line: 2}},
		{Rule: "A", Severity: finding.High, Location: finding.Location{File: "b.md", Line: 9}},
		{Rule: "B", Severity: finding.Low, Location: finding.Location{File: "b.md", Line: 1}},
	}
	finding.Sort(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
