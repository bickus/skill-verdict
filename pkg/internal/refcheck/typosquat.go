// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
	"unicode/utf8"
)

//go:embed data/popular-packages.json
var popularJSON []byte

var popular = sync.OnceValue(func() map[string][]string {
	var file struct {
		PyPI []string `json:"pypi"`
		NPM  []string `json:"npm"`
	}
	if err := json.Unmarshal(popularJSON, &file); err != nil {
		panic(err)
	}
	return map[string][]string{registryPyPI: normalized(registryPyPI, file.PyPI), registryNPM: normalized(registryNPM, file.NPM)}
})

func normalized(registry string, names []string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = normalize(registry, name)
	}
	return out
}

func normalize(registry, name string) string {
	name = strings.ToLower(name)
	if registry == registryPyPI {
		name = strings.ReplaceAll(name, "_", "-")
	}
	return name
}

func typosquat(registry, name string) (string, bool) {
	name = normalize(registry, name)
	if len(name) < 3 {
		return "", false
	}
	runes := utf8.RuneCountInString(name)
	for _, other := range popular()[registry] {
		if other == name {
			return "", false
		}
		if diff := runes - utf8.RuneCountInString(other); diff > 2 || diff < -2 {
			continue
		}
		d := levenshtein(name, other)
		if d <= 2 && d*3 <= min(len(name), len(other)) {
			return other, true
		}
	}
	return "", false
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
