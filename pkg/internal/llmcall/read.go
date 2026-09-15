// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/llm"
)

const ReadTool = "read"

//go:embed read.md read.json origins.md
var files embed.FS

type Read struct {
	Skill    *inventory.Skill
	MaxChars int
}

func (Read) Spec() llm.Tool {
	return llm.Tool{Name: ReadTool, Description: Text(files, "read.md"), Parameters: Schema(files, "read.json")}
}

func (Read) Final() bool {
	return false
}

type readArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (r Read) Call(_ context.Context, raw json.RawMessage) (string, error) {
	var args readArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("bad arguments: %w", err)
	}
	a := r.Skill.Artifact(args.Path)
	if a == nil {
		return "", fmt.Errorf("no artifact at %q; use a path from the listing", args.Path)
	}
	if err := unreadable(a); err != nil {
		return "", err
	}
	if a.ContentType == inventory.Symlink {
		return fmt.Sprintf("## Symlink: %s -> %s\n", a.Path, a.Content), nil
	}
	lines := split(a.Content)
	if total, start := len(lines), max(args.Offset, 1); start > max(total, 1) {
		return "", fmt.Errorf("%s has %d lines, offset %d is past the end", a.Path, total, start)
	}
	return page(a, lines, args, r.MaxChars), nil
}

func Inline(a *inventory.Artifact, maxChars int) string {
	if err := unreadable(a); err != nil {
		return err.Error() + "\n"
	}
	return page(a, split(a.Content), readArgs{}, maxChars)
}

func unreadable(a *inventory.Artifact) error {
	switch a.ReadStatus {
	case inventory.ReadFailed:
		return fmt.Errorf("%s: the scanner did not read it (%s): %s", a.Path, a.ReadStatus, strings.Join(a.ScanNotes, "; "))
	case inventory.ReadOK, "":
	}
	switch a.ContentType {
	case inventory.Media, inventory.Unknown:
		return fmt.Errorf("%s: %s content of %d bytes, no text is available", a.Path, a.ContentType, a.Size)
	case inventory.Special, inventory.Directory:
		return fmt.Errorf("%s: a %s, it has no content", a.Path, a.ContentType)
	case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data, inventory.Symlink:
	}
	return nil
}

func page(a *inventory.Artifact, lines []string, args readArgs, maxChars int) string {
	total := len(lines)
	start := max(args.Offset, 1)
	if total == 0 {
		return fmt.Sprintf("## File: %s (empty)\n", a.Path)
	}
	end := total
	if args.Limit > 0 {
		end = min(start+args.Limit-1, total)
	}
	var body strings.Builder
	last := start - 1
	for n := start; n <= end; n++ {
		line := fmt.Sprintf("L%d: %s", n, lines[n-1])
		if body.Len()+len(line) > maxChars && n > start {
			break
		}
		body.WriteString(line)
		last = n
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## File: %s (lines %d-%d of %d)\n", a.Path, start, last, total)
	if last < total {
		fmt.Fprintf(&b, "Lines after %d were not returned; call %s again with offset %d.\n", last, ReadTool, last+1)
	}
	b.WriteString(body.String())
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func split(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.SplitAfter(text, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}
