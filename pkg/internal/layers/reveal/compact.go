// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package reveal

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
)

//go:embed data/spaced-words.json
var spacedWordsJSON []byte

var letterSpacing = regexp.MustCompile(`\b\pL(?:[^\pL\r\n]+\pL){5,}\b`)

var spacedWord = sync.OnceValue(func() *regexp.Regexp {
	var file struct {
		Words []string `json:"words"`
	}
	if err := json.Unmarshal(spacedWordsJSON, &file); err != nil {
		panic(err)
	}
	alternatives := make([]string, len(file.Words))
	for i, word := range file.Words {
		letters := make([]string, 0, len(word))
		for _, r := range word {
			letters = append(letters, regexp.QuoteMeta(string(r)))
		}
		alternatives[i] = strings.Join(letters, `(?:[^\w\r\n]|_){0,8}`)
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(alternatives, "|") + `)\b`)
})

type span struct {
	start int
	end   int
}

func spans(text string) []span {
	var found []span
	for _, re := range []*regexp.Regexp{letterSpacing, spacedWord()} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			if !alphabetic(text[m[0]:m[1]]) {
				found = append(found, span{m[0], m[1]})
			}
		}
	}
	slices.SortFunc(found, func(a, b span) int { return cmp.Compare(a.start, b.start) })
	var merged []span
	for _, s := range found {
		if n := len(merged); n > 0 && s.start <= merged[n-1].end {
			merged[n-1].end = max(merged[n-1].end, s.end)
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

func alphabetic(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func compact(text string) (string, []span) {
	matched := spans(text)
	if len(matched) == 0 {
		return text, nil
	}
	var b strings.Builder
	b.Grow(len(text))
	cursor := 0
	for _, s := range matched {
		b.WriteString(text[cursor:s.start])
		for _, r := range text[s.start:s.end] {
			if unicode.IsLetter(r) {
				b.WriteRune(r)
			}
		}
		cursor = s.end
	}
	b.WriteString(text[cursor:])
	return b.String(), matched
}
