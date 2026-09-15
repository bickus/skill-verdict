// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package finding

import "fmt"

type Severity string

const (
	Low      Severity = "low"
	Medium   Severity = "medium"
	High     Severity = "high"
	Critical Severity = "critical"
)

func (s Severity) Rank() int {
	switch s {
	case Low:
		return 1
	case Medium:
		return 2
	case High:
		return 3
	case Critical:
		return 4
	}
	return 0
}

func Severities() []Severity {
	return []Severity{Critical, High, Medium, Low}
}

func ParseSeverity(s string) (Severity, error) {
	switch Severity(s) {
	case Low, Medium, High, Critical:
		return Severity(s), nil
	}
	return "", fmt.Errorf("unknown severity %q", s)
}
