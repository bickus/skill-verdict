// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/agent"
)

type provider struct {
	replies []llm.Message
	err     error
	calls   int
}

func (p *provider) Complete(context.Context, llm.Request) (llm.Message, error) {
	if p.err != nil {
		return llm.Message{}, p.err
	}
	i := p.calls
	p.calls++
	if i >= len(p.replies) {
		return llm.Message{}, errors.New("no scripted reply")
	}
	return p.replies[i], nil
}

type tool struct {
	name  string
	final bool
}

func (t tool) Spec() llm.Tool                                      { return llm.Tool{Name: t.name} }
func (tool) Call(context.Context, json.RawMessage) (string, error) { return "", nil }
func (t tool) Final() bool                                         { return t.final }

func TestRunReturnsOutcomeAndErrors(t *testing.T) {
	providerErr := errors.New("provider failed")
	first := llm.Message{Role: llm.Assistant, Calls: []llm.Call{{Name: "continue"}}, Usage: llm.Usage{Input: 10, Cached: 2, Output: 3, Reasoning: 1}}
	last := llm.Message{Role: llm.Assistant, Calls: []llm.Call{{Name: "finish"}}, Usage: llm.Usage{Input: 20, Cached: 4, Output: 5, Reasoning: 2}}
	cases := []struct {
		name       string
		provider   *provider
		calls      int
		compaction int
		want       agent.Outcome
		wantErr    error
	}{
		{"final tool returns usage", &provider{replies: []llm.Message{first, last}}, 3, 1000, agent.Outcome{Usage: llm.Usage{Input: 30, Cached: 6, Output: 8, Reasoning: 3}, Requests: 2}, nil},
		{"request budget", &provider{replies: []llm.Message{first}}, 1, 1000, agent.Outcome{Usage: first.Usage, Requests: 1}, agent.ErrBudget},
		{"context threshold", &provider{replies: []llm.Message{first}}, 3, 13, agent.Outcome{Usage: first.Usage, Requests: 1}, agent.ErrCompaction},
		{"provider error", &provider{err: providerErr}, 3, 1000, agent.Outcome{}, providerErr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loop := agent.Loop{Provider: tc.provider, Tools: []agent.Tool{tool{name: "continue"}, tool{name: "finish", final: true}}, Calls: tc.calls, CompactionAt: tc.compaction}
			got, err := loop.Run(t.Context(), "task")
			if !reflect.DeepEqual(got, tc.want) || !errors.Is(err, tc.wantErr) {
				t.Errorf("got %+v, err %v; want %+v, err %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
