// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llmcall

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
)

func Artifacts(s *inventory.Skill) string {
	sorted := make([]*inventory.Artifact, 0, len(s.Artifacts))
	for i := range s.Artifacts {
		if s.Artifacts[i].Kind == inventory.Brief {
			continue
		}
		sorted = append(sorted, &s.Artifacts[i])
	}
	slices.SortStableFunc(sorted, func(a, b *inventory.Artifact) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Path), strings.ToLower(b.Path)), cmp.Compare(a.Path, b.Path))
	})
	var b strings.Builder
	for _, a := range sorted {
		fmt.Fprintf(&b, "%s - %s, %d bytes", a.Path, a.ContentType, a.Size)
		if a.Origin != inventory.Original {
			fmt.Fprintf(&b, ", origin %s", a.Origin)
		}
		if a.ReadStatus == inventory.ReadFailed {
			fmt.Fprintf(&b, ", not read: %s", a.ReadStatus)
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func Origins(s *inventory.Skill) string {
	present := make(map[inventory.Origin]bool)
	for i := range s.Artifacts {
		a := &s.Artifacts[i]
		if o := a.Origin; o != inventory.Original && a.Kind != inventory.Brief {
			present[o] = true
		}
	}
	if len(present) == 0 {
		return ""
	}
	var b strings.Builder
	for _, o := range inventory.Origins() {
		if present[o.Name] {
			fmt.Fprintf(&b, "- %s: %s\n", o.Name, o.Description)
		}
	}
	return Render(files, "origins.md", map[string]string{"origins": strings.TrimSuffix(b.String(), "\n")})
}

const skillFile = "SKILL.md"

func Entry(s *inventory.Skill) *inventory.Artifact {
	if a := s.Artifact(inventory.RevealedPath(skillFile)); a != nil {
		return a
	}
	return s.Artifact(skillFile)
}
