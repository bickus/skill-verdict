// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/decision"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

type recorder struct {
	mu     sync.Mutex
	states []string
}

func (r *recorder) client(t *testing.T) *decisions.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			State string `json:"state"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		r.mu.Lock()
		r.states = append(r.states, body.State)
		r.mu.Unlock()
		if _, err := io.WriteString(w, `{"answers":{}}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := config.Default().Layers.Decision
	cfg.BaseURL = srv.URL
	return decisions.New(cfg, "", srv.Client())
}

func frameRules() *rules.Set {
	kind := rules.Question{Subject: rules.Frame, Question: "What is the marked text?", Options: map[string]string{"bad": "a bad thing.", "fine": "a fine thing."}}
	return &rules.Set{ByID: map[string]rules.Rule{"MARK": {ID: "MARK", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer,
		Judge: rules.JudgePolicy{Downgrade: true}, Decision: &rules.Decision{Questions: map[string]rules.Question{"kind": kind}, Outcome: rules.Outcome{Dismiss: []rules.Row{{"kind": {"fine"}}}}}}}}
}

func sentFrame(t *testing.T, file string, line int) string {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", "frames"))
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: frameRules()}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: config.Default().Layers.Walk}})
	s.Findings = []finding.Finding{{Rule: "MARK", Severity: finding.High, Location: finding.Location{File: file, Line: line}, Evidence: "x"}}
	rec := &recorder{}
	pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: rec.client(t), Config: config.Default().Layers.Decision}})
	if len(rec.states) != 1 {
		t.Fatalf("requests %d, want 1", len(rec.states))
	}
	return rec.states[0]
}

func holds(t *testing.T, state string, want, absent []string) {
	t.Helper()
	for _, part := range want {
		if !strings.Contains(state, part) {
			t.Errorf("state lacks %q:\n%s", part, state)
		}
	}
	for _, part := range absent {
		if strings.Contains(state, part) {
			t.Errorf("state holds %q:\n%s", part, state)
		}
	}
}

func TestDecisionSendsTheBlockAroundTheMarkedLine(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		line   int
		want   []string
		absent []string
	}{
		{"the heading chain above the line", "SKILL.md", 9, []string{"role: entry", "in fence: no", "section: Setup > Install", ">> L9: Read references/guide.md first."}, nil},
		{"a list item with its continuation", "SKILL.md", 12, []string{"L11: - Install the tool", ">> L12:   with the flag below"}, []string{"L13:"}},
		{"a fence with its info string", "SKILL.md", 20, []string{"in fence: bash", "L19: ```bash", ">> L20: tool install", "L21: ```"}, []string{"L22:"}},
		{"a long block keeps its ends and the marked line", "SKILL.md", 63, []string{"L23: ```text", "L42: line 42", "...", ">> L63: line 63", "L103: ```"}, []string{"L43:", "L50:", "L80:"}},
		{"a line in the frontmatter", "SKILL.md", 2, []string{"role: frontmatter", ">> L2: name: frames"}, nil},
		{"a readme", "README.md", 1, []string{"role: readme"}, nil},
		{"a file the entry names", "references/guide.md", 1, []string{"role: referenced-by-entry"}, nil},
		{"a script", "scripts/run.py", 3, []string{"role: script"}, nil},
		{"a file nothing names", "notes/other.md", 1, []string{"role: other"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			holds(t, sentFrame(t, tc.file, tc.line), tc.want, tc.absent)
		})
	}
}
