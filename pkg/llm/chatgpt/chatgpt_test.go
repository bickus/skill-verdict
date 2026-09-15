// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package chatgpt_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/llm"
	"github.com/bickus/skill-verdict/pkg/llm/chatgpt"
)

type rewrite struct {
	target *url.URL
}

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

type server struct {
	t      *testing.T
	stream string
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.t.Error(err)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	if _, err := io.WriteString(w, s.stream); err != nil {
		s.t.Error(err)
	}
}

func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("event: x\ndata: " + e + "\n\n")
	}
	return b.String()
}

const (
	reasoningItem = `{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"ENC"}`
	callItem      = `{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read","arguments":"{\"path\":\"a\"}"}`
	messageItem   = `{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"Paris"},{"type":"output_text","text":" is sunny."}]}`
	usage         = `{"input_tokens":120,"input_tokens_details":{"cached_tokens":100},"output_tokens":30,"output_tokens_details":{"reasoning_tokens":12}}`
)

func completed() string {
	return sse(`{"type":"response.created","response":{}}`,
		`{"type":"response.completed","response":{"output":[`+reasoningItem+`,`+callItem+`,`+messageItem+`],"usage":`+usage+`}}`)
}

func streamed() string {
	return sse(`{"type":"response.created","response":{}}`,
		`{"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":""}}`,
		`{"type":"response.output_item.done","item":`+reasoningItem+`}`,
		`{"type":"response.output_item.done","item":`+callItem+`}`,
		`{"type":"response.output_item.done","item":`+messageItem+`}`,
		`{"type":"response.completed","response":{"output":[],"usage":`+usage+`}}`)
}

func newClient(t *testing.T, s *server) *chatgpt.Client {
	t.Helper()
	s.t = t
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().LLM
	cfg.Model = "gpt-test"
	cfg.Timeout = config.Duration(5 * time.Second)
	return chatgpt.New(cfg, "k", &http.Client{Transport: rewrite{target: target}})
}

func TestCompleteParsesStream(t *testing.T) {
	want := llm.Message{Role: llm.Assistant, Text: "Paris is sunny.",
		Calls:  []llm.Call{{ID: "call_1", Name: "read", Arguments: json.RawMessage(`{"path":"a"}`)}},
		Native: json.RawMessage(`[` + reasoningItem + `,` + callItem + `,` + messageItem + `]`),
		Usage:  llm.Usage{Input: 120, Cached: 100, Output: 30, Reasoning: 12}}
	cases := []struct {
		name   string
		stream string
	}{
		{"items in the completed event", completed()},
		{"items streamed one by one", streamed()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, &server{stream: tc.stream})
			got, err := c.Complete(context.Background(), llm.Request{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %+v\nwant %+v", got, want)
			}
		})
	}
}

func TestCompleteRejectsBadStreams(t *testing.T) {
	cases := []struct {
		name   string
		stream string
		want   string
	}{
		{"incomplete response", sse(`{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`), "max_output_tokens"},
		{"failed response", sse(`{"type":"response.failed","response":{"error":{"message":"quota"}}}`), "quota"},
		{"error event", sse(`{"type":"error","message":"bad request"}`), "bad request"},
		{"stream without completion", sse(`{"type":"response.created","response":{}}`), "without a completed response"},
		{"garbage data", "data: {not json\n\n", "stream event"},
		{"empty body", "", "without a completed response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, &server{stream: tc.stream})
			_, err := c.Complete(context.Background(), llm.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err %v, want one containing %q", err, tc.want)
			}
		})
	}
}
