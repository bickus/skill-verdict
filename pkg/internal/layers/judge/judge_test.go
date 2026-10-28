// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package judge_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/judge"
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

func list(items ...string) string {
	return `{"verdicts":[` + strings.Join(items, ",") + `]}`
}

func record(items ...string) llm.Message {
	return turn(judge.VerdictTool, list(items...))
}

func submit(items ...string) llm.Message {
	return turn(judge.SubmitTool, list(items...))
}

func read(path string) llm.Message {
	return turn(llmcall.ReadTool, `{"path":"`+path+`"}`)
}

func verdict(id, value string, confidence int) string {
	return fmt.Sprintf(`{"finding":%q,"verdict":%q,"confidence":%d,"explanation":"because"}`, id, value, confidence)
}

func testRules() *rules.Set {
	return &rules.Set{ByID: map[string]rules.Rule{
		"KEEP":  {ID: "KEEP", Category: "exfiltration", Severity: finding.High, Kind: finding.KindLLM, Layer: config.DiscoveryLayer},
		"DROP":  {ID: "DROP", Category: "exfiltration", Severity: finding.High, Kind: finding.KindLLM, Layer: config.DiscoveryLayer, Judge: rules.JudgePolicy{Downgrade: true}},
		"FLOOR": {ID: "FLOOR", Category: "exfiltration", Severity: finding.High, Kind: finding.KindLLM, Layer: config.DiscoveryLayer, Judge: rules.JudgePolicy{Downgrade: true, DowngradeFloor: finding.Medium}},
		"TEXT":  {ID: "TEXT", Category: "exfiltration", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer, Judge: rules.JudgePolicy{Downgrade: true}},
		"FACT":  {ID: "FACT", Category: "reference", Severity: finding.High, Kind: finding.KindFact, Layer: config.ReferencesLayer, Judge: rules.JudgePolicy{Downgrade: true}},
	}}
}

func found(rule, file string, line int, severity finding.Severity) finding.Finding {
	return finding.Finding{Rule: rule, Severity: severity,
		Location: finding.Location{File: file, Line: line}, Evidence: "evidence"}
}

func newState(t *testing.T, findings ...finding.Finding) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", "skill"))
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: testRules()}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: config.Default().Layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	s.Findings = slices.Clone(findings)
	return s
}

func run(s *pipeline.State, p llm.Provider, calls int) {
	l := judge.Layer{Runner: llmcall.Runner{Provider: p, LLM: config.Default().LLM, Calls: calls}}
	pipeline.Run(context.Background(), s, []pipeline.Layer{l})
}

func TestRunAppliesTheVerdict(t *testing.T) {
	cases := []struct {
		name     string
		finding  finding.Finding
		replies  []llm.Message
		state    finding.State
		severity finding.Severity
	}{
		{"the rule severity confirms", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "high", 4))}, finding.Confirmed, finding.High},
		{"dismiss without a floor drops the finding", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "dismiss", 4))}, finding.Dismissed, finding.High},
		{"dismiss with a floor lands on the floor", found("FLOOR", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "dismiss", 4))}, finding.Downgraded, finding.Medium},
		{"dismiss of a rule that forbids lowering keeps the severity", found("KEEP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("ND1", "dismiss", 4))}, finding.Upheld, finding.High},
		{"below the floor lands on the floor", found("FLOOR", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "low", 4))}, finding.Downgraded, finding.Medium},
		{"inside the range lowers", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "low", 4))}, finding.Downgraded, finding.Low},
		{"above the rule severity confirms", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "critical", 4))}, finding.Confirmed, finding.High},
		{"lowering a rule that forbids it keeps the severity", found("KEEP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("ND1", "low", 4))}, finding.Upheld, finding.High},
		{"a later verdict replaces the earlier one", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{record(verdict("D1", "dismiss", 4)), record(verdict("D1", "high", 4))}, finding.Confirmed, finding.High},
		{"submit carries the last verdicts", found("DROP", "SKILL.md", 3, finding.High), []llm.Message{submit(verdict("D1", "low", 4))}, finding.Downgraded, finding.Low},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, tc.finding)
			run(s, &fake{replies: tc.replies}, 10)
			got := s.Findings[0]
			want := &finding.Judgement{State: tc.state, From: tc.finding.Severity, Confidence: 4, Reason: "because"}
			if got.Severity != tc.severity || !reflect.DeepEqual(got.Judgement, want) {
				t.Errorf("got %s %+v, want %s %+v", got.Severity, got.Judgement, tc.severity, want)
			}
		})
	}
}

func TestRunLeavesFindingsUnjudged(t *testing.T) {
	cases := []struct {
		name    string
		replies []llm.Message
	}{
		{"a guess below the threshold", []llm.Message{record(verdict("D1", "dismiss", 2))}},
		{"an unknown finding id", []llm.Message{record(verdict("D9", "dismiss", 4)), record(verdict("D1", "high", 1))}},
		{"a verdict that is neither a severity nor dismiss", []llm.Message{record(verdict("D1", "maybe", 4)), record(verdict("D1", "high", 1))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := found("DROP", "SKILL.md", 3, finding.High)
			s := newState(t, original)
			run(s, &fake{replies: tc.replies}, 10)
			if !reflect.DeepEqual(s.Findings, []finding.Finding{original}) {
				t.Errorf("findings %+v, want %+v untouched", s.Findings, original)
			}
			if len(s.Layers) != 2 || s.Layers[1].Status != pipeline.Done {
				t.Errorf("layers %+v, want a done judge layer", s.Layers)
			}
		})
	}
}

func TestRunFailsTheLayer(t *testing.T) {
	cases := []struct {
		name  string
		fake  *fake
		calls int
		err   string
	}{
		{"provider error", &fake{err: errors.New("boom")}, 10, "boom"},
		{"budget exhausted", &fake{replies: []llm.Message{read("SKILL.md"), read("helper.sh")}}, 1, "llm: request budget exhausted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := found("DROP", "SKILL.md", 3, finding.High)
			s := newState(t, original)
			run(s, tc.fake, tc.calls)
			if !reflect.DeepEqual(s.Findings, []finding.Finding{original}) {
				t.Errorf("findings %+v, want %+v untouched", s.Findings, original)
			}
			if len(s.Layers) != 2 || s.Layers[1].Status != pipeline.Failed || s.Layers[1].Error != tc.err {
				t.Errorf("layers %+v, want judge failed with %q", s.Layers, tc.err)
			}
		})
	}
}

func TestRunSkipsTheModelWithoutFindings(t *testing.T) {
	s := newState(t)
	provider := &fake{}
	run(s, provider, 10)
	if provider.calls != 0 {
		t.Errorf("model requests %d, want none", provider.calls)
	}
	if !reflect.DeepEqual(s.Layers, []pipeline.LayerStatus{{Layer: "walk", Status: "done"}, {Layer: "judge", Status: "done"}}) {
		t.Errorf("layers %+v; want a done layer", s.Layers)
	}
}

func TestRunJudgesOnlyFindingsOfDiscoveryRules(t *testing.T) {
	text, fact := found("TEXT", "SKILL.md", 3, finding.High), found("FACT", "SKILL.md", 4, finding.High)
	cases := []struct {
		name     string
		findings []finding.Finding
		calls    int
		want     []finding.Finding
	}{
		{"static and reference findings alone reach no model", []finding.Finding{text, fact}, 0, []finding.Finding{text, fact}},
		{"the discovery finding is D1 and the others stay untouched", []finding.Finding{text, fact, found("DROP", "SKILL.md", 5, finding.High)}, 1, []finding.Finding{text, fact, {
			Rule: "DROP", Severity: finding.High, Location: finding.Location{File: "SKILL.md", Line: 5}, Evidence: "evidence",
			Judgement: &finding.Judgement{State: finding.Dismissed, From: finding.High, Confidence: 4, Reason: "because"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, tc.findings...)
			provider := &fake{replies: []llm.Message{submit(verdict("D1", "dismiss", 4))}}
			run(s, provider, 10)
			if provider.calls != tc.calls || !reflect.DeepEqual(s.Findings, tc.want) {
				t.Errorf("model requests %d, findings %+v; want %d, %+v", provider.calls, s.Findings, tc.calls, tc.want)
			}
		})
	}
}

func TestRunListsSeverestFileFirstAndServesFiles(t *testing.T) {
	s := newState(t, found("DROP", "SKILL.md", 3, finding.Low), found("DROP", "helper.sh", 2, finding.Critical))
	run(s, &fake{replies: []llm.Message{read("helper.sh"), submit(verdict("D1", "critical", 4), verdict("D2", "low", 1))}}, 10)
	if s.Findings[1].Judgement == nil || s.Findings[1].Judgement.State != finding.Confirmed || s.Findings[0].Judgement != nil {
		t.Errorf("findings %+v, want only helper.sh confirmed", s.Findings)
	}
}

func TestRunRefusesSubmitUntilEveryFindingHasAVerdict(t *testing.T) {
	s := newState(t, found("DROP", "SKILL.md", 3, finding.Low), found("DROP", "helper.sh", 2, finding.Critical))
	run(s, &fake{replies: []llm.Message{submit(verdict("D1", "critical", 4)), submit(verdict("D2", "dismiss", 4))}}, 10)
	if s.Findings[1].Judgement == nil || s.Findings[1].Judgement.State != finding.Confirmed || s.Findings[0].Judgement == nil || s.Findings[0].Judgement.State != finding.Dismissed {
		t.Errorf("findings %+v, want helper.sh confirmed and SKILL.md dismissed", s.Findings)
	}
	if len(s.Layers) != 2 || s.Layers[1].Status != pipeline.Done {
		t.Errorf("layers %+v, want a done judge layer", s.Layers)
	}
}
