// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan

import (
	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type Verdict string

const (
	Clean      Verdict = "clean"
	Review     Verdict = "review"
	Incomplete Verdict = "incomplete"
	Block      Verdict = "block"
)

func (v Verdict) Rank() int {
	switch v {
	case Clean:
		return 0
	case Review:
		return 1
	case Incomplete:
		return 2
	case Block:
		return 3
	}
	return -1
}

func Compute(findings []finding.Finding, artifacts []Artifact, reports []refs.Report, layers []LayerStatus, gate config.GateConfig) Verdict {
	review := false
	for _, f := range findings {
		if !f.Live() {
			continue
		}
		if f.Severity.Rank() >= gate.Severity.Rank() {
			return Block
		}
		review = true
	}
	if !Complete(artifacts, reports, layers) {
		return Incomplete
	}
	if review {
		return Review
	}
	return Clean
}

func (a Artifact) Inspected() bool {
	return a.ReadStatus != "failed" && a.ContentType != "unknown"
}

func (l LayerStatus) Inspected() bool {
	return l.Status != "failed" && len(l.Errors) == 0
}

func Complete(artifacts []Artifact, reports []refs.Report, layers []LayerStatus) bool {
	for _, a := range artifacts {
		if !a.Inspected() {
			return false
		}
	}
	for _, r := range reports {
		if r.Status != refs.Checked {
			return false
		}
	}
	for _, l := range layers {
		if !l.Inspected() {
			return false
		}
	}
	return true
}
