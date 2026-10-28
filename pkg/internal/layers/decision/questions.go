// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"embed"

	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/llm/decisions"
	"github.com/bickus/skill-verdict/pkg/rules"
)

//go:embed *.md *.json
var files embed.FS

func question(d *rules.DecisionPolicy, subject string) decisions.Question {
	return decisions.Question{Options: d.Options, Instructions: map[string]string{
		"question": d.Question,
		"subject":  subject,
		"warning":  llmcall.Text(files, "warning.md"),
	}}
}

func chars(q decisions.Question) int {
	n := 0
	for _, m := range []map[string]string{q.Instructions, q.Options} {
		for k, v := range m {
			n += len(k) + len(v)
		}
	}
	return n
}
