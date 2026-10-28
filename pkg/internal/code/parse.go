// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/syntax"
)

var extensions = map[string]Lang{
	".py": Python, ".sh": Shell, ".bash": Shell, ".zsh": Shell,
	".js": JavaScript, ".mjs": JavaScript, ".cjs": JavaScript, ".jsx": JavaScript,
	".ts": TypeScript, ".mts": TypeScript, ".cts": TypeScript, ".tsx": TSX,
	".ps1": PowerShell, ".psm1": PowerShell,
}

var interpreters = map[string]Lang{"python": Python, "python3": Python, "sh": Shell, "bash": Shell, "zsh": Shell, "node": JavaScript, "pwsh": PowerShell}

var fenceTags = map[string]Lang{
	"python": Python, "py": Python, "python3": Python,
	"bash": Shell, "sh": Shell, "shell": Shell, "zsh": Shell,
	"javascript": JavaScript, "js": JavaScript, "node": JavaScript, "mjs": JavaScript, "jsx": JavaScript,
	"typescript": TypeScript, "ts": TypeScript, "tsx": TSX,
	"powershell": PowerShell, "pwsh": PowerShell, "ps1": PowerShell, "ps": PowerShell,
}

func FileLang(name, content string) (Lang, bool) {
	if lang, ok := extensions[strings.ToLower(path.Ext(name))]; ok {
		return lang, true
	}
	first, _, _ := strings.Cut(content, "\n")
	if path.Ext(name) != "" || !strings.HasPrefix(first, "#!") {
		return "", false
	}
	fields := strings.Fields(strings.TrimPrefix(first, "#!"))
	for _, f := range fields {
		if lang, ok := interpreters[path.Base(f)]; ok {
			return lang, true
		}
	}
	return "", false
}

type Block struct {
	Lang Lang
	Span Span
}

func Blocks(markdown string) []Block {
	var out []Block
	var open *Block
	marker := ""
	offset := 0
	for _, line := range strings.SplitAfter(markdown, "\n") {
		run, info := Fence(line)
		switch {
		case open == nil && run != "":
			marker = run
			tag, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(info)), " ")
			open = &Block{Lang: fenceTags[strings.Trim(tag, "{}.")], Span: Span{Start: offset + len(line)}}
		case open != nil && strings.HasPrefix(run, marker) && strings.TrimSpace(info) == "":
			open.Span.End = offset
			if open.Lang != "" && open.Span.End > open.Span.Start {
				out = append(out, *open)
			}
			open = nil
		}
		offset += len(line)
	}
	if open != nil && open.Lang != "" && offset > open.Span.Start {
		open.Span.End = offset
		out = append(out, *open)
	}
	return out
}

func Fence(line string) (run, info string) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return "", ""
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == trimmed[0] {
		n++
	}
	if n < 3 {
		return "", ""
	}
	return trimmed[:n], trimmed[n:]
}

type Parser struct {
	trees *syntax.Parser
}

var ErrRuntime = errors.New("code: parser runtime failed to start")

func (p *Parser) Close(ctx context.Context) error {
	if p.trees == nil {
		return nil
	}
	return p.trees.Close(ctx)
}

func (p *Parser) Parse(ctx context.Context, name string, lang Lang, content string, part Span) (*Unit, error) {
	unit := newUnit(name, lang, content)
	source := content[part.Start:part.End]
	if lang == Shell {
		if len(source) > syntax.MaxBytes {
			return nil, syntax.ErrTooLarge
		}
		lowerShell(unit, source, part.Start)
		return unit, nil
	}
	if p.trees == nil {
		trees, err := syntax.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRuntime, err)
		}
		p.trees = trees
	}
	language := syntax.PowerShell
	if sp := specs[lang]; sp != nil {
		language = sp.language
	}
	tree, err := p.trees.Parse(ctx, language, source)
	if err != nil {
		return nil, err
	}
	switch {
	case tree.Root == nil:
		unit.Broken = append(unit.Broken, part)
	case lang == PowerShell:
		(&powershell{tree: tree, unit: unit, base: part.Start}).walk(tree.Root, nil, false)
	default:
		l := &lowerer{spec: specs[lang], tree: tree, unit: unit, base: part.Start}
		l.block(tree.Root, nil)
	}
	return unit, nil
}
