// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package static

import (
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
)

const maxHops = 40

func scanLink(s *pipeline.State, a *inventory.Artifact) {
	target := resolve(s.Skill, a)
	for _, r := range s.Rules.Static {
		if applies(r, a) && matchesAny(r.Patterns, target) && !matchesAny(r.Exclude, target) {
			s.Record(r.Finding(finding.Location{File: a.Path}, textutil.Evidence(a.Content)))
		}
	}
}

type resolved struct {
	root  string
	up    int
	names []string
}

func resolve(skill *inventory.Skill, a *inventory.Artifact) string {
	r := &resolved{}
	if dir := path.Dir(a.Path); dir != "." {
		r.names = strings.Split(dir, "/")
	}
	parts := r.start(a.Content)
	hops := 0
	for len(parts) > 0 {
		part := parts[0]
		parts = parts[1:]
		switch part {
		case "", ".":
		case "..":
			r.parent()
		default:
			r.names = append(r.names, part)
			if target, ok := r.link(skill); ok && hops < maxHops {
				hops++
				r.names = r.names[:len(r.names)-1]
				parts = append(r.start(target), parts...)
			}
		}
	}
	return r.String()
}

func (r *resolved) start(target string) []string {
	target = strings.ReplaceAll(target, `\`, "/")
	switch {
	case strings.HasPrefix(target, "/"):
		*r = resolved{root: "/"}
	case drive(target):
		*r = resolved{root: target[:2] + "/"}
		target = target[2:]
	}
	return strings.Split(target, "/")
}

func drive(p string) bool {
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := unicode.ToLower(rune(p[0]))
	return 'a' <= c && c <= 'z'
}

func (r *resolved) parent() {
	switch {
	case len(r.names) > 0:
		r.names = r.names[:len(r.names)-1]
	case r.root == "":
		r.up++
	}
}

func (r *resolved) link(skill *inventory.Skill) (string, bool) {
	if r.root != "" || r.up > 0 {
		return "", false
	}
	a := skill.Artifact(strings.Join(r.names, "/"))
	if a == nil || a.ContentType != inventory.Symlink {
		return "", false
	}
	return a.Content, true
}

func (r *resolved) String() string {
	if r.root != "" {
		return r.root + strings.Join(r.names, "/")
	}
	if p := strings.Join(append(slices.Repeat([]string{".."}, r.up), r.names...), "/"); p != "" {
		return p
	}
	return "."
}
