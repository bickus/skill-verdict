// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package finding

import (
	"cmp"
	"slices"
)

type Kind string

const (
	KindHeuristic Kind = "heuristic"
	KindLLM       Kind = "llm"
	KindFact      Kind = "fact"
)

type Location struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine,omitempty"`
}

type State string

const (
	Confirmed  State = "confirmed"
	Upheld     State = "upheld"
	Downgraded State = "downgraded"
	Dismissed  State = "dismissed"
)

type Judgement struct {
	State      State    `json:"state"`
	From       Severity `json:"from"`
	Confidence int      `json:"confidence"`
	Reason     string   `json:"reason"`
}

type Finding struct {
	Rule      string     `json:"rule"`
	Severity  Severity   `json:"severity"`
	Location  Location   `json:"location"`
	Evidence  string     `json:"evidence"`
	Reference string     `json:"reference,omitempty"`
	Judgement *Judgement `json:"judgement,omitempty"`
}

func (f Finding) Live() bool {
	return f.Judgement == nil || f.Judgement.State != Dismissed
}

type dedupKey struct {
	rule string
	file string
	line int
}

func Dedup(fs []Finding) []Finding {
	seen := make(map[dedupKey]bool, len(fs))
	out := make([]Finding, 0, len(fs))
	for _, f := range fs {
		key := dedupKey{f.Rule, f.Location.File, f.Location.Line}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

func Sort(fs []Finding) {
	slices.SortStableFunc(fs, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(b.Severity.Rank(), a.Severity.Rank()),
			cmp.Compare(a.Location.File, b.Location.File),
			cmp.Compare(a.Location.Line, b.Location.Line),
			cmp.Compare(a.Rule, b.Rule),
		)
	})
}
