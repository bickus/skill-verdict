// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decisions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/transport"
)

const choice = "choice"

type Client struct {
	cfg  config.DecisionConfig
	key  string
	http transport.Client
}

func New(cfg config.DecisionConfig, key string, client *http.Client) *Client {
	return &Client{cfg: cfg, key: key, http: transport.Client{HTTP: client, Timeout: time.Duration(cfg.Timeout)}}
}

type Question struct {
	Instructions map[string]string
	Options      map[string]string
}

type Answer struct {
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type wireQuestion struct {
	Type         string            `json:"type"`
	Instructions map[string]string `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type request struct {
	Model     string                  `json:"model"`
	State     json.RawMessage         `json:"state"`
	Questions map[string]wireQuestion `json:"questions"`
}

type reply struct {
	Answers map[string]Answer `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *Client) Decide(ctx context.Context, state json.RawMessage, questions map[string]Question) (map[string]Answer, llm.Usage, error) {
	wire := request{Model: c.cfg.Model, State: state, Questions: make(map[string]wireQuestion, len(questions))}
	for id, q := range questions {
		wire.Questions[id] = wireQuestion{Type: choice, Instructions: q.Instructions, Criteria: q.Options}
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, llm.Usage{}, err
	}
	header := http.Header{}
	header.Set("Accept", "application/json")
	if c.key != "" {
		header.Set("Authorization", "Bearer "+c.key)
	}
	data, err := c.http.Post(ctx, strings.TrimSuffix(c.cfg.BaseURL, "/")+"/decisions", header, body)
	if err != nil {
		return nil, llm.Usage{}, err
	}
	var r reply
	if err = json.Unmarshal(data, &r); err != nil {
		return nil, llm.Usage{}, fmt.Errorf("decisions: %w", err)
	}
	return r.Answers, llm.Usage{Input: r.Usage.InputTokens, Output: r.Usage.OutputTokens}, nil
}
