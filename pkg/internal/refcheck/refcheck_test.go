// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/layers/walk"
	"github.com/bickus/skill-verdict/pkg/internal/pipeline"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
	"github.com/bickus/skill-verdict/pkg/rules"
)

const (
	bootstrapKey  = "data.iana.org/rdap/dns.json"
	bootstrapBody = `{"services":[[["com","xyz","uk","ly","org"],["https://rdap.test/"]]]}`
)

type route struct {
	status   int
	body     string
	location string
	headers  map[string]string
	limit    int
	hold     bool
}

var ok = route{status: http.StatusOK, body: "{}"}

func redirect(to string) route {
	return route{status: http.StatusFound, location: to}
}

type fakeNet struct {
	t      *testing.T
	mu     sync.Mutex
	routes map[string]route
	hits   map[string]int
	auth   map[string]string
	raw    *http.Client
	Client *http.Client
}

func newNet(t *testing.T, routes map[string]route) *fakeNet {
	t.Helper()
	n := &fakeNet{t: t, routes: routes, hits: make(map[string]int), auth: make(map[string]string)}
	srv := httptest.NewServer(n)
	t.Cleanup(srv.Close)
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	safe := &http.Transport{DialContext: refcheck.SafeDialer().DialContext}
	n.raw = &http.Client{Transport: &rewrite{target: target, local: srv.Client().Transport, safe: safe}}
	n.Client = refcheck.Paced(n.raw, testConfig())
	return n
}

func (n *fakeNet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Host + r.URL.Path
	n.mu.Lock()
	n.hits[key]++
	hit := n.hits[key]
	n.auth[key] = r.Header.Get("Authorization")
	rt, found := n.routes[key]
	n.mu.Unlock()
	if !found {
		http.Error(w, "{}", http.StatusNotFound)
		return
	}
	if hit <= rt.limit {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	if rt.hold {
		<-r.Context().Done()
		return
	}
	if rt.location != "" {
		w.Header().Set("Location", rt.location)
	}
	for k, v := range rt.headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(rt.status)
	if _, err := io.WriteString(w, rt.body); err != nil {
		n.t.Error(err)
	}
}

func (n *fakeNet) count(key string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.hits[key]
}

func (n *fakeNet) authorization(key string) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.auth[key]
}

type rewrite struct {
	target *url.URL
	local  http.RoundTripper
	safe   http.RoundTripper
}

func (rt *rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	if net.ParseIP(req.URL.Hostname()) != nil {
		return rt.safe.RoundTrip(req)
	}
	r := req.Clone(req.Context())
	r.Host = req.URL.Host
	r.URL.Scheme = rt.target.Scheme
	r.URL.Host = rt.target.Host
	resp, err := rt.local.RoundTrip(r)
	if resp != nil {
		resp.Request = req
	}
	return resp, err
}

func testConfig() config.ReferencesConfig {
	cfg := config.Default().Layers.References
	cfg.Timeout = config.Duration(2 * time.Second)
	cfg.Budget = config.Duration(5 * time.Second)
	cfg.MaxRedirects = 3
	cfg.Concurrency = 2
	cfg.RequestDelay = 0
	cfg.RetryDelays = nil
	return cfg
}

func daysAgo(n int) time.Time {
	return time.Now().AddDate(0, 0, -n)
}

func daysAhead(n int) time.Time {
	return time.Now().AddDate(0, 0, n)
}

func rdapRoute(name string, registered, expires time.Time, registrant string) route {
	body := fmt.Sprintf(`{"ldhName":%q,"status":["active"],"events":[{"eventAction":"registration","eventDate":%q},{"eventAction":"expiration","eventDate":%q}],"entities":[{"roles":["registrar"],"vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","Registrar Inc"]]]},{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text",%q]]]}]}`,
		strings.ToUpper(name), registered.Format(time.RFC3339), expires.Format(time.RFC3339), registrant)
	return route{status: http.StatusOK, body: body}
}

func rdapOld(name string) route {
	return rdapRoute(name, daysAgo(800), daysAhead(300), "Jane Doe")
}

func userRoute(login string, created time.Time) route {
	body := fmt.Sprintf(`{"login":%q,"type":"User","created_at":%q,"public_repos":3}`, login, created.Format(time.RFC3339))
	return route{status: http.StatusOK, body: body}
}

func repoRoute(ownerLogin string, stars int, fork, archived bool) route {
	body := fmt.Sprintf(`{"owner":{"login":%q},"stargazers_count":%d,"fork":%t,"archived":%t,"created_at":"2020-01-02T00:00:00Z","pushed_at":"2024-03-04T00:00:00Z"}`, ownerLogin, stars, fork, archived)
	return route{status: http.StatusOK, body: body}
}

func statsRoute(version string, first time.Time, downloads int) route {
	body := fmt.Sprintf(`{"latest_release_number":%q,"first_release_published_at":%q,"downloads":%d,"downloads_period":"last-month"}`,
		version, first.Format(time.RFC3339), downloads)
	return route{status: http.StatusOK, body: body}
}

func npmRoute(version string) route {
	return route{status: http.StatusOK, body: fmt.Sprintf(`{"version":%q}`, version)}
}

func pypiRoute(version, upload string) route {
	body := fmt.Sprintf(`{"info":{"version":%q},"releases":{"1.0":[{"upload_time_iso_8601":%q}]}}`, version, upload)
	return route{status: http.StatusOK, body: body}
}

type expect struct {
	status   refs.Status
	findings []string
	facts    map[string]string
	detail   string
}

func factValue(rep refs.Report, key string) (string, bool) {
	for _, f := range rep.Facts {
		if f.Key == key {
			return f.Value, true
		}
	}
	return "", false
}

func verify(t *testing.T, rep refs.Report, want expect) {
	t.Helper()
	if rep.Status != want.status {
		t.Errorf("status %s (%s), want %s", rep.Status, rep.Detail, want.status)
	}
	if !reflect.DeepEqual(rep.Findings, want.findings) {
		t.Errorf("findings %v, want %v", rep.Findings, want.findings)
	}
	for key, value := range want.facts {
		if got, found := factValue(rep, key); !found || got != value {
			t.Errorf("fact %s = %q (%v), want %q", key, got, found, value)
		}
	}
	if want.detail != "" && !strings.Contains(rep.Detail, want.detail) {
		t.Errorf("detail %q lacks %q", rep.Detail, want.detail)
	}
}

func walked(t *testing.T, dir string) *inventory.Skill {
	t.Helper()
	root, err := walk.Resolve(filepath.Join("testdata", dir))
	if err != nil {
		t.Fatal(err)
	}
	layers := config.Default().Layers
	set, err := rules.Load("", layers.Rules())
	if err != nil {
		t.Fatal(err)
	}
	s := &pipeline.State{Skill: &inventory.Skill{Root: root}, Rules: set}
	pipeline.Run(context.Background(), s, []pipeline.Layer{walk.Layer{Config: layers.Walk}})
	if s.Layers[0].Status != pipeline.Done {
		t.Fatalf("walk %+v", s.Layers[0])
	}
	return s.Skill
}

func artifact(path, text string) *inventory.Artifact {
	return &inventory.Artifact{Path: path, Kind: inventory.Content, ContentType: inventory.Markdown, Size: int64(len(text)), Content: text, Origin: inventory.Original}
}
