// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/transport"
)

type Client struct {
	cfg  config.LLMConfig
	key  string
	http transport.Client
}

func New(cfg config.LLMConfig, key string, client *http.Client) *Client {
	return &Client{cfg: cfg, key: key, http: transport.Client{HTTP: client, Timeout: time.Duration(cfg.Timeout)}}
}

type function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type tool struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type request struct {
	Model             string            `json:"model"`
	Messages          []json.RawMessage `json:"messages"`
	Tools             []tool            `json:"tools,omitempty"`
	ParallelToolCalls bool              `json:"parallel_tool_calls,omitempty"`
	MaxTokens         int               `json:"max_tokens"`
	ReasoningEffort   string            `json:"reasoning_effort,omitempty"`
}

func (c *Client) Complete(ctx context.Context, req llm.Request) (llm.Message, error) {
	messages, err := lower(req)
	if err != nil {
		return llm.Message{}, err
	}
	wire := request{Model: c.cfg.Model, Messages: messages, MaxTokens: c.cfg.MaxOutputTokens, ReasoningEffort: c.cfg.ReasoningEffort}
	for _, t := range req.Tools {
		wire.Tools = append(wire.Tools, tool{Type: "function", Function: function{Name: t.Name, Description: t.Description, Parameters: t.Parameters}})
	}
	wire.ParallelToolCalls = len(wire.Tools) > 0
	body, err := json.Marshal(wire)
	if err != nil {
		return llm.Message{}, err
	}
	header := http.Header{}
	header.Set("Accept", "application/json")
	if c.key != "" {
		header.Set("Authorization", "Bearer "+c.key)
	}
	data, err := c.http.Post(ctx, strings.TrimSuffix(c.cfg.BaseURL, "/")+"/chat/completions", header, body)
	if err != nil {
		return llm.Message{}, err
	}
	return parse(data)
}

type reply struct {
	Choices []struct {
		Message      json.RawMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

type message struct {
	Content   string `json:"content"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

func parse(data []byte) (llm.Message, error) {
	var out reply
	if err := json.Unmarshal(data, &out); err != nil {
		return llm.Message{}, fmt.Errorf("llm: %w", err)
	}
	if len(out.Choices) == 0 {
		return llm.Message{}, errors.New("llm: reply without choices")
	}
	choice := out.Choices[0]
	if choice.FinishReason == "length" {
		return llm.Message{}, errors.New("llm: reply cut at the output token limit")
	}
	var m message
	if err := json.Unmarshal(choice.Message, &m); err != nil {
		return llm.Message{}, fmt.Errorf("llm: message: %w", err)
	}
	msg := llm.Message{Role: llm.Assistant, Text: m.Content, Native: choice.Message}
	for _, tc := range m.ToolCalls {
		msg.Calls = append(msg.Calls, llm.Call{ID: tc.ID, Name: tc.Function.Name, Arguments: llm.Arguments(tc.Function.Arguments)})
	}
	u := out.Usage
	msg.Usage = llm.Usage{Input: u.PromptTokens, Cached: u.PromptTokensDetails.CachedTokens, Output: u.CompletionTokens, Reasoning: u.CompletionTokensDetails.ReasoningTokens}
	return msg, nil
}
