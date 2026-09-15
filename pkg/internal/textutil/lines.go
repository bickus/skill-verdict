// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package textutil

import (
	"sort"
	"strings"
	"unicode"
)

func Lines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

func LineStarts(text string) []int {
	starts := []int{0}
	for i := range len(text) {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func LineAt(starts []int, offset int) int {
	return sort.Search(len(starts), func(i int) bool { return starts[i] > offset })
}

func Evidence(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, Clip(s))
}
