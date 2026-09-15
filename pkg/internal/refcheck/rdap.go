// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bickus/skill-verdict/pkg/refs"
)

const bootstrapURL = "https://data.iana.org/rdap/dns.json"

var privacyMarkers = []string{"privacy", "proxy", "redacted", "withheld", "whoisguard"}

type bootstrap struct {
	fetch   fetcher
	mu      sync.Mutex
	servers map[string]string
}

func (b *bootstrap) server(ctx context.Context, tld string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.servers == nil {
		servers, err := b.load(ctx)
		if err != nil {
			return "", err
		}
		b.servers = servers
	}
	return b.servers[tld], nil
}

func (b *bootstrap) load(ctx context.Context) (map[string]string, error) {
	var file struct {
		Services [][][]string `json:"services"`
	}
	switch found, err := b.fetch.getJSON(ctx, bootstrapURL, nil, &file); {
	case err != nil:
		return nil, fmt.Errorf("rdap bootstrap: %w", err)
	case !found:
		return nil, errors.New("rdap bootstrap: not found")
	}
	servers := make(map[string]string)
	for _, svc := range file.Services {
		if len(svc) < 2 || len(svc[1]) == 0 {
			continue
		}
		server := svc[1][0]
		if !strings.HasSuffix(server, "/") {
			server += "/"
		}
		for _, t := range svc[0] {
			servers[strings.ToLower(t)] = server
		}
	}
	return servers, nil
}

type rdapEvent struct {
	Action string `json:"eventAction"`
	Date   string `json:"eventDate"`
}

type rdapEntity struct {
	Roles []string        `json:"roles"`
	VCard json.RawMessage `json:"vcardArray"`
}

type rdapDomain struct {
	LDHName  string       `json:"ldhName"`
	Events   []rdapEvent  `json:"events"`
	Entities []rdapEntity `json:"entities"`
}

func rdapLookup(ctx context.Context, f fetcher, server, host string) (*rdapDomain, error) {
	labels := strings.Split(host, ".")
	for n := 2; n <= min(len(labels), 5); n++ {
		candidate := strings.Join(labels[len(labels)-n:], ".")
		var dom rdapDomain
		found, err := f.getJSON(ctx, server+"domain/"+candidate, nil, &dom)
		if err != nil {
			return nil, err
		}
		if found {
			return &dom, nil
		}
	}
	return nil, nil
}

func (r *rdapDomain) event(action string) (time.Time, bool) {
	for _, e := range r.Events {
		if e.Action != action {
			continue
		}
		if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (r *rdapDomain) entityName(role string) string {
	for _, e := range r.Entities {
		if slices.Contains(e.Roles, role) {
			if name := vcardName(e.VCard); name != "" {
				return name
			}
		}
	}
	return ""
}

func vcardName(raw json.RawMessage) string {
	var card []json.RawMessage
	if json.Unmarshal(raw, &card) != nil || len(card) < 2 {
		return ""
	}
	var props [][]json.RawMessage
	if json.Unmarshal(card[1], &props) != nil {
		return ""
	}
	for _, p := range props {
		var key, value string
		if len(p) == 4 && json.Unmarshal(p[0], &key) == nil && key == "fn" && json.Unmarshal(p[3], &value) == nil {
			return value
		}
	}
	return ""
}

func (r *rdapDomain) private() bool {
	name := strings.ToLower(r.entityName("registrant"))
	for _, m := range privacyMarkers {
		if strings.Contains(name, m) {
			return true
		}
	}
	return false
}

func (d *DomainCollector) domainFacts(dom *rdapDomain, host string, rep *refs.Report) (string, int) {
	registrable := strings.ToLower(dom.LDHName)
	if registrable == "" {
		registrable = host
	}
	addFact(rep, "registrable", registrable)
	age := -1
	if t, ok := dom.event("registration"); ok {
		age = ageDays(t)
		addFact(rep, "registered", t.Format(time.DateOnly))
		addFact(rep, "age_days", strconv.Itoa(age))
		if age < d.cfg.YoungDomainDays {
			addFinding(rep, "DOMAIN-NEW")
		}
	}
	if t, ok := dom.event("expiration"); ok {
		addFact(rep, "expires", t.Format(time.DateOnly))
		if time.Until(t) < time.Duration(d.cfg.ExpiryDays)*24*time.Hour {
			addFinding(rep, "DOMAIN-EXPIRING")
		}
	}
	if name := dom.entityName("registrar"); name != "" {
		addFact(rep, "registrar", name)
	}
	if dom.private() {
		addFact(rep, "privacy", "true")
	}
	return registrable, age
}
