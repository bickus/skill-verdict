// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package references_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/references"
	"github.com/bickus/skill-verdict/pkg/internal/layers/static"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/rules"
)

type rewrite struct {
	target *url.URL
	base   http.RoundTripper
}

func (rt rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.URL.Scheme = rt.target.Scheme
	r.URL.Host = rt.target.Host
	return rt.base.RoundTrip(r)
}

func fake(t *testing.T, bodies map[string]string) *http.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, found := bodies[r.URL.Path]
		if !found {
			http.Error(w, "{}", http.StatusNotFound)
			return
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Transport: rewrite{target: target, base: srv.Client().Transport}}
}

func fakeGitHub(t *testing.T) *http.Client {
	return fake(t, map[string]string{"/users/acme": `{"login":"acme","type":"User"}`})
}

func newState(t *testing.T, dir string, staticRules, referenceRules map[string]config.RuleSetting) *pipeline.State {
	t.Helper()
	root, err := walk.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	layers := config.Default().Layers
	layers.Static.Rules = staticRules
	layers.References.Rules = referenceRules
	set, err := rules.Load("", layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	return s
}

func TestRunTurnsFactsIntoFindings(t *testing.T) {
	off := false
	cases := []struct {
		name     string
		settings map[string]config.RuleSetting
		want     []finding.Finding
	}{
		{"enabled fact rule becomes a finding", nil, []finding.Finding{{Rule: "GITHUB-REPO-MISSING", Severity: finding.Medium,
			Location: finding.Location{File: "SKILL.md", Line: 3}, Evidence: "acme/gone", Reference: "acme/gone"}}},
		{"disabled fact rule is skipped", map[string]config.RuleSetting{"GITHUB-REPO-MISSING": {Enabled: &off}}, nil},
	}
	cfg := config.Default().Layers.References
	cfg.Collectors = map[string]bool{"github": true}
	cfg.Timeout = config.Duration(2 * time.Second)
	cfg.Budget = config.Duration(5 * time.Second)
	cfg.RequestDelay = 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newState(t, "testdata/skill", nil, tc.settings)
			layer := references.New(fakeGitHub(t), cfg, "")
			pipeline.Run(context.Background(), s, []pipeline.Layer{layer})
			if !reflect.DeepEqual(s.Findings, tc.want) {
				t.Errorf("findings %+v, want %+v", s.Findings, tc.want)
			}
			if len(s.Refs) != 1 || s.Refs[0].ID != "R1" || s.Refs[0].Ref.Value != "acme/gone" || s.Refs[0].Status != "checked" {
				t.Errorf("refs %+v, want one checked for acme/gone numbered R1", s.Refs)
			}
			if want := []pipeline.LayerStatus{{Layer: "walk", Status: "done"}, {Layer: "references", Status: "done"}}; !reflect.DeepEqual(s.Layers, want) {
				t.Errorf("layers %+v, want %+v", s.Layers, want)
			}
		})
	}
}

func TestRunWritesTheBriefArtifact(t *testing.T) {
	s := newState(t, "testdata/skill", nil, nil)
	cfg := config.Default().Layers.References
	cfg.Collectors = map[string]bool{"github": true}
	cfg.RequestDelay = 0
	pipeline.Run(context.Background(), s, []pipeline.Layer{references.New(fakeGitHub(t), cfg, "")})
	a := s.Skill.Artifact("references.brief.md")
	if a == nil {
		t.Fatalf("artifacts %+v, want references.brief.md", s.Skill.Artifacts)
	}
	if a.Kind != inventory.Brief || a.Origin != "references" || a.ContentType != inventory.Markdown || a.Size != int64(len(a.Content)) {
		t.Errorf("artifact %+v, want kind brief, origin references, markdown", a)
	}
	for _, part := range []string{"### R1 acme/gone\n", "- status: checked\n", "- R1.1 used at SKILL.md:3 `Uses https://github.com/acme/gone for things.`\n", "- repo: missing\n"} {
		if !strings.Contains(a.Content, part) {
			t.Errorf("brief lacks %q:\n%s", part, a.Content)
		}
	}
}

func TestRunStopsOverMaxReferences(t *testing.T) {
	s := newState(t, "testdata/many", nil, nil)
	cfg := config.Default().Layers.References
	cfg.Collectors = map[string]bool{"github": true}
	cfg.RequestDelay = 0
	cfg.MaxReferences = 1
	pipeline.Run(context.Background(), s, []pipeline.Layer{references.New(fakeGitHub(t), cfg, "")})
	want := []finding.Finding{{Rule: "REFERENCES-TOO-MANY", Severity: finding.High, Location: finding.Location{File: "SKILL.md", Line: 4}, Evidence: "2 references, limit 1"}}
	if !reflect.DeepEqual(s.Findings, want) {
		t.Errorf("findings %+v, want %+v", s.Findings, want)
	}
	if len(s.Refs) != 0 {
		t.Errorf("refs %+v, want none", s.Refs)
	}
	layers := []pipeline.LayerStatus{{Layer: "walk", Status: "done"}, {Layer: "references", Status: "interrupted", Interrupt: "REFERENCES-TOO-MANY"}}
	if !reflect.DeepEqual(s.Layers, layers) {
		t.Errorf("layers %+v, want %+v", s.Layers, layers)
	}
}

func scanUnpinned(t *testing.T, staticRules map[string]config.RuleSetting) *pipeline.State {
	t.Helper()
	s := newState(t, "testdata/unpinned", staticRules, nil)
	cfg := config.Default().Layers.References
	cfg.Collectors = map[string]bool{"github": true, "package": true}
	cfg.RequestDelay = 0
	client := fake(t, map[string]string{
		"/api/v1/registries/npmjs.org/packages/rarepkg": `{"downloads":10,"downloads_period":"last-month","first_release_published_at":"2015-01-01T00:00:00Z"}`,
		"/api/v1/registries/npmjs.org/packages/bigpkg":  `{"downloads":5000000,"downloads_period":"last-month","first_release_published_at":"2015-01-01T00:00:00Z"}`,
		"/users/acme":      `{"login":"acme","type":"User","created_at":"2015-01-01T00:00:00Z"}`,
		"/repos/acme/tool": `{"owner":{"login":"acme"},"stargazers_count":5,"forks_count":0}`,
	})
	pipeline.Run(context.Background(), s, []pipeline.Layer{static.Layer{}, references.New(client, cfg, "")})
	return s
}

func TestUnpopularFindingsStandAtUnpinnedPlaces(t *testing.T) {
	s := scanUnpinned(t, nil)
	cases := []struct {
		name string
		rule string
		line int
		want []string
	}{
		{"unpopular package installed without a version", "PACKAGE-UNPINNED-UNPOPULAR", 5, []string{"npm:rarepkg"}},
		{"popular package installed without a version", "PACKAGE-UNPINNED-UNPOPULAR", 6, nil},
		{"unpopular package with an exact version", "PACKAGE-UNPINNED-UNPOPULAR", 7, nil},
		{"unpopular repository cloned without a commit", "REPO-UNPINNED-UNPOPULAR", 8, []string{"acme/tool"}},
		{"repository on a host without counts", "REPO-UNPINNED-UNPOPULAR", 9, []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := referencesAt(s.Findings, tc.rule, tc.line); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s on line %d names %q, want %q", tc.rule, tc.line, got, tc.want)
			}
		})
	}
}

func TestUnpopularFindingsStayWithTheToolRuleOff(t *testing.T) {
	off := false
	s := scanUnpinned(t, map[string]config.RuleSetting{"UNPINNED-NPM": {Enabled: &off}, "UNPINNED-GIT": {Enabled: &off}})
	cases := []struct {
		name string
		rule string
		line int
		want []string
	}{
		{"npm rule that is off records no finding", "UNPINNED-NPM", 5, nil},
		{"unpopular package keeps its finding", "PACKAGE-UNPINNED-UNPOPULAR", 5, []string{"npm:rarepkg"}},
		{"git rule that is off records no finding", "UNPINNED-GIT", 8, nil},
		{"unpopular repository keeps its finding", "REPO-UNPINNED-UNPOPULAR", 8, []string{"acme/tool"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := referencesAt(s.Findings, tc.rule, tc.line); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s on line %d names %q, want %q", tc.rule, tc.line, got, tc.want)
			}
		})
	}
}

func referencesAt(found []finding.Finding, rule string, line int) []string {
	var out []string
	for _, f := range found {
		if f.Rule == rule && f.Location.Line == line {
			out = append(out, f.Reference)
		}
	}
	return out
}
