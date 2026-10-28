// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package rules

import "github.com/bickus/skill-verdict/pkg/config"

func Settings(s *Set) map[string]map[string]config.RuleSetting {
	out := make(map[string]map[string]config.RuleSetting)
	for id, r := range s.ByID {
		category, severity, enabled, interrupt := r.Category, r.Severity, r.Enabled, r.Interrupt
		downgrade, floor, instructions := r.Judge.Downgrade, r.Judge.DowngradeFloor, r.Judge.Instructions
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
			Judge:       &config.JudgeSetting{Downgrade: &downgrade, DowngradeFloor: &floor, Instructions: &instructions},
			Attribution: r.Attribution,
		}
		if d := r.Decision; d != nil {
			subject, question := string(d.Subject), d.Question
			setting.Decision = &config.DecisionSetting{Subject: &subject, Question: &question, Options: d.Options, Dismiss: d.Dismiss, Confirm: d.Confirm}
			if bar := d.MinProbability; bar > 0 {
				setting.Decision.MinProbability = &bar
			}
		}
		out[r.Layer][id] = setting
	}
	return out
}
