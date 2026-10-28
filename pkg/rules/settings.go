// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import "github.com/bickus/skill-verdict/pkg/config"

func Settings(s *Set) map[string]map[string]config.RuleSetting {
	out := make(map[string]map[string]config.RuleSetting)
	for id, r := range s.ByID {
		category, severity, enabled, interrupt := r.Category, r.Severity, r.Enabled, r.Interrupt
		downgrade, floor := r.Judge.Downgrade, r.Judge.DowngradeFloor
		if out[r.Layer] == nil {
			out[r.Layer] = make(map[string]config.RuleSetting)
		}
		setting := config.RuleSetting{
			Title:       r.Title,
			Description: r.Description,
			Kind:        r.Kind,
			Category:    &category,
			Severity:    &severity,
			Enabled:     &enabled,
			Interrupt:   &interrupt,
			Judge:       &config.JudgeSetting{Downgrade: &downgrade, DowngradeFloor: &floor},
			Attribution: r.Attribution,
		}
		out[r.Layer][id] = setting
	}
	return out
}
