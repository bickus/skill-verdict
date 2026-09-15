// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package openai

import (
	"encoding/json"
	"fmt"

	"github.com/bickus/skill-verdict/pkg/llm"
)

type text struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolResult struct {
	Role       string `json:"role"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

func lower(req llm.Request) ([]json.RawMessage, error) {
	var out []json.RawMessage
	if req.System != "" {
		data, err := json.Marshal(text{Role: "system", Content: req.System})
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	for _, m := range req.Messages {
		items, err := encode(m)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func encode(m llm.Message) ([]json.RawMessage, error) {
	switch m.Role {
	case llm.User:
		return one(text{Role: "user", Content: m.Text})
	case llm.Assistant:
		native, err := replay(m)
		return []json.RawMessage{native}, err
	case llm.ToolRole:
		var out []json.RawMessage
		for _, r := range m.Results {
			data, err := json.Marshal(toolResult{Role: "tool", ToolCallID: r.CallID, Content: r.Output})
			if err != nil {
				return nil, err
			}
			out = append(out, data)
		}
		return out, nil
	}
	return nil, fmt.Errorf("llm: unknown role %q", m.Role)
}

func one(v any) ([]json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []json.RawMessage{data}, nil
}

func replay(m llm.Message) (json.RawMessage, error) {
	var fields map[string]any
	if err := json.Unmarshal(m.Native, &fields); err != nil {
		return nil, fmt.Errorf("llm: native message: %w", err)
	}
	for k, v := range fields {
		if v == nil {
			delete(fields, k)
		}
	}
	return json.Marshal(fields)
}
