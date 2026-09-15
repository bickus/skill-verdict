// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package discovery_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/discovery"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/rules"
)

type fake struct {
	replies []llm.Message
	err     error
	calls   int
}

func (f *fake) Complete(context.Context, llm.Request) (llm.Message, error) {
	i := f.calls
	f.calls++
	if f.err != nil {
		return llm.Message{}, f.err
	}
	if i < len(f.replies) {
		return f.replies[i], nil
	}
	return submit(), nil
}

func turn(name, args string) llm.Message {
	return llm.Message{Role: llm.Assistant, Calls: []llm.Call{{ID: "id-" + name, Name: name, Arguments: json.RawMessage(args)}}}
}

func submit(items ...string) llm.Message {
	return turn(discovery.SubmitTool, `{"findings":[`+strings.Join(items, ",")+`]}`)
}

func add(item string) llm.Message {
	return turn(discovery.FindingTool, `{"action":"add",`+item[1:])
}

func read(path string) llm.Message {
	return turn(llmcall.ReadTool, `{"path":"`+path+`"}`)
}

func item(file, rule string, line, confidence int) string {
	return `{"file":"` + file + `","rule_id":"` + rule + `","message":"leaks input","start_line":` + strconv.Itoa(line) +
		`,"end_line":` + strconv.Itoa(line+1) + `,"confidence":` + strconv.Itoa(confidence) + `}`
}

func newState(t *testing.T, settings map[string]config.RuleSetting) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", "skill"))
	if err != nil {
		t.Fatal(err)
	}
	layers := config.Default().Layers
	layers.Discovery.Rules = settings
	set, err := rules.Load("", layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	return s
}

func layer(p llm.Provider, calls int) discovery.Layer {
	return discovery.Layer{Runner: llmcall.Runner{Provider: p, LLM: config.Default().LLM, Calls: calls}}
}

func ruleIDs(findings []finding.Finding) []string {
	var ids []string
	for _, f := range findings {
		ids = append(ids, f.Rule)
	}
	return ids
}

func TestRunKeepsOnlyEnabledConfidentRules(t *testing.T) {
	on := true
	cases := []struct {
		name     string
		settings map[string]config.RuleSetting
		replies  []llm.Message
		want     []string
	}{
		{"enabled rule becomes a finding", nil, []llm.Message{add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 4))}, []string{"EXFILTRATION-IN-PROSE"}},
		{"disabled rule dropped", nil, []llm.Message{add(item("SKILL.md", "PROMPT-INJECTION-SEMANTIC", 4, 4))}, nil},
		{"rule kept once enabled", map[string]config.RuleSetting{"PROMPT-INJECTION-SEMANTIC": {Enabled: &on}}, []llm.Message{add(item("SKILL.md", "PROMPT-INJECTION-SEMANTIC", 4, 4))}, []string{"PROMPT-INJECTION-SEMANTIC"}},
		{"below threshold dropped", nil, []llm.Message{add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 2))}, nil},
		{"static rule id dropped", nil, []llm.Message{add(item("SKILL.md", "REMOTE-ENDPOINT-UPLOAD", 4, 4))}, nil},
		{"unknown file rejected", nil, []llm.Message{add(item("NOTES.md", "EXFILTRATION-IN-PROSE", 4, 4))}, nil},
		{"removed finding dropped", nil, []llm.Message{add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 4)), add(`{"action":"remove","file":"SKILL.md","rule_id":"EXFILTRATION-IN-PROSE","start_line":4}`)}, nil},
		{"submitted with the last findings", nil, []llm.Message{submit(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 4), item("README.md", "DECEPTION-GRADUAL", 1, 3))}, []string{"EXFILTRATION-IN-PROSE", "DECEPTION-GRADUAL"}},
		{"rejected submit keeps the loop going", nil, []llm.Message{submit(item("NOTES.md", "EXFILTRATION-IN-PROSE", 4, 4)), add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 4))}, []string{"EXFILTRATION-IN-PROSE"}},
		{"nothing recorded", nil, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, tc.settings)
			pipeline.Run(context.Background(), s, []pipeline.Layer{layer(&fake{replies: tc.replies}, 10)})
			if got := ruleIDs(s.Findings); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("findings %v, want %v", got, tc.want)
			}
			if s.Layers[1].Status != pipeline.Done {
				t.Errorf("layers %+v; want a done layer", s.Layers)
			}
		})
	}
}

func TestRunServesFilesToTheModel(t *testing.T) {
	s := newState(t, nil)
	pipeline.Run(context.Background(), s, []pipeline.Layer{layer(&fake{replies: []llm.Message{read("SKILL.md"), add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 3))}}, 10)})
	if len(s.Findings) != 1 {
		t.Fatalf("findings %+v, want one", s.Findings)
	}
	want := finding.Finding{Rule: "EXFILTRATION-IN-PROSE", Severity: finding.High,
		Location: finding.Location{File: "SKILL.md", Line: 4, EndLine: 5}, Evidence: "leaks input"}
	if !reflect.DeepEqual(s.Findings[0], want) {
		t.Errorf("finding %+v, want %+v", s.Findings[0], want)
	}
}

func TestRunFailsTheLayer(t *testing.T) {
	cases := []struct {
		name  string
		fake  *fake
		calls int
		err   string
		want  []string
	}{
		{"provider error", &fake{err: errors.New("boom")}, 10, "boom", nil},
		{"budget exhausted", &fake{replies: []llm.Message{read("SKILL.md"), read("README.md")}}, 1, "llm: request budget exhausted", nil},
		{"budget exhausted keeps what was recorded", &fake{replies: []llm.Message{add(item("SKILL.md", "EXFILTRATION-IN-PROSE", 4, 4)), read("README.md")}}, 2, "llm: request budget exhausted", []string{"EXFILTRATION-IN-PROSE"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, nil)
			pipeline.Run(context.Background(), s, []pipeline.Layer{layer(tc.fake, tc.calls)})
			if got := ruleIDs(s.Findings); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("findings %v, want %v", got, tc.want)
			}
			if len(s.Layers) != 2 || s.Layers[1].Status != pipeline.Failed || s.Layers[1].Error != tc.err {
				t.Errorf("layers %+v, want discovery failed with %q", s.Layers, tc.err)
			}
		})
	}
}
