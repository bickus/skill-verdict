// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

func githubRunner(n *fakeNet, cfg config.ReferencesConfig) *refcheck.Runner {
	return &refcheck.Runner{Collectors: []refcheck.Collector{refcheck.NewGitHub(n.Client, cfg, ""), refcheck.NewPackage(n.Client, cfg)}, Config: cfg}
}

func domainRunner(n *fakeNet, cfg config.ReferencesConfig) *refcheck.Runner {
	return &refcheck.Runner{Collectors: []refcheck.Collector{refcheck.NewDomain(n.Client, cfg)}, Config: cfg}
}

func run(t *testing.T, r *refcheck.Runner, dir string) []refs.Report {
	t.Helper()
	skill := walked(t, dir)
	return r.Run(context.Background(), skill)
}

func TestRunMergesDuplicateReferences(t *testing.T) {
	n := newNet(t, map[string]route{"api.github.com/users/acme": userRoute("acme", daysAgo(3000)), "api.github.com/repos/acme/tool": repoRoute("acme", 10, false, false)})
	reports := run(t, githubRunner(n, testConfig()), "runner/dedup")
	if len(reports) != 1 {
		t.Fatalf("reports %+v, want one", reports)
	}
	want := []finding.Location{{File: "README.md", Line: 2}, {File: "SKILL.md", Line: 3}, {File: "SKILL.md", Line: 5}}
	if !reflect.DeepEqual(reports[0].Ref.Locations, want) {
		t.Errorf("locations %v, want %v", reports[0].Ref.Locations, want)
	}
}

func TestRunCapsReferences(t *testing.T) {
	n := newNet(t, map[string]route{"api.github.com/users/acme": userRoute("acme", daysAgo(3000)), "api.github.com/repos/acme/one": repoRoute("acme", 10, false, false)})
	cfg := testConfig()
	cfg.MaxReferences = 1
	r := githubRunner(n, cfg)
	r.Capped = func(refs.Ref, int) {}
	reports := run(t, r, "runner/cap")
	verify(t, reports[0], expect{status: refs.Checked})
	verify(t, reports[1], expect{status: refs.NotChecked, detail: "cap"})
}

func TestRunMarksRemainderAtBudget(t *testing.T) {
	n := newNet(t, map[string]route{"api.github.com/users/acme": {hold: true}})
	cfg := testConfig()
	cfg.Budget = config.Duration(300 * time.Millisecond)
	cfg.Concurrency = 1
	reports := run(t, githubRunner(n, cfg), "runner/budget")
	if len(reports) != 2 {
		t.Fatalf("reports %+v, want two", reports)
	}
	verify(t, reports[0], expect{status: refs.NotChecked, detail: "budget"})
	verify(t, reports[1], expect{status: refs.NotChecked, detail: "budget"})
}

func TestRunFollowsDerivedReferencesOnce(t *testing.T) {
	n := newNet(t, map[string]route{
		bootstrapKey:                     {status: http.StatusOK, body: bootstrapBody},
		"rdap.test/domain/young.com":     rdapRoute("young.com", daysAgo(30), daysAhead(300), "Jane Doe"),
		"young.com/":                     redirect("https://elsewhere.org/landing"),
		"elsewhere.org/landing":          ok,
		"rdap.test/domain/elsewhere.org": rdapOld("elsewhere.org"),
		"elsewhere.org/":                 redirect("https://third.net/"),
		"third.net/":                     ok,
	})
	reports := run(t, domainRunner(n, testConfig()), "runner/derived")
	if len(reports) != 2 {
		t.Fatalf("reports %+v, want two", reports)
	}
	verify(t, reports[0], expect{status: refs.Checked, findings: []string{"DOMAIN-NEW", "DOMAIN-NEW-REDIRECT"}})
	verify(t, reports[1], expect{status: refs.Checked, facts: map[string]string{"redirect_cross_domain": "third.net"}})
	want := refs.Ref{Kind: refs.Domain, Value: "elsewhere.org", Locations: []finding.Location{{File: "SKILL.md", Line: 3}}, Derived: true}
	if !reflect.DeepEqual(reports[1].Ref, want) {
		t.Errorf("derived reference %+v, want %+v", reports[1].Ref, want)
	}
}

func TestRunMergesDerivedIntoExtractedReference(t *testing.T) {
	n := newNet(t, map[string]route{
		bootstrapKey:                     {status: http.StatusOK, body: bootstrapBody},
		"rdap.test/domain/young.com":     rdapRoute("young.com", daysAgo(30), daysAhead(300), "Jane Doe"),
		"young.com/":                     redirect("https://elsewhere.org/"),
		"rdap.test/domain/elsewhere.org": rdapOld("elsewhere.org"),
		"elsewhere.org/":                 ok,
	})
	reports := run(t, domainRunner(n, testConfig()), "runner/merge")
	if len(reports) != 2 {
		t.Fatalf("reports %+v, want two", reports)
	}
	want := refs.Ref{Kind: refs.Domain, Value: "elsewhere.org", URLs: []string{"https://elsewhere.org/"},
		Locations: []finding.Location{{File: "SKILL.md", Line: 5}, {File: "SKILL.md", Line: 3}}}
	if !reflect.DeepEqual(reports[1].Ref, want) {
		t.Errorf("merged reference %+v, want %+v", reports[1].Ref, want)
	}
}
