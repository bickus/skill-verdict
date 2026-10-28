// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package pipeline

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
