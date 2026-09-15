// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package inventory

import (
	"slices"
	"strings"
)

func isFence(line string) bool {
	return strings.TrimRight(line, " \t\r") == "---"
}

func fenced(text string) ([]string, int) {
	lines := strings.Split(strings.TrimPrefix(text, "\uFEFF"), "\n")
	if !isFence(lines[0]) {
		return lines, -1
	}
	return lines, slices.IndexFunc(lines[1:], isFence)
}

func frontmatter(text string) (string, string) {
	lines, end := fenced(text)
	if end < 0 {
		return "", ""
	}
	body := lines[1 : 1+end]
	return field(body, "name"), field(body, "description")
}

func Body(text string) string {
	lines, end := fenced(text)
	if end < 0 {
		return text
	}
	return strings.TrimLeft(strings.Join(lines[2+end:], "\n"), "\n")
}

func field(lines []string, key string) string {
	for i, line := range lines {
		rest, ok := strings.CutPrefix(line, key+":")
		if !ok {
			continue
		}
		value := strings.TrimSpace(rest)
		if strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
			return block(lines[i+1:], value[0] == '>')
		}
		return unquote(value)
	}
	return ""
}

func block(lines []string, fold bool) string {
	var parts []string
	for _, line := range lines {
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		trimmed := strings.TrimSpace(line)
		if !indented && trimmed != "" {
			break
		}
		parts = append(parts, trimmed)
	}
	if fold {
		parts = slices.DeleteFunc(parts, func(p string) bool { return p == "" })
		return strings.Join(parts, " ")
	}
	return strings.TrimRight(strings.Join(parts, "\n"), "\n")
}

func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		return value[1 : len(value)-1]
	}
	return value
}
