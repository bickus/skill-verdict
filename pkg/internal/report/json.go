// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package report

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
	"github.com/bickus/skill-verdict/pkg/scan"
)

type document struct {
	scan.Result
	Rules map[string]firedRule `json:"rules"`
}

type compact struct {
	Schema     int                  `json:"schema"`
	Skill      compactSkill         `json:"skill"`
	Verdict    scan.Verdict         `json:"verdict"`
	Findings   []compactFinding     `json:"findings"`
	References []compactReport      `json:"references"`
	Layers     []scan.LayerStatus   `json:"layers"`
	Cost       float64              `json:"cost,omitempty"`
	Rules      map[string]firedRule `json:"rules"`
}

type compactSkill struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Hash string `json:"hash,omitempty"`
}

type compactFinding struct {
	Rule      string            `json:"rule"`
	Severity  finding.Severity  `json:"severity"`
	Location  string            `json:"location"`
	Evidence  string            `json:"evidence"`
	Judgement *compactJudgement `json:"judgement,omitempty"`
}

type compactJudgement struct {
	State  finding.State `json:"state"`
	Reason string        `json:"reason"`
}

type compactReport struct {
	Ref    compactRef  `json:"reference"`
	Status refs.Status `json:"status"`
	Detail string      `json:"detail,omitempty"`
	Facts  []refs.Fact `json:"facts"`
}

type compactRef struct {
	Kind    refs.Kind `json:"kind"`
	Value   string    `json:"value"`
	URLs    []string  `json:"urls,omitempty"`
	Derived bool      `json:"derived,omitempty"`
}

type firedRule struct {
	Title       string       `json:"title"`
	Category    string       `json:"category"`
	Kind        finding.Kind `json:"kind"`
	Description string       `json:"description"`
}

func JSON(w io.Writer, results []scan.Result, byID map[string]rules.Rule) error {
	docs := make([]compact, 0, len(results))
	for _, r := range results {
		docs = append(docs, compact{
			Schema:     r.Schema,
			Skill:      compactSkill{Path: r.Skill.Path, Name: r.Skill.Name, Hash: r.Skill.Hash},
			Verdict:    r.Verdict,
			Findings:   compactFindings(r.Findings),
			References: compactReports(r.References),
			Layers:     r.Layers,
			Cost:       r.Cost,
			Rules:      fired(r.Findings, byID),
		})
	}
	return encode(w, docs)
}

func VerboseJSON(w io.Writer, results []scan.Result, byID map[string]rules.Rule) error {
	docs := make([]document, 0, len(results))
	for _, r := range results {
		docs = append(docs, document{Result: r, Rules: fired(slices.Concat(r.Findings, r.Dismissed), byID)})
	}
	return encode(w, docs)
}

func encode[T any](w io.Writer, docs []T) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if len(docs) == 1 {
		return enc.Encode(docs[0])
	}
	return enc.Encode(docs)
}

func compactFindings(fs []finding.Finding) []compactFinding {
	out := make([]compactFinding, 0, len(fs))
	for _, f := range fs {
		c := compactFinding{Rule: f.Rule, Severity: f.Severity, Location: span(f.Location), Evidence: f.Evidence}
		if f.Judgement != nil {
			c.Judgement = &compactJudgement{State: f.Judgement.State, Reason: f.Judgement.Reason}
		}
		out = append(out, c)
	}
	return out
}

func span(loc finding.Location) string {
	if loc.Line > 0 && loc.EndLine > loc.Line {
		return fmt.Sprintf("%s-%d", location(loc), loc.EndLine)
	}
	return location(loc)
}

func compactReports(reports []refs.Report) []compactReport {
	out := make([]compactReport, 0, len(reports))
	for _, r := range reports {
		out = append(out, compactReport{
			Ref:    compactRef{Kind: r.Ref.Kind, Value: r.Ref.Value, URLs: r.Ref.URLs, Derived: r.Ref.Derived},
			Status: r.Status,
			Detail: r.Detail,
			Facts:  r.Facts,
		})
	}
	return out
}

func fired(fs []finding.Finding, byID map[string]rules.Rule) map[string]firedRule {
	out := make(map[string]firedRule)
	for _, f := range fs {
		rule := byID[f.Rule]
		out[f.Rule] = firedRule{Title: rule.Title, Category: rule.Category, Kind: rule.Kind, Description: rule.Description}
	}
	return out
}
