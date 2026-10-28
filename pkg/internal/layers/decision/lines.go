// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package decision

import (
	"context"
	"encoding/json"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/llmcall"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const linesID = "lines"

func (r *run) lines(ctx context.Context, rule rules.Rule, f finding.Finding) (outcome, error) {
	values, _ := located(r.state, f.Location, f.Evidence)
	if values == nil {
		return outcome{skipped: "the line was not found"}, nil
	}
	state, err := json.Marshal(llmcall.Render(files, "lines.md", values))
	if err != nil {
		return outcome{}, err
	}
	return r.send(ctx, rule.Decision, linesID, state, question(rule.Decision, f.Evidence))
}
