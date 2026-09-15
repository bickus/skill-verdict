// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan_test

import (
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/scan"
)

var (
	complete   = []scan.Artifact{{Path: "SKILL.md", Kind: "content", Origin: "original", ContentType: "markdown", ReadStatus: "ok"}}
	incomplete = []scan.Artifact{{Path: "blob.bin", Kind: "content", Origin: "original", ContentType: "unknown", ReadStatus: "ok"}}
	atGate     = finding.Finding{Rule: "COMMAND-ON-LOAD", Severity: finding.High}
	critical   = finding.Finding{Rule: "REMOTE-SCRIPT-PIPED", Severity: finding.Critical}
	belowGate  = finding.Finding{Rule: "REMOTE-ENDPOINT-UPLOAD", Severity: finding.Medium}
)

func judged(f finding.Finding, state finding.State, severity finding.Severity) finding.Finding {
	f.Judgement = &finding.Judgement{State: state, From: f.Severity, Confidence: 80, Reason: "reason"}
	f.Severity = severity
	return f
}

func TestComputeVerdict(t *testing.T) {
	cases := []struct {
		name     string
		findings []finding.Finding
		entries  []scan.Artifact
		want     scan.Verdict
	}{
		{"nothing found", nil, complete, scan.Clean},
		{"finding at the gate blocks", []finding.Finding{atGate}, complete, scan.Block},
		{"confirmed blocks", []finding.Finding{judged(critical, finding.Confirmed, finding.Critical)}, complete, scan.Block},
		{"upheld at the gate blocks", []finding.Finding{judged(atGate, finding.Upheld, finding.High)}, complete, scan.Block},
		{"downgraded below the gate reviews", []finding.Finding{judged(critical, finding.Downgraded, finding.Low)}, complete, scan.Review},
		{"dismissed is ignored", []finding.Finding{judged(critical, finding.Dismissed, finding.Critical)}, complete, scan.Clean},
		{"below the gate reviews", []finding.Finding{belowGate}, complete, scan.Review},
		{"incomplete beats review", []finding.Finding{belowGate}, incomplete, scan.Incomplete},
		{"block beats incomplete", []finding.Finding{atGate}, incomplete, scan.Block},
	}
	gate := config.Default().Gate
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scan.Compute(tc.findings, tc.entries, nil, nil, gate); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	cases := []struct {
		name      string
		artifacts []scan.Artifact
		reports   []refs.Report
		layers    []scan.LayerStatus
		want      bool
	}{
		{"text, media and symlink read", []scan.Artifact{{ReadStatus: "ok", ContentType: "markdown"}, {ReadStatus: "ok", ContentType: "media"}, {ReadStatus: "ok", ContentType: "symlink"}}, nil, []scan.LayerStatus{{Layer: "static", Status: "done"}}, true},
		{"derived copy", []scan.Artifact{{Origin: "reveal", ContentType: "markdown"}}, nil, nil, true},
		{"unknown content", []scan.Artifact{{ReadStatus: "ok", ContentType: "unknown"}}, nil, nil, false},
		{"file not read", []scan.Artifact{{ReadStatus: "failed"}}, nil, nil, false},
		{"reference checked", nil, []refs.Report{{Status: refs.Checked}}, nil, true},
		{"reference not checked", nil, []refs.Report{{Status: refs.NotChecked}}, nil, false},
		{"reference failed", nil, []refs.Report{{Status: refs.Failed}}, nil, false},
		{"layer failed", nil, nil, []scan.LayerStatus{{Layer: "judge", Status: "failed"}}, false},
		{"layer interrupted", nil, nil, []scan.LayerStatus{{Layer: "walk", Status: "interrupted", Interrupt: "FILE-TOO-LARGE"}}, true},
		{"layer skipped after an interrupt", nil, nil, []scan.LayerStatus{{Layer: "judge", Status: "skipped"}}, true},
		{"layer with an artifact error", nil, nil, []scan.LayerStatus{{Layer: "judge", Status: "done", Errors: []scan.ArtifactError{{Artifact: "SKILL.md", Error: "boom"}}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scan.Complete(tc.artifacts, tc.reports, tc.layers); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
