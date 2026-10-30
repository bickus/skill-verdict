// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package pipeline

import (
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
)

const hidden = "-HIDDEN"

type covered struct {
	rule string
	file string
}

type lines struct {
	first int
	last  int
}

func (s *State) Cover(rule, file string, first, last int) {
	if s.covers == nil {
		s.covers = make(map[covered][]lines)
	}
	k := covered{rule, file}
	s.covers[k] = append(s.covers[k], lines{first, last})
}

func (s *State) Covered(rule, file string, first, last int) bool {
	for _, l := range s.covers[covered{rule, file}] {
		if first <= l.last && last >= l.first {
			return true
		}
	}
	return false
}

func (s *State) Restated(rule, file string, line int) bool {
	text, ok := strings.CutSuffix(rule, hidden)
	if !ok {
		return false
	}
	twin, found := s.Rules.ByID[text]
	if !found || twin.Severity.Rank() < s.Rules.ByID[rule].Severity.Rank() {
		return false
	}
	return slices.ContainsFunc(s.Findings, func(f finding.Finding) bool {
		return f.Rule == text && f.Location.File == file && line >= f.Location.Line && line <= max(f.Location.Line, f.Location.EndLine)
	})
}
