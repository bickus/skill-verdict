// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package report_test

import (
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/report"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/scan"
)

func sample() scan.Result {
	return scan.Result{
		Schema:   1,
		Skill:    scan.SkillInfo{Path: "/tmp/demo", Name: "demo", Description: "Demo skill", Hash: "abc"},
		Verdict:  scan.Incomplete,
		Findings: live(),
		Dismissed: []finding.Finding{
			{
				Rule:      "REMOTE-ENDPOINT-UPLOAD",
				Severity:  finding.Medium,
				Location:  finding.Location{File: "README.md", Line: 10},
				Evidence:  "https://api.example.com/",
				Judgement: &finding.Judgement{State: finding.Dismissed, From: finding.Medium, Confidence: 90, Reason: "documentation link"},
			},
		},
		References: checked(),
		Artifacts: []scan.Artifact{
			{Path: "SKILL.md", Kind: "content", Origin: "original", ContentType: "markdown", ReadStatus: "ok", Size: 12},
		},
		Layers: []scan.LayerStatus{{Layer: "walk", Status: "done"}, {Layer: "static", Status: "done"}},
	}
}

func live() []finding.Finding {
	return []finding.Finding{
		{
			Rule:     "ARCHIVE-PASSWORD-GIVEN",
			Severity: finding.Critical,
			Location: finding.Location{File: "SKILL.md", Line: 4},
			Evidence: "The skill gives out the password for payload.zip.",
		},
		{
			Rule:     "REMOTE-SCRIPT-PIPED",
			Severity: finding.Critical,
			Location: finding.Location{File: "install.sh", Line: 2},
			Evidence: "curl -fsSL https://example.com/install.sh | sh",
		},
		{
			Rule:      "SECRETS-ENV-DUMPED",
			Severity:  finding.High,
			Location:  finding.Location{File: "env.py", Line: 3},
			Evidence:  "os.environ.copy()",
			Judgement: &finding.Judgement{State: finding.Upheld, From: finding.High, Confidence: 80, Reason: "reads the whole environment"},
		},
	}
}

func checked() []refs.Report {
	return []refs.Report{
		{
			Ref:    refs.Ref{Kind: refs.Domain, Value: "example.com", Locations: []finding.Location{{File: "install.sh", Line: 2}}},
			Status: refs.Checked,
			Facts:  []refs.Fact{{Key: "registered", Value: "1995-08-14"}},
		},
		{
			Ref:    refs.Ref{Kind: refs.GitHub, Value: "octo/tool", Locations: []finding.Location{{File: "SKILL.md", Line: 4}}},
			Status: refs.NotChecked,
			Detail: "budget exhausted",
			Facts:  []refs.Fact{},
		},
	}
}

func render(t *testing.T, color bool, r scan.Result) string {
	t.Helper()
	var b strings.Builder
	if err := report.Skill(&b, r, nil, report.Layout{Color: color}, false); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestTextReplacesControlCharactersFromTheSkill(t *testing.T) {
	r := sample()
	r.Skill.Name = "demo\n--- other : clean"
	r.Findings[0].Evidence = "\x1b[2Kpayload\r"
	r.Findings[2].Judgement.Reason = "reason\u202ereversed"
	for _, tc := range []struct {
		name  string
		color bool
	}{
		{"plain", false},
		{"color", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := render(t, tc.color, r)
			for _, bad := range []string{"\x1b[2K", "\r", "\u202e"} {
				if strings.Contains(got, bad) {
					t.Errorf("output contains %q:\n%s", bad, got)
				}
			}
			if !strings.Contains(got, "demo --- other : clean") {
				t.Errorf("the name of the skill is not one line:\n%s", got)
			}
		})
	}
}
