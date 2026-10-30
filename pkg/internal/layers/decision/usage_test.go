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

func TestToolServerAnswerRaisesOneServerFinding(t *testing.T) {
	host := refs.Ref{Kind: refs.Domain, Value: "tools.example.com", URLs: []string{"https://tools.example.com/mcp"}, Locations: []finding.Location{{File: "SKILL.md", Line: 3}}}
	twice := refs.Ref{Kind: refs.Domain, Value: "tools.example.com", URLs: []string{"https://tools.example.com/mcp"}, Locations: []finding.Location{{File: "SKILL.md", Line: 6}, {File: "SKILL.md", Line: 3}}}
	repo := refs.Ref{Kind: refs.GitHub, Value: "acme/tools", URLs: []string{"https://github.com/acme/tools"}, Locations: []finding.Location{{File: "SKILL.md", Line: 4}}}
	cases := []struct {
		name string
		ref  refs.Ref
		rule string
		want []string
	}{
		{"a low fact gives the remote server finding", host, "DOMAIN-HTTP-ERROR", []string{"DOMAIN-HTTP-ERROR", "TOOL-SERVER-REMOTE-HIDDEN"}},
		{"a fact above low gives only the untrusted server finding", repo, "GITHUB-OWNER-MISSING", []string{"GITHUB-OWNER-MISSING", "TOOL-SERVER-UNTRUSTED"}},
		{"the text rule at the place gives no second finding", host, "TOOL-SERVER-REMOTE", []string{"TOOL-SERVER-REMOTE"}},
		{"the text rule at another place keeps the finding at this place", twice, "TOOL-SERVER-REMOTE", []string{"TOOL-SERVER-REMOTE", "TOOL-SERVER-REMOTE-HIDDEN"}},
		{"a repository that runs as a tool server is not remote", repo, "GITHUB-REPO-MOVED", []string{"GITHUB-REPO-MOVED"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := referenceState(t, tc.ref, tc.rule)
			client := model(t, `{"R1.1.usage":{"choice":"tool-server","probabilities":{"tool-server":0.97}},"R1.2.usage":{"choice":"tool-server","probabilities":{"tool-server":0.97}}}`)
			pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: client, Config: config.Default().Layers.Decision}})
			if s.Layers[1].Status != pipeline.Done || s.Refs[0].Usages[0] != "tool-server" {
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

func TestUsageRaisesARuleOnlyAtTheDismissProbability(t *testing.T) {
	host := refs.Ref{Kind: refs.Domain, Value: "tools.example.com", URLs: []string{"https://tools.example.com/mcp"}, Locations: []finding.Location{{File: "SKILL.md", Line: 3}}}
	repo := refs.Ref{Kind: refs.GitHub, Value: "acme/tools", URLs: []string{"https://github.com/acme/tools"}, Locations: []finding.Location{{File: "SKILL.md", Line: 4}}}
	cases := []struct {
		name    string
		ref     refs.Ref
		rule    string
		answers string
		want    []string
	}{
		{"an install answer at 0.8 raises the rule", host, "DOMAIN-HTTP-ERROR", `{"R1.1.usage":{"choice":"agent-install","probabilities":{"agent-install":0.8,"unknown":0.2}}}`, []string{"DOMAIN-HTTP-ERROR", "REMOTE-CODE-AT-INSTALL"}},
		{"an install answer below 0.8 raises nothing", host, "DOMAIN-HTTP-ERROR", `{"R1.1.usage":{"choice":"agent-install","probabilities":{"agent-install":0.79,"unknown":0.21}}}`, []string{"DOMAIN-HTTP-ERROR"}},
		{"a tool server answer at 0.8 raises the rule", repo, "GITHUB-OWNER-MISSING", `{"R1.1.usage":{"choice":"tool-server","probabilities":{"tool-server":0.8,"unknown":0.2}}}`, []string{"GITHUB-OWNER-MISSING", "TOOL-SERVER-UNTRUSTED"}},
		{"a tool server answer below 0.8 raises nothing", repo, "GITHUB-OWNER-MISSING", `{"R1.1.usage":{"choice":"tool-server","probabilities":{"tool-server":0.79,"unknown":0.21}}}`, []string{"GITHUB-OWNER-MISSING"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := referenceState(t, tc.ref, tc.rule)
			pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: model(t, tc.answers), Config: config.Default().Layers.Decision}})
			var got []string
			for _, f := range s.Findings {
				got = append(got, f.Rule)
			}
			if s.Layers[1].Status != pipeline.Done || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("layer %+v, findings %v, want %v", s.Layers[1], got, tc.want)
			}
		})
	}
}

func TestUsageFindingCoversOnlyTheBlockOfItsPlace(t *testing.T) {
	host := refs.Ref{Kind: refs.Domain, Value: "tools.example.com", URLs: []string{"https://tools.example.com/mcp"}, Locations: []finding.Location{{File: "SKILL.md", Line: 6}}}
	s := referenceState(t, host, "DOMAIN-HTTP-ERROR")
	client := model(t, `{"R1.1.usage":{"choice":"tool-server","probabilities":{"tool-server":0.97}}}`)
	pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: client, Config: config.Default().Layers.Decision}})
	cases := []struct {
		name string
		line int
		want bool
	}{
		{"the list item of the place", 6, true},
		{"the next list item", 7, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.Covered("TOOL-SERVER-REMOTE-HIDDEN", "SKILL.md", tc.line, tc.line); got != tc.want {
				t.Errorf("line %d covered = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}
