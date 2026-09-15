// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type packageCase struct {
	name   string
	value  string
	routes map[string]route
	want   expect
}

const (
	npmStats  = "packages.ecosyste.ms/api/v1/registries/npmjs.org/packages/"
	pypiStats = "packages.ecosyste.ms/api/v1/registries/pypi.org/packages/"
)

func registryCases() []packageCase {
	down := route{status: http.StatusInternalServerError, body: "{}"}
	return []packageCase{
		{"npm package missing", "npm:nopkg", map[string]route{},
			expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}, facts: map[string]string{"published": "false"}}},
		{"npm package present", "npm:express", map[string]route{npmStats + "express": statsRoute("4.0.0", daysAgo(4000), 5000000)},
			expect{status: refs.Checked, facts: map[string]string{"version": "4.0.0", "monthly_downloads": "5000000"}}},
		{"npm package new with few downloads", "npm:fresh", map[string]route{npmStats + "fresh": statsRoute("0.0.1", daysAgo(5), 12)},
			expect{status: refs.Checked, findings: []string{"PACKAGE-NEW", "PACKAGE-LOW-DOWNLOADS"}, facts: map[string]string{"monthly_downloads": "12"}}},
		{"scoped npm package", "npm:@types/node", map[string]route{npmStats + "@types/node": statsRoute("18.0.0", daysAgo(4000), 5000000)},
			expect{status: refs.Checked, facts: map[string]string{"version": "18.0.0"}}},
		{"npm package unknown to stats but in the registry", "npm:brandnew",
			map[string]route{"registry.npmjs.org/brandnew/latest": npmRoute("1.0.0")},
			expect{status: refs.Checked, facts: map[string]string{"version": "1.0.0"}}},
		{"pypi package missing", "pypi:nopkg", map[string]route{},
			expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
		{"pypi package present", "pypi:requests", map[string]route{pypiStats + "requests": statsRoute("2.31.0", time.Date(2011, 2, 14, 0, 0, 0, 0, time.UTC), 100000)},
			expect{status: refs.Checked, facts: map[string]string{"version": "2.31.0", "monthly_downloads": "100000", "first_published": "2011-02-14"}}},
		{"stats down, registry answers", "pypi:requests",
			map[string]route{pypiStats + "requests": down, "pypi.org/pypi/requests/json": pypiRoute("2.31.0", "2011-02-14T00:00:00Z")},
			expect{status: refs.Checked, facts: map[string]string{"version": "2.31.0", "first_published": "2011-02-14", "monthly_downloads": "unknown"}}},
		{"stats down, registry missing", "npm:nopkg", map[string]route{npmStats + "nopkg": down},
			expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
		{"registry error", "npm:err", map[string]route{"registry.npmjs.org/err/latest": {status: http.StatusInternalServerError, body: "{}"}},
			expect{status: refs.Failed, detail: "HTTP 500"}},
		{"unknown registry", "gem:rails", map[string]route{}, expect{status: refs.Failed, detail: "unknown registry"}},
	}
}

var typosquatCases = []packageCase{
	{"typosquat of a popular pypi package", "pypi:reqeusts", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING", "PACKAGE-TYPOSQUAT"}, facts: map[string]string{"typosquat_of": "requests"}}},
	{"short name within distance is not a typosquat", "pypi:task", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
	{"pypi underscore normalises to the popular name", "pypi:scikit_learn", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
	{"npm underscore stays a typosquat", "npm:lo_dash", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING", "PACKAGE-TYPOSQUAT"}, facts: map[string]string{"typosquat_of": "lodash"}}},
	{"npm popular name in another case", "npm:React", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
	{"name under three characters", "pypi:re", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
	{"two edits on a long name", "npm:jsonwebtokn", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING", "PACKAGE-TYPOSQUAT"}, facts: map[string]string{"typosquat_of": "jsonwebtoken"}}},
	{"three edits away", "pypi:rqsts", map[string]route{},
		expect{status: refs.Checked, findings: []string{"PACKAGE-MISSING"}}},
}

func TestPackageCollect(t *testing.T) {
	cases := append(registryCases(), typosquatCases...)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := newNet(t, tc.routes)
			rep := refcheck.NewPackage(n.Client, testConfig()).Collect(context.Background(), refs.Ref{Kind: refs.Package, Value: tc.value})
			verify(t, rep, tc.want)
		})
	}
}
