// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type domainCase struct {
	name   string
	host   string
	routes map[string]route
	noBoot bool
	want   expect
	noHits []string
}

func youngDomainCases() []domainCase {
	young := rdapRoute("young.com", daysAgo(30), daysAhead(300), "Jane Doe")
	return []domainCase{
		{"young domain redirecting elsewhere", "young.com",
			map[string]route{"rdap.test/domain/young.com": young, "young.com/": redirect("https://elsewhere.org/"), "elsewhere.org/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-NEW", "DOMAIN-NEW-REDIRECT"},
				facts: map[string]string{"registrable": "young.com", "redirect_cross_domain": "elsewhere.org", "final_url": "https://elsewhere.org/"}}, nil},
		{"young domain redirecting within its site", "young.com",
			map[string]route{"rdap.test/domain/young.com": young, "young.com/": redirect("https://www.young.com/"), "www.young.com/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-NEW"}}, nil},
		{"old domain redirecting elsewhere", "old.com",
			map[string]route{"rdap.test/domain/old.com": rdapOld("old.com"), "old.com/": redirect("https://elsewhere.org/"), "elsewhere.org/": ok}, false,
			expect{status: refs.Checked, facts: map[string]string{"redirect_cross_domain": "elsewhere.org"}}, nil},
		{"young domain without redirect", "young.com",
			map[string]route{"rdap.test/domain/young.com": young, "young.com/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-NEW"}}, nil},
	}
}

func registrationCases() []domainCase {
	return []domainCase{
		{"domain expiring within days", "soon.com",
			map[string]route{"rdap.test/domain/soon.com": rdapRoute("soon.com", daysAgo(800), daysAhead(3), "Jane Doe"), "soon.com/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-EXPIRING"}}, nil},
		{"domain expiring later", "later.com",
			map[string]route{"rdap.test/domain/later.com": rdapRoute("later.com", daysAgo(800), daysAhead(60), "Jane Doe"), "later.com/": ok}, false,
			expect{status: refs.Checked}, nil},
		{"unregistered domain", "nothere.com", map[string]route{}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-UNREGISTERED"}, facts: map[string]string{"registered": "false"}},
			[]string{"nothere.com/"}},
		{"registered under a longer suffix", "app.site.co.uk",
			map[string]route{"rdap.test/domain/site.co.uk": rdapOld("site.co.uk"), "app.site.co.uk/": ok}, false,
			expect{status: refs.Checked, facts: map[string]string{"registrable": "site.co.uk"}}, nil},
		{"tld without rdap server", "foo.zz", map[string]route{"foo.zz/": ok}, false,
			expect{status: refs.Checked}, nil},
		{"bootstrap failure", "young.com", map[string]route{"young.com/": ok}, true,
			expect{status: refs.Failed, detail: "rdap bootstrap"}, []string{"young.com/"}},
		{"rdap server error", "err.com", map[string]route{"rdap.test/domain/err.com": {status: http.StatusInternalServerError, body: "{}"}}, false,
			expect{status: refs.Failed, detail: "HTTP 500"}, nil},
		{"rdap malformed reply", "bad.com", map[string]route{"rdap.test/domain/bad.com": {status: http.StatusOK, body: "not json"}}, false,
			expect{status: refs.Failed, detail: "invalid character"}, nil},
		{"privacy registrant", "cheap.xyz",
			map[string]route{"rdap.test/domain/cheap.xyz": rdapRoute("cheap.xyz", daysAgo(800), daysAhead(300), "Privacy Protect LLC"), "cheap.xyz/": ok}, false,
			expect{status: refs.Checked, facts: map[string]string{"privacy": "true", "registrar": "Registrar Inc"}}, nil},
	}
}

func redirectCases() []domainCase {
	return []domainCase{
		{"platform host skips rdap", "thing.vercel.app", map[string]route{"thing.vercel.app/": ok}, false,
			expect{status: refs.Checked, facts: map[string]string{"platform": "vercel.app"}},
			[]string{"rdap.test/domain/vercel.app", "rdap.test/domain/thing.vercel.app"}},
		{"shortener redirect", "bit.ly",
			map[string]route{"rdap.test/domain/bit.ly": rdapOld("bit.ly"), "bit.ly/": redirect("https://bitly.com/"), "bitly.com/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-SHORTENER"}, facts: map[string]string{"redirect_cross_domain": "bitly.com"}}, nil},
		{"paste site", "glot.io",
			map[string]route{"rdap.test/domain/glot.io": rdapOld("glot.io"), "glot.io/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-PASTE"}}, nil},
		{"file sharing site", "dl.gofile.io",
			map[string]route{"rdap.test/domain/gofile.io": rdapOld("gofile.io"), "dl.gofile.io/": ok}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-FILEHOST"}}, nil},
		{"ordinary host is neither", "example.com",
			map[string]route{"rdap.test/domain/example.com": rdapOld("example.com"), "example.com/": ok}, false,
			expect{status: refs.Checked}, nil},
		{"redirect chain capped", "loop.com",
			map[string]route{"rdap.test/domain/loop.com": rdapOld("loop.com"), "loop.com/": redirect("https://loop.com/a"), "loop.com/a": redirect("https://loop.com/b"),
				"loop.com/b": redirect("https://loop.com/c"), "loop.com/c": redirect("https://loop.com/d"), "loop.com/d": ok}, false,
			expect{status: refs.Checked},
			[]string{"loop.com/d"}},
		{"redirect to loopback refused", "trap.com",
			map[string]route{"rdap.test/domain/trap.com": rdapOld("trap.com"), "trap.com/": redirect("http://127.0.0.1:9/")}, false,
			expect{status: refs.Failed, detail: "not a public address"}, nil},
	}
}

func TestDomainCollect(t *testing.T) {
	var cases []domainCase
	cases = append(cases, youngDomainCases()...)
	cases = append(cases, registrationCases()...)
	cases = append(cases, redirectCases()...)
	cases = append(cases, statusCases()...)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.noBoot {
				tc.routes[bootstrapKey] = route{status: http.StatusOK, body: bootstrapBody}
			}
			n := newNet(t, tc.routes)
			rep := refcheck.NewDomain(n.Client, testConfig()).Collect(context.Background(), refs.Ref{Kind: refs.Domain, Value: tc.host})
			verify(t, rep, tc.want)
			for _, key := range tc.noHits {
				if n.count(key) != 0 {
					t.Errorf("%s was requested", key)
				}
			}
		})
	}
}

func TestDomainBootstrapRetriesAfterFailure(t *testing.T) {
	routes := map[string]route{bootstrapKey: {status: http.StatusInternalServerError, body: "{}"}, "rdap.test/domain/old.com": rdapOld("old.com"), "old.com/": ok}
	n := newNet(t, routes)
	col := refcheck.NewDomain(n.Client, testConfig())
	ref := refs.Ref{Kind: refs.Domain, Value: "old.com"}
	verify(t, col.Collect(context.Background(), ref), expect{status: refs.Failed, detail: "rdap bootstrap"})
	routes[bootstrapKey] = route{status: http.StatusOK, body: bootstrapBody}
	verify(t, col.Collect(context.Background(), ref), expect{status: refs.Checked, facts: map[string]string{"registrable": "old.com"}})
	if n.count(bootstrapKey) != 2 {
		t.Errorf("bootstrap fetched %d times, want once per attempt", n.count(bootstrapKey))
	}
}

func statusCases() []domainCase {
	return []domainCase{
		{"root answers not found", "gone.com",
			map[string]route{"rdap.test/domain/gone.com": rdapOld("gone.com"), "gone.com/": {status: http.StatusNotFound}}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-HTTP-ERROR"}, facts: map[string]string{"http_status": "404"}}, nil},
		{"root answers server error", "broken.com",
			map[string]route{"rdap.test/domain/broken.com": rdapOld("broken.com"), "broken.com/": {status: http.StatusInternalServerError}}, false,
			expect{status: refs.Checked, findings: []string{"DOMAIN-HTTP-ERROR"}, facts: map[string]string{"http_status": "500"}}, nil},
	}
}

func tlsClient(t *testing.T, handler http.Handler, cfg *tls.Config) *http.Client {
	t.Helper()
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = cfg
	srv.StartTLS()
	t.Cleanup(srv.Close)
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	addr := srv.Listener.Addr().String()
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	tr := &http.Transport{DialContext: dial, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: "example.com"}}
	return refcheck.Paced(&http.Client{Transport: tr}, testConfig())
}

func TestDomainUnreachable(t *testing.T) {
	clientCertOnly := &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAnyClientCert}
	drop := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := http.NewResponseController(w).Hijack()
		if err == nil {
			err = conn.Close()
		}
		if err != nil {
			t.Error(err)
		}
	})
	silent := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	cases := []struct {
		name    string
		handler http.Handler
		tls     *tls.Config
		want    expect
	}{
		{"tls handshake refused", http.NotFoundHandler(), clientCertOnly,
			expect{status: refs.Checked, findings: []string{"DOMAIN-UNREACHABLE"},
				facts: map[string]string{"connection_error": `Get "https://thing.vercel.app/": remote error: tls: certificate required`}}},
		{"connection closed without answer", drop, nil,
			expect{status: refs.Checked, findings: []string{"DOMAIN-UNREACHABLE"}}},
		{"no answer before timeout", silent, nil,
			expect{status: refs.Failed, detail: "deadline exceeded"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := tlsClient(t, tc.handler, tc.tls)
			rep := refcheck.NewDomain(client, testConfig()).Collect(context.Background(), refs.Ref{Kind: refs.Domain, Value: "thing.vercel.app"})
			verify(t, rep, tc.want)
		})
	}
}
