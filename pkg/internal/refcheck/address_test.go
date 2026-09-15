// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

func TestAddressCollect(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  expect
	}{
		{"public address", "137.117.157.128", expect{status: refs.Checked, findings: []string{"IP-ADDRESS-PUBLIC"}, facts: map[string]string{"public": "true"}}},
		{"private address", "10.0.0.1", expect{status: refs.Checked, facts: map[string]string{"public": "false"}}},
		{"malformed value", "300.1.1.1", expect{status: refs.Failed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := refcheck.NewAddress().Collect(context.Background(), refs.Ref{Kind: refs.Address, Value: tc.value})
			verify(t, rep, tc.want)
		})
	}
}
