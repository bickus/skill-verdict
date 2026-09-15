// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

//go:embed data/hosts.json
var hostsJSON []byte

type hostData struct {
	Platforms  []string `json:"platforms"`
	Shorteners []string `json:"shorteners"`
	Pastes     []string `json:"pastes"`
	FileHosts  []string `json:"filehosts"`
}

var hosts = sync.OnceValue(func() hostData {
	var data hostData
	if err := json.Unmarshal(hostsJSON, &data); err != nil {
		panic(err)
	}
	return data
})

var chromeHeaders = http.Header{
	"sec-ch-ua":                 {`"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`},
	"sec-ch-ua-mobile":          {"?0"},
	"sec-ch-ua-platform":        {`"Windows"`},
	"Upgrade-Insecure-Requests": {"1"},
	"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"},
	"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/\x2a;q=0.8,application/signed-exchange;v=b3;q=0.7"},
	"Sec-Fetch-Site":            {"none"},
	"Sec-Fetch-Mode":            {"navigate"},
	"Sec-Fetch-User":            {"?1"},
	"Sec-Fetch-Dest":            {"document"},
	"Accept-Encoding":           {"gzip, deflate, br, zstd"},
	"Accept-Language":           {"en-US,en;q=0.9"},
	"Priority":                  {"u=0, i"},
}

func suffixOf(host string, suffixes []string) (string, bool) {
	for _, s := range suffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return s, true
		}
	}
	return "", false
}

func tld(host string) string {
	return host[strings.LastIndex(host, ".")+1:]
}

func hostFindings(host string, rep *refs.Report) {
	if _, ok := suffixOf(host, hosts().Shorteners); ok {
		addFinding(rep, "DOMAIN-SHORTENER")
	}
	if _, ok := suffixOf(host, hosts().Pastes); ok {
		addFinding(rep, "DOMAIN-PASTE")
	}
	if _, ok := suffixOf(host, hosts().FileHosts); ok {
		addFinding(rep, "DOMAIN-FILEHOST")
	}
}

type DomainCollector struct {
	fetch     fetcher
	transport http.RoundTripper
	cfg       config.ReferencesConfig
	rdap      *bootstrap
}

func NewDomain(client *http.Client, cfg config.ReferencesConfig) *DomainCollector {
	f := newFetcher(client, cfg)
	return &DomainCollector{fetch: f, transport: client.Transport, cfg: cfg, rdap: &bootstrap{fetch: f}}
}

func (*DomainCollector) Kind() refs.Kind {
	return refs.Domain
}

func (*DomainCollector) Extract(a *inventory.Artifact) []refs.Ref {
	return domainRefs(a)
}

type registration struct {
	registrable  string
	age          int
	unregistered bool
}

func (d *DomainCollector) Collect(ctx context.Context, ref refs.Ref) refs.Report {
	rep := refs.Report{Ref: ref, Status: refs.Checked}
	host := ref.Value
	reg := registration{registrable: host, age: -1}
	if suffix, ok := suffixOf(host, hosts().Platforms); ok {
		addFact(&rep, "platform", suffix)
	} else if err := d.register(ctx, host, &reg, &rep); err != nil {
		fail(&rep, err)
		return rep
	}
	hostFindings(host, &rep)
	if reg.unregistered {
		return rep
	}
	cross, err := d.follow(ctx, host, reg.registrable, &rep)
	if err != nil {
		fail(&rep, err)
		return rep
	}
	if cross && reg.age >= 0 && reg.age < d.cfg.YoungDomainDays {
		addFinding(&rep, "DOMAIN-NEW-REDIRECT")
	}
	return rep
}

func (d *DomainCollector) register(ctx context.Context, host string, reg *registration, rep *refs.Report) error {
	server, err := d.rdap.server(ctx, tld(host))
	if err != nil {
		return err
	}
	if server == "" {
		return nil
	}
	dom, err := rdapLookup(ctx, d.fetch, server, host)
	if err != nil {
		return err
	}
	if dom == nil {
		reg.unregistered = true
		addFact(rep, "registered", "false")
		addFinding(rep, "DOMAIN-UNREGISTERED")
		return nil
	}
	reg.registrable, reg.age = d.domainFacts(dom, host, rep)
	return nil
}

func (d *DomainCollector) follow(ctx context.Context, host, registrable string, rep *refs.Report) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/", nil)
	if err != nil {
		return false, err
	}
	req.Header = chromeHeaders.Clone()
	hops := 0
	var state *tls.ConnectionState
	client := &http.Client{Transport: d.transport, CheckRedirect: func(next *http.Request, _ []*http.Request) error {
		if state == nil {
			state = next.Response.TLS
		}
		if hops >= d.cfg.MaxRedirects {
			return http.ErrUseLastResponse
		}
		hops++
		return nil
	}}
	resp, err := client.Do(req)
	if unreachable(err) {
		addFact(rep, "connection_error", err.Error())
		addFinding(rep, "DOMAIN-UNREACHABLE")
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err = resp.Body.Close(); err != nil {
		return false, err
	}
	if state == nil {
		state = resp.TLS
	}
	certFacts(state, rep)
	if resp.StatusCode >= http.StatusBadRequest {
		addFact(rep, "http_status", strconv.Itoa(resp.StatusCode))
		addFinding(rep, "DOMAIN-HTTP-ERROR")
	}
	final := strings.ToLower(resp.Request.URL.Hostname())
	if final == registrable || strings.HasSuffix(final, "."+registrable) {
		return false, nil
	}
	addFact(rep, "final_url", resp.Request.URL.String())
	addFact(rep, "redirect_cross_domain", final)
	if publicDomain(final) {
		rep.Derived = append(rep.Derived, refs.Ref{Kind: refs.Domain, Value: final})
	}
	return true, nil
}

var dropped = []error{syscall.ECONNREFUSED, syscall.ECONNRESET, io.EOF, io.ErrUnexpectedEOF}

func unreachable(err error) bool {
	var dns *net.DNSError
	var op *net.OpError
	var cert *tls.CertificateVerificationError
	switch {
	case errors.As(err, &dns):
		return dns.IsNotFound
	case errors.As(err, &op) && op.Op == "remote error", errors.As(err, &cert):
		return true
	}
	return slices.ContainsFunc(dropped, func(target error) bool { return errors.Is(err, target) })
}
