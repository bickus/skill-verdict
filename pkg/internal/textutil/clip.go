// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package textutil

import "unicode/utf8"

const (
	Limit    = 300
	ellipsis = "…"
)

func Clip(s string) string {
	if utf8.RuneCountInString(s) <= Limit {
		return s
	}
	end := 0
	for range Limit - utf8.RuneCountInString(ellipsis) {
		_, size := utf8.DecodeRuneInString(s[end:])
		end += size
	}
	return s[:end] + ellipsis
}

func Cut(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
