// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"
)

//go:embed preamble.md
var preamble string

var (
	htmlComment     = regexp.MustCompile(`(?s)<!--.*?-->[ \t]*\r?\n?`)
	placeholder     = regexp.MustCompile(`\{\{([a-z]+)\}\}`)
	placeholderLine = regexp.MustCompile(`(?m)^\{\{([a-z]+)\}\}\n`)
)

func strip(text string) string {
	return strings.TrimSpace(htmlComment.ReplaceAllString(text, ""))
}

func Text(embedded fs.FS, name string) string {
	data, err := fs.ReadFile(embedded, name)
	if err != nil {
		panic(err)
	}
	return strip(string(data))
}

func System(embedded fs.FS, name string, values map[string]string) string {
	return strip(preamble) + "\n\n" + Render(embedded, name, values)
}

func Render(embedded fs.FS, name string, values map[string]string) string {
	text := Text(embedded, name)
	var wanted []string
	for _, m := range placeholder.FindAllStringSubmatch(text, -1) {
		wanted = append(wanted, m[1])
	}
	slices.Sort(wanted)
	if given := slices.Sorted(maps.Keys(values)); !slices.Equal(slices.Compact(wanted), given) {
		panic(fmt.Sprintf("llmcall: %s has placeholders %v, code supplies %v", name, wanted, given))
	}
	text = placeholderLine.ReplaceAllStringFunc(text, func(m string) string {
		if values[m[2:len(m)-3]] == "" {
			return ""
		}
		return m
	})
	return placeholder.ReplaceAllStringFunc(text, func(m string) string {
		return values[m[2:len(m)-2]]
	}) + "\n"
}

func Schema(embedded fs.FS, name string) json.RawMessage {
	data, err := fs.ReadFile(embedded, name)
	if err != nil {
		panic(err)
	}
	return data
}
