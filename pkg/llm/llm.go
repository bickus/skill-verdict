// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package llm

import (
	"context"
	"encoding/json"
	"strings"
)

type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
	ToolRole  Role = "tool"
)

type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type Call struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

func Arguments(s string) json.RawMessage {
	if strings.TrimSpace(s) == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}

type Result struct {
	CallID string
	Output string
}

type Usage struct {
	Input     int
	Cached    int
	Output    int
	Reasoning int
}

func (u Usage) Add(o Usage) Usage {
	return Usage{Input: u.Input + o.Input, Cached: u.Cached + o.Cached, Output: u.Output + o.Output, Reasoning: u.Reasoning + o.Reasoning}
}

func (u Usage) Context() int {
	return u.Input + u.Output
}

type Message struct {
	Role    Role
	Text    string
	Calls   []Call
	Results []Result
	Native  json.RawMessage
	Usage   Usage
}

type Request struct {
	System   string
	Tools    []Tool
	Messages []Message
}

type Provider interface {
	Complete(ctx context.Context, req Request) (Message, error)
}
