// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

func pacedNet(t *testing.T, routes map[string]route, retries ...time.Duration) (*fakeNet, config.ReferencesConfig) {
	t.Helper()
	cfg := testConfig()
	for _, r := range retries {
		cfg.RetryDelays = append(cfg.RetryDelays, config.Duration(r))
	}
	n := newNet(t, routes)
	n.Client = refcheck.Paced(n.raw, cfg)
	return n, cfg
}

func TestPacedRetriesRateLimit(t *testing.T) {
	n, cfg := pacedNet(t, map[string]route{
		"api.github.com/users/acme": {status: http.StatusOK, body: `{"login":"acme","type":"User"}`, limit: 2},
	}, time.Millisecond, time.Millisecond)
	rep := refcheck.NewGitHub(n.Client, cfg, "").Collect(t.Context(), refs.Ref{Kind: refs.GitHub, Value: "acme"})
	verify(t, rep, expect{status: refs.Checked, facts: map[string]string{"owner_type": "User"}})
}

func TestPacedClosesHostAfterLastRetry(t *testing.T) {
	n, cfg := pacedNet(t, map[string]route{
		"api.github.com/users/aaa": {status: http.StatusOK, body: `{"login":"aaa","type":"User"}`, limit: 5},
		"api.github.com/users/bbb": {status: http.StatusOK, body: `{"login":"bbb","type":"User"}`},
		"registry.npmjs.org/eee":   {status: http.StatusNotFound, body: "{}"},
	}, time.Millisecond)
	cfg.Concurrency = 1
	reports := run(t, githubRunner(n, cfg), "runner/ratelimit")
	verify(t, reports[0], expect{status: refs.NotChecked, detail: "rate limit"})
	verify(t, reports[1], expect{status: refs.NotChecked, detail: "rate limit"})
}

func TestPacedRequestsOneOwnerOnceForTwoRepositories(t *testing.T) {
	n, cfg := pacedNet(t, map[string]route{
		"api.github.com/users/acme":     userRoute("acme", daysAgo(3000)),
		"api.github.com/repos/acme/one": repoRoute("acme", 10, false, false),
		"api.github.com/repos/acme/two": repoRoute("acme", 20, false, false),
	})
	reports := run(t, githubRunner(n, cfg), "runner/owner")
	verify(t, reports[0], expect{status: refs.Checked, facts: map[string]string{"stars": "10"}})
	verify(t, reports[1], expect{status: refs.Checked, facts: map[string]string{"stars": "20"}})
	if got := n.count("api.github.com/users/acme"); got != 1 {
		t.Errorf("owner requests %d, want 1", got)
	}
}
