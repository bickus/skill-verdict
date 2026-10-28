// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package ast

import (
	"context"
	"errors"

	"github.com/bickus/skill-verdict/pkg/internal/code"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/trace"
)

const name = "ast"

type Layer struct {
	Parser *code.Parser
}

func (Layer) Name() string {
	return name
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	var units []*code.Unit
	for i := range s.Skill.Artifacts {
		a := &s.Skill.Artifacts[i]
		if a.Kind != inventory.Content {
			continue
		}
		for _, p := range parts(a) {
			unit, err := l.Parser.Parse(ctx, a.Path, p.Lang, a.Content, p.Span)
			if ctx.Err() != nil {
				return pipeline.Report{}, ctx.Err()
			}
			if errors.Is(err, code.ErrRuntime) {
				return pipeline.Report{}, err
			}
			if err != nil {
				a.Note(name + ": " + err.Error())
				continue
			}
			units = append(units, unit)
		}
	}
	s.Code = trace.NewProgram(units)
	return pipeline.Report{}, nil
}

func parts(a *inventory.Artifact) []code.Block {
	switch a.ContentType {
	case inventory.Markdown:
		return code.Blocks(a.Content)
	case inventory.Code, inventory.Text:
		if lang, ok := code.FileLang(a.Path, a.Content); ok {
			return []code.Block{{Lang: lang, Span: code.Span{End: len(a.Content)}}}
		}
	case inventory.Data, inventory.Symlink, inventory.Media, inventory.Unknown, inventory.Special, inventory.Directory:
	}
	return nil
}
