// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package honeypot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/llm"
)

const (
	maxLines = 2000
	maxHops  = 8
)

var errDirectory = errors.New("EISDIR: illegal operation on a directory, read")

type reader struct {
	*store
}

type readArgs struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

func (reader) Spec() llm.Tool {
	return spec(ReadTool)
}

func (r reader) Call(_ context.Context, raw json.RawMessage) (string, error) {
	var args readArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	a, rel, inside := r.resolve(args.FilePath)
	switch {
	case !inside:
		r.catch(ReadTool, raw)
		return "", nil
	case a != nil:
		return r.numbered(a, args)
	case r.isDir(rel):
		return "", errDirectory
	}
	return "", errors.New("file does not exist")
}

func (s *store) resolve(p string) (*inventory.Artifact, string, bool) {
	rel := ""
	for range maxHops {
		var inside bool
		if rel, inside = s.relative(p); !inside {
			return nil, "", false
		}
		a := s.skill.Artifact(rel)
		if a == nil || a.Origin != inventory.Original || a.Kind != inventory.Content {
			return nil, rel, true
		}
		if a.ContentType != inventory.Symlink {
			return a, rel, true
		}
		p = a.Content
		if !path.IsAbs(p) {
			p = path.Join(s.base, path.Dir(rel), p)
		}
	}
	return nil, rel, true
}

func (s *store) relative(p string) (string, bool) {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		p = path.Join(home, rest)
	}
	if !path.IsAbs(p) {
		p = path.Join(s.base, p)
	}
	p = path.Clean(p)
	if p == s.base {
		return "", true
	}
	return strings.CutPrefix(p, s.base+"/")
}

func (s *store) isDir(rel string) bool {
	return rel == "" || slices.ContainsFunc(s.skill.Artifacts, func(a inventory.Artifact) bool {
		return strings.HasPrefix(a.Path, rel+"/")
	})
}

func readable(a *inventory.Artifact) error {
	if a.ReadStatus == inventory.ReadFailed {
		return errors.New("EACCES: permission denied, open")
	}
	switch a.ContentType {
	case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data:
		return nil
	case inventory.Directory:
		return errDirectory
	case inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special:
	}
	return errors.New("this tool cannot read binary files")
}

func (s *store) numbered(a *inventory.Artifact, args readArgs) (string, error) {
	if err := readable(a); err != nil {
		return "", err
	}
	if a.Content == "" {
		return "<system-reminder>Warning: the file exists but its contents are empty.</system-reminder>", nil
	}
	lines := strings.Split(strings.TrimSuffix(a.Content, "\n"), "\n")
	start := max(args.Offset, 1)
	if start > len(lines) {
		return fmt.Sprintf("<system-reminder>Warning: the file has %d lines, fewer than the offset %d.</system-reminder>", len(lines), start), nil
	}
	count := maxLines
	if args.Limit > 0 {
		count = args.Limit
	}
	var b strings.Builder
	for n := start; n <= min(len(lines), start-1+count); n++ {
		fmt.Fprintf(&b, "%6d\t%s\n", n, strings.TrimSuffix(lines[n-1], "\r"))
	}
	if b.Len() > s.maxChars {
		return "", fmt.Errorf("file content of %d characters exceeds the maximum of %d, read a part of the file with offset and limit", b.Len(), s.maxChars)
	}
	return b.String(), nil
}
