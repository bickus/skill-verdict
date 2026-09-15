// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static

import (
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func fences(lines []string) []bool {
	fenced := make([]bool, len(lines))
	var open byte
	width := 0
	for i := range fenced {
		c, n, rest := fenceRun(lines[i])
		switch {
		case open == 0 && n >= 3:
			open, width = c, n
			fenced[i] = true
		case open != 0:
			fenced[i] = true
			if c == open && n >= width && strings.TrimSpace(rest) == "" {
				open = 0
			}
		}
	}
	return fenced
}

func fenceRun(line string) (byte, int, string) {
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(line) {
		return 0, 0, ""
	}
	c := line[indent]
	if c != '`' && c != '~' {
		return 0, 0, ""
	}
	n := 0
	for indent+n < len(line) && line[indent+n] == c {
		n++
	}
	return c, n, line[indent+n:]
}

func allowed(kind inventory.ContentType, scope rules.Scope, inFence bool) bool {
	switch scope {
	case rules.Any:
		return true
	case rules.Code:
		return codeAllowed(kind, inFence)
	case rules.Prose:
		return proseAllowed(kind, inFence)
	case rules.Link:
	}
	return false
}

func codeAllowed(kind inventory.ContentType, inFence bool) bool {
	switch kind {
	case inventory.Markdown:
		return inFence
	case inventory.Code, inventory.Data:
		return true
	case inventory.Text, inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special, inventory.Directory:
		return false
	}
	return false
}

func proseAllowed(kind inventory.ContentType, inFence bool) bool {
	switch kind {
	case inventory.Markdown:
		return !inFence
	case inventory.Text:
		return true
	case inventory.Code, inventory.Data, inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special, inventory.Directory:
		return false
	}
	return false
}
