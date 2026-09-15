// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
)

func TestSafeDialerRefusesInternalAddresses(t *testing.T) {
	cases := []struct {
		name    string
		address string
	}{
		{"loopback", "127.0.0.1:1"},
		{"private", "10.0.0.1:1"},
		{"link local", "169.254.169.254:1"},
		{"ipv6 unique local", "[fd00::1]:1"},
		{"listed reserved range", "203.0.113.9:1"},
		{"ipv4-mapped", "[::ffff:198.18.0.1]:1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refcheck.SafeDialer().DialContext(context.Background(), "tcp", tc.address)
			if err == nil || !strings.Contains(err.Error(), "not a public address") {
				t.Errorf("err = %v, want a refusal", err)
			}
		})
	}
}
