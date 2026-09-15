// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package chatgpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bickus/skill-verdict/pkg/llm"
)

const maxLineBytes = 8 << 20

type event struct {
	Type     string          `json:"type"`
	Response response        `json:"response"`
	Item     json.RawMessage `json:"item"`
	Message  string          `json:"message"`
}

type response struct {
	Output []json.RawMessage `json:"output"`
	Usage  struct {
		InputTokens        int `json:"input_tokens"`
		InputTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		OutputTokens        int `json:"output_tokens"`
		OutputTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

type item struct {
	Type      string    `json:"type"`
	CallID    string    `json:"call_id"`
	Name      string    `json:"name"`
	Arguments string    `json:"arguments"`
	Content   []content `json:"content"`
}

func parse(data []byte) (llm.Message, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), maxLineBytes)
	var items []json.RawMessage
	for sc.Scan() {
		payload, ok := strings.CutPrefix(sc.Text(), "data:")
		if !ok {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &ev); err != nil {
			return llm.Message{}, fmt.Errorf("llm: stream event: %w", err)
		}
		if ev.Type == "response.output_item.done" {
			items = append(items, ev.Item)
			continue
		}
		msg, done, err := terminal(ev, items)
		if done || err != nil {
			return msg, err
		}
	}
	if err := sc.Err(); err != nil {
		return llm.Message{}, fmt.Errorf("llm: stream: %w", err)
	}
	return llm.Message{}, errors.New("llm: stream ended without a completed response")
}

func terminal(ev event, items []json.RawMessage) (llm.Message, bool, error) {
	switch ev.Type {
	case "response.completed":
		if len(items) > 0 {
			ev.Response.Output = items
		}
		msg, err := assemble(ev.Response)
		return msg, true, err
	case "response.incomplete":
		reason := "unknown reason"
		if ev.Response.IncompleteDetails != nil {
			reason = ev.Response.IncompleteDetails.Reason
		}
		return llm.Message{}, true, fmt.Errorf("llm: response incomplete: %s", reason)
	case "response.failed":
		reason := "unknown error"
		if ev.Response.Error != nil {
			reason = ev.Response.Error.Message
		}
		return llm.Message{}, true, fmt.Errorf("llm: response failed: %s", reason)
	case "error":
		return llm.Message{}, true, fmt.Errorf("llm: %s", ev.Message)
	}
	return llm.Message{}, false, nil
}

func assemble(r response) (llm.Message, error) {
	native, err := json.Marshal(r.Output)
	if err != nil {
		return llm.Message{}, err
	}
	msg := llm.Message{Role: llm.Assistant, Native: native}
	var text strings.Builder
	for _, raw := range r.Output {
		var it item
		if err := json.Unmarshal(raw, &it); err != nil {
			return llm.Message{}, fmt.Errorf("llm: output item: %w", err)
		}
		switch it.Type {
		case "message":
			for _, c := range it.Content {
				if c.Type == "output_text" {
					text.WriteString(c.Text)
				}
			}
		case "function_call":
			msg.Calls = append(msg.Calls, llm.Call{ID: it.CallID, Name: it.Name, Arguments: llm.Arguments(it.Arguments)})
		}
	}
	msg.Text = text.String()
	u := r.Usage
	msg.Usage = llm.Usage{Input: u.InputTokens, Cached: u.InputTokensDetails.CachedTokens, Output: u.OutputTokens, Reasoning: u.OutputTokensDetails.ReasoningTokens}
	return msg, nil
}
