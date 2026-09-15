// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package chatgpt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/transport"
)

const (
	endpoint      = "https://chatgpt.com/backend-api/codex/responses"
	defaultEffort = "medium"
	originator    = "skill-verdict"
)

type Client struct {
	cfg     config.LLMConfig
	key     string
	account string
	session string
	http    transport.Client
}

func New(cfg config.LLMConfig, key string, client *http.Client) *Client {
	return &Client{
		cfg:     cfg,
		key:     key,
		account: accountID(key),
		session: sessionID(),
		http:    transport.Client{HTTP: client, Timeout: time.Duration(cfg.Timeout)},
	}
}

func sessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

type tool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict"`
}

type reasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

type request struct {
	Model             string            `json:"model"`
	Instructions      string            `json:"instructions"`
	Input             []json.RawMessage `json:"input"`
	Tools             []tool            `json:"tools,omitempty"`
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
	Store             bool              `json:"store"`
	Stream            bool              `json:"stream"`
	Include           []string          `json:"include"`
	Reasoning         reasoning         `json:"reasoning"`
	PromptCacheKey    string            `json:"prompt_cache_key"`
}

func (c *Client) Complete(ctx context.Context, req llm.Request) (llm.Message, error) {
	input, err := lower(req.Messages)
	if err != nil {
		return llm.Message{}, err
	}
	effort := c.cfg.ReasoningEffort
	if effort == "" {
		effort = defaultEffort
	}
	wire := request{
		Model:             c.cfg.Model,
		Instructions:      req.System,
		Input:             input,
		ParallelToolCalls: true,
		Store:             false,
		Stream:            true,
		Include:           []string{"reasoning.encrypted_content"},
		Reasoning:         reasoning{Effort: effort, Summary: "auto"},
		PromptCacheKey:    c.session,
	}
	for _, t := range req.Tools {
		wire.Tools = append(wire.Tools, tool{Type: "function", Name: t.Name, Description: t.Description, Parameters: t.Parameters})
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return llm.Message{}, err
	}
	data, err := c.http.Post(ctx, endpoint, c.headers(), body)
	if err != nil {
		return llm.Message{}, err
	}
	return parse(data)
}

func (c *Client) headers() http.Header {
	h := http.Header{}
	h.Set("Accept", "text/event-stream")
	h.Set("Authorization", "Bearer "+c.key)
	h.Set("OpenAI-Beta", "responses=experimental")
	h.Set("User-Agent", originator)
	h.Set("originator", originator)
	h.Set("session_id", c.session)
	if c.account != "" {
		h.Set("ChatGPT-Account-Id", c.account)
	}
	return h
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type message struct {
	Type    string    `json:"type"`
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type callOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

func lower(messages []llm.Message) ([]json.RawMessage, error) {
	var out []json.RawMessage
	for _, m := range messages {
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
		return marshal(message{Type: "message", Role: "user", Content: []content{{Type: "input_text", Text: m.Text}}})
	case llm.Assistant:
		return replay(m)
	case llm.ToolRole:
		items := make([]any, 0, len(m.Results))
		for _, r := range m.Results {
			items = append(items, callOutput{Type: "function_call_output", CallID: r.CallID, Output: r.Output})
		}
		return marshal(items...)
	}
	return nil, fmt.Errorf("llm: unknown role %q", m.Role)
}

func marshal(items ...any) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(items))
	for _, it := range items {
		data, err := json.Marshal(it)
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}

func replay(m llm.Message) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(m.Native, &items); err != nil {
		return nil, fmt.Errorf("llm: native items: %w", err)
	}
	return items, nil
}
