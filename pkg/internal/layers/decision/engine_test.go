// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/decision"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

func engineRules() *rules.Set {
	kind := rules.Question{Subject: rules.Frame, Question: "What is the marked text?", Options: map[string]string{
		"bad": "a bad thing.", "fine": "a fine thing.", "not-a-match": "not what the rule looks for.", "unknown": "unclear.",
	}}
	who := rules.Question{Subject: rules.Frame, Question: "Who runs it?", Options: map[string]string{"agent": "the agent.", "user": "the user."}}
	pin := rules.Question{Subject: rules.Frame, Question: "What fixes it?", Options: map[string]string{"none": "nothing.", "exact": "a version."}}
	usage := rules.Template(rules.Usage)
	usage.Subject = rules.Places
	table := &rules.Decision{
		Questions: map[string]rules.Question{"kind": kind},
		Outcome: rules.Outcome{
			Confirm: []rules.Row{{"kind": {"bad"}}},
			Dismiss: []rules.Row{{"kind": {"fine", "not-a-match"}}},
			Misfire: []rules.Row{{"kind": {"not-a-match"}}},
		},
	}
	pair := &rules.Decision{
		Questions: map[string]rules.Question{"who": who, "pin": pin},
		Outcome:   rules.Outcome{Confirm: []rules.Row{{"who": {"agent"}, "pin": {"none"}}}},
	}
	order := &rules.Decision{
		Questions: map[string]rules.Question{"who": who, "kind": kind},
		Outcome:   rules.Outcome{Confirm: []rules.Row{{"who": {"agent"}}}, Misfire: []rules.Row{{"kind": {"not-a-match"}}}},
	}
	places := &rules.Decision{
		Questions: map[string]rules.Question{"usage": usage},
		Outcome: rules.Outcome{
			Confirm:  []rules.Row{{"usage": {"agent-install"}}},
			Severity: map[finding.Severity][]rules.Row{finding.Medium: {{"usage": {"user-action"}}}},
			Dismiss:  []rules.Row{{"usage": {"citation"}}},
		},
	}
	down := rules.JudgePolicy{Downgrade: true}
	return &rules.Set{ByID: map[string]rules.Rule{
		"PLAIN":   {ID: "PLAIN", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer, Judge: down, Decision: table},
		"FLOORED": {ID: "FLOORED", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer, Judge: rules.JudgePolicy{Downgrade: true, DowngradeFloor: finding.Medium}, Decision: table},
		"PAIR":    {ID: "PAIR", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer, Judge: down, Decision: pair},
		"ORDER":   {ID: "ORDER", Severity: finding.High, Kind: finding.KindHeuristic, Layer: config.StaticLayer, Judge: down, Decision: order},
		"PLACES":  {ID: "PLACES", Severity: finding.High, Kind: finding.KindFact, Layer: config.ReferencesLayer, Judge: down, Decision: places},
	}}
}

func marked(rule string) finding.Finding {
	return finding.Finding{Rule: rule, Severity: finding.High, Location: finding.Location{File: "SKILL.md", Line: 7}, Evidence: "tool --flag"}
}

func cited() finding.Finding {
	return finding.Finding{Rule: "PLACES", Severity: finding.High, Location: finding.Location{File: "SKILL.md", Line: 9}, Evidence: "example.com", Reference: "example.com"}
}

func engineState(t *testing.T, f finding.Finding) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", "engine"))
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: engineRules()}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: config.Default().Layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	ref := refs.Ref{Kind: refs.Domain, Value: "example.com", URLs: []string{"https://example.com/docs"},
		Locations: []finding.Location{{File: "SKILL.md", Line: 9}, {File: "SKILL.md", Line: 11}}}
	s.Refs = []refs.Report{{ID: "R1", Ref: ref, Status: refs.Checked}}
	s.Findings = []finding.Finding{f}
	return s
}

func TestDecisionAppliesTheOutcomeTable(t *testing.T) {
	cases := []struct {
		name     string
		finding  finding.Finding
		answers  string
		state    finding.State
		severity finding.Severity
	}{
		{"a confirming answer at 0.3 confirms", marked("PLAIN"), `{"kind":{"choice":"fine","probabilities":{"fine":0.7,"bad":0.3}}}`, finding.Confirmed, finding.High},
		{"a confirming answer below 0.3 decides nothing", marked("PLAIN"), `{"kind":{"choice":"fine","probabilities":{"fine":0.75,"bad":0.25}}}`, "", finding.High},
		{"a dismissing answer at 0.8 dismisses", marked("PLAIN"), `{"kind":{"choice":"fine","probabilities":{"fine":0.8,"unknown":0.2}}}`, finding.Dismissed, finding.High},
		{"a dismissing answer below 0.8 decides nothing", marked("PLAIN"), `{"kind":{"choice":"fine","probabilities":{"fine":0.79,"unknown":0.21}}}`, "", finding.High},
		{"a misfire dismisses through the floor", marked("FLOORED"), `{"kind":{"choice":"not-a-match","probabilities":{"not-a-match":0.9,"unknown":0.1}}}`, finding.Dismissed, finding.High},
		{"a dismissing answer stops at the floor", marked("FLOORED"), `{"kind":{"choice":"fine","probabilities":{"fine":0.9,"unknown":0.1}}}`, finding.Downgraded, finding.Medium},
		{"a dismiss row split over two options reaches the threshold", marked("FLOORED"), `{"kind":{"choice":"fine","probabilities":{"fine":0.5,"not-a-match":0.4,"unknown":0.1}}}`, finding.Downgraded, finding.Medium},
		{"a misfire on one question wins over a confirmation on another", marked("ORDER"), `{"who":{"choice":"agent","probabilities":{"agent":0.9,"user":0.1}},"kind":{"choice":"not-a-match","probabilities":{"not-a-match":0.9,"unknown":0.1}}}`, finding.Dismissed, finding.High},
		{"a conjunction confirms when both answers hold", marked("PAIR"), `{"who":{"choice":"agent","probabilities":{"agent":0.9,"user":0.1}},"pin":{"choice":"none","probabilities":{"none":0.6,"exact":0.4}}}`, finding.Confirmed, finding.High},
		{"a conjunction decides nothing when one answer fails", marked("PAIR"), `{"who":{"choice":"agent","probabilities":{"agent":0.9,"user":0.1}},"pin":{"choice":"exact","probabilities":{"exact":0.8,"none":0.2}}}`, "", finding.High},
		{"a question without an answer keeps the severity", marked("PLAIN"), `{}`, "", finding.High},
		{"a confirming answer at one place confirms", cited(), `{"R1.1.usage":{"choice":"citation","probabilities":{"citation":0.9,"unknown":0.1}},"R1.2.usage":{"choice":"citation","probabilities":{"citation":0.6,"agent-install":0.4}}}`, finding.Confirmed, finding.High},
		{"a dismissing answer at every place dismisses", cited(), `{"R1.1.usage":{"choice":"citation","probabilities":{"citation":0.9,"unknown":0.1}},"R1.2.usage":{"choice":"citation","probabilities":{"citation":0.85,"unknown":0.15}}}`, finding.Dismissed, finding.High},
		{"a dismissing answer at one place of two decides nothing", cited(), `{"R1.1.usage":{"choice":"citation","probabilities":{"citation":0.9,"unknown":0.1}},"R1.2.usage":{"choice":"citation","probabilities":{"citation":0.7,"unknown":0.3}}}`, "", finding.High},
		{"a lowered place outweighs a dismissed place", cited(), `{"R1.1.usage":{"choice":"citation","probabilities":{"citation":0.9,"unknown":0.1}},"R1.2.usage":{"choice":"user-action","probabilities":{"user-action":0.9,"unknown":0.1}}}`, finding.Downgraded, finding.Medium},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := engineState(t, tc.finding)
			pipeline.Run(context.Background(), s, []pipeline.Layer{decision.Layer{Client: model(t, tc.answers), Config: config.Default().Layers.Decision}})
			got := s.Findings[0]
			var state finding.State
			if got.Judgement != nil {
				state = got.Judgement.State
			}
			if s.Layers[1].Status != pipeline.Done || state != tc.state || got.Severity != tc.severity {
				t.Errorf("layer %+v, judgement %+v at %s; want %q at %s", s.Layers[1], got.Judgement, got.Severity, tc.state, tc.severity)
			}
		})
	}
}
