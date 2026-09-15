// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bickus/skill-verdict/pkg/llm"
)

var (
	ErrBudget     = errors.New("llm: request budget exhausted")
	ErrCompaction = errors.New("llm: context reached the compaction threshold")
)

type Tool interface {
	Spec() llm.Tool
	Call(ctx context.Context, args json.RawMessage) (string, error)
	Final() bool
}

type Loop struct {
	Provider     llm.Provider
	System       string
	Tools        []Tool
	Reminder     string
	Calls        int
	CompactionAt int
}

type Outcome struct {
	Usage    llm.Usage
	Requests int
}

type run struct {
	loop    Loop
	tools   map[string]Tool
	outcome Outcome
}

func (l Loop) Run(ctx context.Context, user string) (Outcome, error) {
	r := &run{loop: l, tools: make(map[string]Tool, len(l.Tools))}
	for _, t := range l.Tools {
		r.tools[t.Spec().Name] = t
	}
	req := llm.Request{System: l.System, Tools: l.specs(), Messages: []llm.Message{{Role: llm.User, Text: user}}}
	for {
		if r.outcome.Requests >= l.Calls {
			return r.outcome, ErrBudget
		}
		reply, err := l.Provider.Complete(ctx, req)
		if err != nil {
			return r.outcome, err
		}
		r.outcome.Requests++
		r.outcome.Usage = r.outcome.Usage.Add(reply.Usage)
		req.Messages = append(req.Messages, reply)
		if size := reply.Usage.Context(); l.CompactionAt > 0 && size >= l.CompactionAt {
			return r.outcome, fmt.Errorf("%w: %d tokens, threshold %d", ErrCompaction, size, l.CompactionAt)
		}
		next, done := r.answer(ctx, reply)
		if done {
			return r.outcome, nil
		}
		req.Messages = append(req.Messages, next)
	}
}

func (l Loop) specs() []llm.Tool {
	out := make([]llm.Tool, 0, len(l.Tools))
	for _, t := range l.Tools {
		out = append(out, t.Spec())
	}
	return out
}

func (r *run) answer(ctx context.Context, reply llm.Message) (llm.Message, bool) {
	if len(reply.Calls) == 0 {
		return llm.Message{Role: llm.User, Text: r.loop.Reminder}, false
	}
	msg := llm.Message{Role: llm.ToolRole}
	done := false
	for _, c := range reply.Calls {
		t := r.tools[c.Name]
		if t == nil {
			msg.Results = append(msg.Results, llm.Result{CallID: c.ID, Output: fmt.Sprintf("error: unknown tool %q", c.Name)})
			continue
		}
		output, err := t.Call(ctx, c.Arguments)
		switch {
		case err != nil:
			output = "error: " + err.Error()
		case t.Final():
			done = true
			continue
		}
		msg.Results = append(msg.Results, llm.Result{CallID: c.ID, Output: output})
	}
	return msg, done
}
