// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package references

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	name             = "references"
	tooMany          = "REFERENCES-TOO-MANY"
	packageUnpopular = "PACKAGE-UNPINNED-UNPOPULAR"
	repoUnpopular    = "REPO-UNPINNED-UNPOPULAR"
)

var origin = inventory.RegisterOrigin(name, "Facts the scanner collected about the skill's external references from registries and services. Created by the scanner itself")

var popularity = map[string]refs.Kind{packageUnpopular: refs.Package, repoUnpopular: refs.GitHub}

type placing struct {
	rule   rules.Rule
	kind   refs.Kind
	prefix string
}

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
	refs.Number(reports)
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
	s.Record(l.unpopular(s, reports)...)
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

func (l Layer) unpopular(s *pipeline.State, reports []refs.Report) []finding.Finding {
	var out []finding.Finding
	for _, id := range slices.Sorted(maps.Keys(popularity)) {
		rule, on := enabled(s, id)
		if !on || !l.collects(popularity[id]) {
			continue
		}
		for _, at := range slices.Concat(s.Findings, s.Places) {
			if prefix, ok := rule.Places[at.Rule]; ok {
				out = append(out, placing{rule, popularity[id], prefix}.place(at, reports)...)
			}
		}
	}
	return finding.Dedup(out)
}

func (l Layer) collects(kind refs.Kind) bool {
	return slices.ContainsFunc(l.collectors, func(c refcheck.Collector) bool { return c.Kind() == kind })
}

func (p placing) place(at finding.Finding, reports []refs.Report) []finding.Finding {
	var out []finding.Finding
	named := false
	for _, rep := range reports {
		if !p.names(rep.Ref, at.Location) {
			continue
		}
		named = true
		if rep.Status == refs.Checked && rep.Unpopular {
			f := p.rule.Finding(at.Location, rep.Ref.Value)
			f.Reference = rep.Ref.Value
			out = append(out, f)
		}
	}
	if !named {
		out = append(out, p.rule.Finding(at.Location, at.Evidence))
	}
	return out
}

func (p placing) names(ref refs.Ref, span finding.Location) bool {
	if ref.Kind != p.kind || ref.Derived || !strings.HasPrefix(ref.Value, p.prefix) {
		return false
	}
	return slices.ContainsFunc(ref.Locations, func(loc finding.Location) bool {
		return loc.File == span.File && loc.Line >= span.Line && loc.Line <= max(span.EndLine, span.Line)
	})
}

func location(ref refs.Ref) finding.Location {
	if len(ref.Locations) == 0 {
		return finding.Location{}
	}
	return ref.Locations[0]
}
