// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type githubCase struct {
	name    string
	value   string
	routes  map[string]route
	maxBody int64
	want    expect
}

func githubCases() []githubCase {
	old := userRoute("acme", daysAgo(3000))
	return []githubCase{
		{"owner missing", "ghost", map[string]route{}, 0,
			expect{status: refs.Checked, findings: []string{"GITHUB-OWNER-MISSING"}, facts: map[string]string{"owner": "missing"}}},
		{"owner present repo missing", "acme/gone", map[string]route{"api.github.com/users/acme": old}, 0,
			expect{status: refs.Checked, findings: []string{"GITHUB-REPO-MISSING"}, facts: map[string]string{"repo": "missing", "owner_type": "User"}}},
		{"owner renamed", "acme/tool", map[string]route{"api.github.com/users/acme": old, "api.github.com/repos/acme/tool": repoRoute("newname", 10, false, false)}, 0,
			expect{status: refs.Checked, findings: []string{"GITHUB-OWNER-RENAMED"}, facts: map[string]string{"owner_login": "newname"}}},
		{"owner login differs only in case", "acme/tool", map[string]route{"api.github.com/users/acme": old, "api.github.com/repos/acme/tool": repoRoute("Acme", 10, false, false)}, 0,
			expect{status: refs.Checked, facts: map[string]string{"stars": "10", "repo_created": "2020-01-02"}}},
		{"rate limited", "acme/tool", map[string]route{"api.github.com/users/acme": {status: http.StatusForbidden, body: "{}", headers: map[string]string{"X-RateLimit-Remaining": "0"}}}, 0,
			expect{status: refs.NotChecked, detail: "rate limit"}},
		{"secondary rate limit", "acme/tool", map[string]route{"api.github.com/users/acme": {status: http.StatusTooManyRequests, body: "{}"}}, 0,
			expect{status: refs.NotChecked, detail: "rate limit"}},
		{"forbidden", "acme/tool", map[string]route{"api.github.com/users/acme": {status: http.StatusForbidden, body: "{}"}}, 0,
			expect{status: refs.Failed, detail: "HTTP 403"}},
		{"server error", "acme/tool", map[string]route{"api.github.com/users/acme": {status: http.StatusInternalServerError, body: "{}"}}, 0,
			expect{status: refs.Failed, detail: "HTTP 500"}},
		{"young owner", "kid/tool", map[string]route{"api.github.com/users/kid": userRoute("kid", daysAgo(10)), "api.github.com/repos/kid/tool": repoRoute("kid", 2, true, true)}, 0,
			expect{status: refs.Checked, findings: []string{"GITHUB-OWNER-NEW"}, facts: map[string]string{"stars": "2", "fork": "true", "archived": "true"}}},
		{"owner only", "acme", map[string]route{"api.github.com/users/acme": old}, 0,
			expect{status: refs.Checked, facts: map[string]string{"owner_type": "User", "owner_repos": "3"}}},
		{"oversized body", "big", map[string]route{"api.github.com/users/big": {status: http.StatusOK, body: strings.Repeat("x", 100)}}, 32,
			expect{status: refs.Failed, detail: "body over"}},
	}
}

func TestGitHubCollect(t *testing.T) {
	for _, tc := range githubCases() {
		t.Run(tc.name, func(t *testing.T) {
			n := newNet(t, tc.routes)
			cfg := testConfig()
			if tc.maxBody > 0 {
				cfg.MaxBodyBytes = tc.maxBody
			}
			rep := refcheck.NewGitHub(n.Client, cfg, "").Collect(context.Background(), refs.Ref{Kind: refs.GitHub, Value: tc.value})
			verify(t, rep, tc.want)
		})
	}
}

func TestGitHubToken(t *testing.T) {
	cases := []struct {
		name  string
		token string
		want  string
	}{
		{"token sent as bearer", "tok", "Bearer tok"},
		{"no header without token", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := newNet(t, map[string]route{"api.github.com/users/acme": userRoute("acme", daysAgo(3000))})
			refcheck.NewGitHub(n.Client, testConfig(), tc.token).Collect(context.Background(), refs.Ref{Kind: refs.GitHub, Value: "acme"})
			if got := n.authorization("api.github.com/users/acme"); got != tc.want {
				t.Errorf("authorization %q, want %q", got, tc.want)
			}
		})
	}
}
