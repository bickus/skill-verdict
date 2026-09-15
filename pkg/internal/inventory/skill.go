// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package inventory

import "path/filepath"

type Skill struct {
	Root        string
	Name        string
	Description string
	Hash        string
	Artifacts   []Artifact
}

func (s *Skill) Artifact(path string) *Artifact {
	for i := range s.Artifacts {
		if s.Artifacts[i].Path == path {
			return &s.Artifacts[i]
		}
	}
	return nil
}

func (s *Skill) Identify() {
	if a := s.Artifact("SKILL.md"); a != nil {
		s.Name, s.Description = frontmatter(a.Content)
	}
	if s.Name == "" {
		s.Name = filepath.Base(s.Root)
	}
}
