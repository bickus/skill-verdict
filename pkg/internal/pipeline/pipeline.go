// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package pipeline

import (
	"context"
	"slices"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/trace"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

type Status string

const (
	Done        Status = "done"
	Failed      Status = "failed"
	Interrupted Status = "interrupted"
	Skipped     Status = "skipped"
)

type ArtifactError struct {
	Artifact string
	Error    string
}

type Report struct {
	Errors   []ArtifactError
	Usage    llm.Usage
	Requests int
}

type LayerStatus struct {
	Layer     string
	Status    Status
	Error     string
	Interrupt string
	Report
}

type State struct {
	Skill     *inventory.Skill
	Rules     *rules.Set
	Findings  []finding.Finding
	Refs      []refs.Report
	Layers    []LayerStatus
	Interrupt string
	Code      *trace.Program
	links     map[link][]*trace.Subject
	unlinked  map[link]bool
	covers    map[covered][]lines
	stop      context.CancelFunc
}

type link struct {
	rule string
	file string
	line int
}

func (s *State) Link(rule, file string, line int, subjects []*trace.Subject) {
	if s.links == nil {
		s.links = make(map[link][]*trace.Subject)
		s.unlinked = make(map[link]bool)
	}
	k := link{rule, file, line}
	if len(subjects) == 0 {
		s.unlinked[k] = true
	}
	for _, sub := range subjects {
		if !slices.Contains(s.links[k], sub) {
			s.links[k] = append(s.links[k], sub)
		}
	}
}

func (s *State) Linked(f finding.Finding) []*trace.Subject {
	k := link{f.Rule, f.Location.File, f.Location.Line}
	if s.unlinked[k] {
		return nil
	}
	return s.links[k]
}

func (s *State) Record(fs ...finding.Finding) {
	s.Findings = append(s.Findings, fs...)
	if s.Interrupt != "" {
		return
	}
	for _, f := range fs {
		if !s.Rules.ByID[f.Rule].Interrupt {
			continue
		}
		s.Interrupt = f.Rule
		if s.stop != nil {
			s.stop()
		}
		return
	}
}

type Layer interface {
	Name() string
	Run(ctx context.Context, s *State) (Report, error)
}

func Run(ctx context.Context, s *State, layers []Layer) {
	for _, l := range layers {
		if s.Interrupt != "" || ctx.Err() != nil {
			s.Layers = append(s.Layers, LayerStatus{Layer: l.Name(), Status: Skipped})
			continue
		}
		s.Layers = append(s.Layers, run(ctx, s, l))
	}
}

func run(ctx context.Context, s *State, l Layer) LayerStatus {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.stop = cancel
	rep, err := l.Run(ctx, s)
	st := LayerStatus{Layer: l.Name(), Status: Done, Report: rep}
	switch {
	case s.Interrupt != "":
		st.Status, st.Interrupt = Interrupted, s.Interrupt
	case err != nil:
		st.Status, st.Error = Failed, err.Error()
	}
	return st
}
