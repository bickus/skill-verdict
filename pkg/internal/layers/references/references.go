// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package references

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name    = "references"
	tooMany = "REFERENCES-TOO-MANY"
)

var origin = inventory.RegisterOrigin(name, "Facts the scanner collected about the skill's external references from registries and services. Created by the scanner itself")

type Layer struct {
	collectors []refcheck.Collector
	cfg        config.ReferencesConfig
}

func New(client *http.Client, cfg config.ReferencesConfig, token string) Layer {
	client = refcheck.Paced(client, cfg)
	var collectors []refcheck.Collector
	if cfg.Collectors[string(refs.Domain)] {
		collectors = append(collectors, refcheck.NewDomain(client, cfg))
	}
	if cfg.Collectors[string(refs.GitHub)] {
		collectors = append(collectors, refcheck.NewGitHub(client, cfg, token))
	}
	if cfg.Collectors[string(refs.Package)] {
		collectors = append(collectors, refcheck.NewPackage(client, cfg))
	}
	if cfg.Collectors[string(refs.Address)] {
		collectors = append(collectors, refcheck.NewAddress())
	}
	return Layer{collectors: collectors, cfg: cfg}
}

func (Layer) Name() string {
	return name
}

func (l Layer) Run(ctx context.Context, s *pipeline.State) (pipeline.Report, error) {
	r := &refcheck.Runner{Collectors: l.collectors, Config: l.cfg, Capped: func(first refs.Ref, total int) {
		if rule, ok := enabled(s, tooMany); ok {
			s.Record(rule.Finding(location(first), fmt.Sprintf("%d references, limit %d", total, l.cfg.MaxReferences)))
		}
	}}
	reports := r.Run(ctx, s.Skill)
	if s.Interrupt != "" {
		reports = slices.DeleteFunc(reports, func(rep refs.Report) bool { return rep.Status != refs.Checked })
	}
	s.Refs = append(s.Refs, reports...)
	for _, rep := range reports {
		for _, id := range rep.Findings {
			if rule, ok := enabled(s, id); ok {
				f := rule.Finding(location(rep.Ref), rep.Ref.Value)
				f.Reference = rep.Ref.Value
				s.Record(f)
			}
		}
	}
	if len(reports) > 0 {
		text := refcheck.Brief(s.Skill, reports)
		s.Skill.Artifacts = append(s.Skill.Artifacts, inventory.Artifact{
			Path: refcheck.BriefPath, Kind: inventory.Brief, Origin: origin, ContentType: inventory.Markdown, Size: int64(len(text)), Content: text,
		})
	}
	return pipeline.Report{}, ctx.Err()
}

func enabled(s *pipeline.State, id string) (rules.Rule, bool) {
	rule, ok := s.Rules.ByID[id]
	return rule, ok && rule.Enabled
}

func location(ref refs.Ref) finding.Location {
	if len(ref.Locations) == 0 {
		return finding.Location{}
	}
	return ref.Locations[0]
}
