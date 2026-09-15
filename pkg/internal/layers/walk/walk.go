// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package walk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
)

const name = "walk"

type Layer struct {
	Config config.WalkConfig
}

func (Layer) Name() string {
	return name
}

func Resolve(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", root)
	}
	info, err = os.Lstat(filepath.Join(resolved, "SKILL.md"))
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s: SKILL.md is not a regular file", root)
	}
	return resolved, nil
}

type entry struct {
	rel string
	abs string
	d   fs.DirEntry
}

type walker struct {
	state   *pipeline.State
	hash    hash.Hash
	pending []entry
	limits
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	defer s.Skill.Identify()
	w := &walker{state: s, hash: sha256.New(), limits: newLimits(l.Config, s.Rules)}
	if err := filepath.WalkDir(s.Skill.Root, w.visit); err != nil {
		return pipeline.Report{}, err
	}
	slices.SortFunc(w.pending, func(a, b entry) int { return strings.Compare(a.rel, b.rel) })
	for _, e := range w.pending {
		if err := ctx.Err(); err != nil {
			return pipeline.Report{}, err
		}
		w.process(ctx, e)
	}
	if err := ctx.Err(); err != nil {
		return pipeline.Report{}, err
	}
	s.Skill.Hash = hex.EncodeToString(w.hash.Sum(nil))
	return pipeline.Report{}, nil
}

func (w *walker) visit(p string, d fs.DirEntry, err error) error {
	root := w.state.Skill.Root
	if p == root {
		return err
	}
	rel := filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(p, root), string(filepath.Separator)))
	if err != nil {
		w.failed(rel+"/", inventory.Directory, err)
		return nil
	}
	if d.IsDir() {
		return nil
	}
	w.pending = append(w.pending, entry{rel: rel, abs: p, d: d})
	return nil
}

func (w *walker) failed(rel string, ctype inventory.ContentType, err error) {
	w.state.Skill.Artifacts = append(w.state.Skill.Artifacts, inventory.Artifact{
		Path: rel, Kind: inventory.Content, Origin: inventory.Original, ContentType: ctype, ReadStatus: inventory.ReadFailed,
		ScanNotes: []string{"read: " + err.Error()},
	})
}

func (w *walker) process(ctx context.Context, e entry) {
	w.count(w.state, e.rel)
	info, err := e.d.Info()
	if err != nil {
		w.failed(e.rel, "", err)
		return
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		w.state.Skill.Artifacts = append(w.state.Skill.Artifacts, inventory.Artifact{
			Path: e.rel, Kind: inventory.Content, Origin: inventory.Original, ContentType: inventory.Symlink, ReadStatus: inventory.ReadOK, Content: linkTarget(e.abs),
		})
		return
	}
	if !info.Mode().IsRegular() {
		w.state.Skill.Artifacts = append(w.state.Skill.Artifacts, inventory.Artifact{
			Path: e.rel, Kind: inventory.Content, Origin: inventory.Original, ContentType: inventory.Special, ReadStatus: inventory.ReadOK,
		})
		return
	}
	w.measure(w.state, e.rel, info.Size())
	if info.Size() > w.cfg.MaxFileBytes {
		w.failed(e.rel, "", fmt.Errorf("skipped, %d bytes over limit %d", info.Size(), w.cfg.MaxFileBytes))
		return
	}
	if ctx.Err() != nil {
		return
	}
	raw, err := os.ReadFile(e.abs)
	if err != nil {
		w.failed(e.rel, "", err)
		return
	}
	w.digest(e.rel, raw)
	w.archive(w.state, e.rel, raw)
	a := inventory.Regular(e.rel, raw)
	w.text(w.state, a)
	w.format(w.state, a, raw)
	w.state.Skill.Artifacts = append(w.state.Skill.Artifacts, a)
}

func (w *walker) digest(rel string, raw []byte) {
	w.hash.Write([]byte(rel))
	w.hash.Write([]byte{0})
	w.hash.Write(raw)
	w.hash.Write([]byte{0})
}

func linkTarget(abs string) string {
	target, err := os.Readlink(abs)
	if err != nil {
		return ""
	}
	return target
}
