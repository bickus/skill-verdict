// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	fileBytes   = "FILE-TOO-LARGE"
	bundleBytes = "BUNDLE-TOO-LARGE"
	textBytes   = "BUNDLE-TOO-MUCH-TEXT"
	fileCount   = "BUNDLE-TOO-MANY-FILES"
	depth       = "BUNDLE-TOO-DEEP"
	encrypted   = "ARCHIVE-ENCRYPTED"
	executable  = "FILE-BINARY"
	installer   = "FILE-INSTALLER"
)

type limits struct {
	cfg     config.WalkConfig
	allowed map[string]bool
	named   []rules.Rule
	done    map[string]bool
	files   int
	bundle  int64
	total   int64
}

func newLimits(cfg config.WalkConfig, set *rules.Set) limits {
	l := limits{cfg: cfg, allowed: set.ForLayer(name), done: make(map[string]bool)}
	for _, id := range slices.Sorted(maps.Keys(set.ByID)) {
		if r := set.ByID[id]; l.allowed[id] && len(r.Files) > 0 {
			r.Files = lower(r.Files)
			l.named = append(l.named, r)
		}
	}
	return l
}

func lower(globs []string) []string {
	out := make([]string, len(globs))
	for i, g := range globs {
		out[i] = strings.ToLower(g)
	}
	return out
}

func (l *limits) count(s *pipeline.State, rel string) {
	l.files++
	if l.files > l.cfg.MaxFiles {
		l.once(s, fileCount, rel, fmt.Sprintf("%d files, limit %d", l.files, l.cfg.MaxFiles))
	}
}

func (l *limits) measure(s *pipeline.State, rel string, size int64) {
	if d := strings.Count(rel, "/"); d > l.cfg.MaxDepth {
		l.once(s, depth, rel, fmt.Sprintf("depth %d, limit %d", d, l.cfg.MaxDepth))
	}
	if size > l.cfg.MaxFileBytes {
		l.report(s, fileBytes, rel, fmt.Sprintf("%d bytes, limit %d", size, l.cfg.MaxFileBytes))
	}
	l.bundle += size
	if l.bundle > l.cfg.MaxBundleBytes {
		l.once(s, bundleBytes, rel, fmt.Sprintf("%d bytes in the bundle, limit %d", l.bundle, l.cfg.MaxBundleBytes))
	}
	base := strings.ToLower(path.Base(rel))
	for _, r := range l.named {
		if config.MatchAny(r.Files, base) {
			l.report(s, r.ID, rel, "matches "+strings.Join(r.Files, ", "))
		}
	}
}

func (l *limits) text(s *pipeline.State, a inventory.Artifact) {
	switch a.ContentType {
	case inventory.Markdown, inventory.Code, inventory.Text, inventory.Data:
	case inventory.Media, inventory.Unknown, inventory.Symlink, inventory.Special, inventory.Directory:
		return
	}
	l.total += a.Size
	if l.total > l.cfg.MaxTextBytes {
		l.once(s, textBytes, a.Path, fmt.Sprintf("%d bytes of text, limit %d", l.total, l.cfg.MaxTextBytes))
	}
}

func (l *limits) archive(s *pipeline.State, rel string, raw []byte) {
	if format := encryptedArchive(raw); format != "" {
		l.report(s, encrypted, rel, format+" archive, contents encrypted")
	}
}

func (l *limits) format(s *pipeline.State, a inventory.Artifact, raw []byte) {
	if a.ContentType != inventory.Unknown {
		return
	}
	if sig, ok := detect(raw); ok {
		l.report(s, sig.rule, a.Path, sig.format)
	}
}

func (l *limits) once(s *pipeline.State, id, rel, evidence string) {
	if l.done[id] {
		return
	}
	l.done[id] = true
	l.report(s, id, rel, evidence)
}

func (l *limits) report(s *pipeline.State, id, rel, evidence string) {
	if !l.allowed[id] {
		return
	}
	s.Record(s.Rules.ByID[id].Finding(finding.Location{File: rel}, evidence))
}
