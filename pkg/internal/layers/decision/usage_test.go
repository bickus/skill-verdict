// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/decision"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func model(t *testing.T, answers string) *decisions.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, `{"answers":`+answers+`}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := config.Default().Layers.Decision
	cfg.BaseURL = srv.URL
	return decisions.New(cfg, "", srv.Client())
}

func referenceState(t *testing.T, ref refs.Ref, rule string) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", "tool-server"))
	if err != nil {
		t.Fatal(err)
	}
	layers := config.Default().Layers
	set, err := rules.Load("", layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	s.Refs = []refs.Report{{ID: "R1", Ref: ref, Status: refs.Checked}}
	s.Findings = []finding.Finding{{Rule: rule, Severity: set.ByID[rule].Severity, Location: ref.Locations[0], Evidence: ref.Value, Reference: ref.Value}}
	return s
}

func TestToolServerUntrustedCountsFactsAboveLow(t *testing.T) {
	host := refs.Ref{Kind: refs.Domain, Value: "tools.example.com", URLs: []string{"https://tools.example.com/mcp"}, Locations: []finding.Location{{File: "SKILL.md", Line: 3}}}
	repo := refs.Ref{Kind: refs.GitHub, Value: "acme/tools", URLs: []string{"https://github.com/acme/tools"}, Locations: []finding.Location{{File: "SKILL.md", Line: 4}}}
	cases := []struct {
		name string
		ref  refs.Ref
		rule string
		want []string
	}{
		{"error at the root alone raises nothing", host, "DOMAIN-HTTP-ERROR", []string{"DOMAIN-HTTP-ERROR"}},
		{"missing owner raises the finding", repo, "GITHUB-OWNER-MISSING", []string{"GITHUB-OWNER-MISSING", "TOOL-SERVER-UNTRUSTED"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := referenceState(t, tc.ref, tc.rule)
			client := model(t, `{"R1.1.usage":{"choice":"tool-server","probabilities":{"tool-server":0.97}}}`)
			pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: client, Config: config.Default().Layers.Decision}})
			if s.Layers[1].Status != pipeline.Done || !reflect.DeepEqual(s.Refs[0].Usages, []string{"tool-server"}) {
				t.Fatalf("decision layer %+v, usages %v", s.Layers[1], s.Refs[0].Usages)
			}
			var got []string
			for _, f := range s.Findings {
				got = append(got, f.Rule)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("findings %v, want %v", got, tc.want)
			}
		})
	}
}
