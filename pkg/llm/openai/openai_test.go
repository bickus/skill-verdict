// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/openai"
)

const okBody = `{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`

type server struct {
	t    *testing.T
	body string
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.t.Error(err)
	}
	reply := s.body
	if reply == "" {
		reply = okBody
	}
	if _, err := io.WriteString(w, reply); err != nil {
		s.t.Error(err)
	}
}

func newClient(t *testing.T, s *server) *openai.Client {
	t.Helper()
	s.t = t
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	cfg := config.Default().LLM
	cfg.BaseURL = srv.URL + "/"
	cfg.Model = "test-model"
	cfg.MaxOutputTokens = 100
	cfg.Timeout = config.Duration(5 * time.Second)
	return openai.New(cfg, "", &http.Client{})
}

func TestCompleteParsesReply(t *testing.T) {
	message := `{"role":"assistant","content":"look","reasoning":"r","tool_calls":[{"id":"c1","type":"function","function":{"name":"read","arguments":"{\"path\":\"a\"}"}},{"id":"c2","type":"function","function":{"name":"submit_result","arguments":""}}]}`
	body := `{"choices":[{"message":` + message + `,"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":50,"completion_tokens":7,"prompt_tokens_details":{"cached_tokens":40},"completion_tokens_details":{"reasoning_tokens":3}}}`
	s := &server{body: body}
	c := newClient(t, s)
	got, err := c.Complete(context.Background(), llm.Request{})
	if err != nil {
		t.Fatal(err)
	}
	want := llm.Message{Role: llm.Assistant, Text: "look", Native: json.RawMessage(message),
		Calls: []llm.Call{{ID: "c1", Name: "read", Arguments: json.RawMessage(`{"path":"a"}`)}, {ID: "c2", Name: "submit_result", Arguments: json.RawMessage(`{}`)}},
		Usage: llm.Usage{Input: 50, Cached: 40, Output: 7, Reasoning: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestCompleteRejectsBadReplies(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"cut at the token limit", `{"choices":[{"message":{"content":"x"},"finish_reason":"length"}]}`, "output token limit"},
		{"no choices", `{"choices":[]}`, "without choices"},
		{"not JSON", `<html>`, "llm:"},
		{"message of the wrong shape", `{"choices":[{"message":[1]}]}`, "message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, &server{body: tc.body})
			_, err := c.Complete(context.Background(), llm.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err %v, want one containing %q", err, tc.want)
			}
		})
	}
}
