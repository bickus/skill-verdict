// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package scan

import (
	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const schemaVersion = 1

type SkillInfo struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Hash        string `json:"hash,omitempty"`
}

type Artifact struct {
	Path        string
	Kind        string
	Origin      string
	ContentType string
	ReadStatus  string
	Size        int64
	Notes       []string
}

type ArtifactError struct {
	Artifact string `json:"artifact"`
	Error    string `json:"error"`
}

type LayerStatus struct {
	Layer     string          `json:"layer"`
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	Interrupt string          `json:"interrupt,omitempty"`
	Errors    []ArtifactError `json:"errors,omitempty"`
	Tokens    *Tokens         `json:"tokens,omitempty"`
}

type Model struct {
	Name   string `json:"name"`
	Effort string `json:"effort,omitempty"`
}

type Tokens struct {
	Requests  int     `json:"requests"`
	Input     int     `json:"input"`
	Cached    int     `json:"cached"`
	Output    int     `json:"output"`
	Reasoning int     `json:"reasoning"`
	Cost      float64 `json:"cost,omitempty"`
}

type Result struct {
	Schema     int               `json:"schema"`
	Skill      SkillInfo         `json:"skill"`
	Verdict    Verdict           `json:"verdict"`
	Findings   []finding.Finding `json:"findings"`
	Dismissed  []finding.Finding `json:"dismissed"`
	References []refs.Report     `json:"references"`
	Artifacts  []Artifact        `json:"-"`
	Layers     []LayerStatus     `json:"layers"`
	Model      *Model            `json:"model,omitempty"`
	Cost       float64           `json:"cost,omitempty"`
}

func result(s *pipeline.State, cfg config.Config) Result {
	live, dismissed := []finding.Finding{}, []finding.Finding{}
	for _, f := range s.Findings {
		if f.Live() {
			live = append(live, f)
		} else {
			dismissed = append(dismissed, f)
		}
	}
	finding.Sort(live)
	finding.Sort(dismissed)
	reports := s.Refs
	if reports == nil {
		reports = []refs.Report{}
	}
	refs.Sort(reports)
	r := Result{
		Schema: schemaVersion,
		Skill: SkillInfo{
			Path:        s.Skill.Root,
			Name:        s.Skill.Name,
			Description: s.Skill.Description,
			Hash:        s.Skill.Hash,
		},
		Findings:   live,
		Dismissed:  dismissed,
		References: reports,
		Artifacts:  artifacts(s),
		Layers:     layers(s, cfg.LLM.Price),
	}
	if used, sum := usage(r.Layers); used {
		r.Model = &Model{Name: cfg.LLM.Model, Effort: cfg.LLM.ReasoningEffort}
		r.Cost = sum
	}
	r.Verdict = Compute(s.Findings, r.Artifacts, r.References, r.Layers, cfg.Gate)
	return r
}

func artifacts(s *pipeline.State) []Artifact {
	out := make([]Artifact, 0, len(s.Skill.Artifacts))
	for _, a := range s.Skill.Artifacts {
		out = append(out, Artifact{
			Path: a.Path, Kind: string(a.Kind), Origin: string(a.Origin), ContentType: string(a.ContentType),
			ReadStatus: string(a.ReadStatus), Size: a.Size, Notes: a.ScanNotes,
		})
	}
	return out
}

func layers(s *pipeline.State, price config.PriceConfig) []LayerStatus {
	out := make([]LayerStatus, 0, len(s.Layers))
	for _, l := range s.Layers {
		st := LayerStatus{Layer: l.Layer, Status: string(l.Status), Error: l.Error, Interrupt: l.Interrupt}
		for _, e := range l.Errors {
			st.Errors = append(st.Errors, ArtifactError{Artifact: e.Artifact, Error: e.Error})
		}
		if l.Requests > 0 {
			u := l.Usage
			st.Tokens = &Tokens{Requests: l.Requests, Input: u.Input, Cached: u.Cached, Output: u.Output, Reasoning: u.Reasoning,
				Cost: price.Cost(u.Input, u.Cached, u.Output)}
		}
		out = append(out, st)
	}
	return out
}

func usage(layers []LayerStatus) (bool, float64) {
	used, sum := false, 0.0
	for _, l := range layers {
		if l.Tokens != nil {
			used = true
			sum += l.Tokens.Cost
		}
	}
	return used, sum
}
