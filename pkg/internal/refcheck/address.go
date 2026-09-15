// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"sync"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

var bareAddressPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3}):\d{1,5}\b`),
	regexp.MustCompile(`\[([0-9A-Fa-f:.]+)\]:\d{1,5}\b`),
	regexp.MustCompile(`(?i)\b(?:nc|ncat|netcat|socat|ssh|scp|sftp|telnet|curl|wget|ftp|tftp|rsync)\b[^|;&]*?\b(\d{1,3}(?:\.\d{1,3}){3})\b`),
}

//go:embed data/reserved-ranges.json
var reservedRangesJSON []byte

var reservedRanges = sync.OnceValue(func() []netip.Prefix {
	var ranges []netip.Prefix
	if err := json.Unmarshal(reservedRangesJSON, &ranges); err != nil {
		panic(err)
	}
	return ranges
})

type AddressCollector struct{}

func NewAddress() AddressCollector {
	return AddressCollector{}
}

func (AddressCollector) Kind() refs.Kind {
	return refs.Address
}

func (AddressCollector) Extract(a *inventory.Artifact) []refs.Ref {
	var out []refs.Ref
	eachLine(a, func(line int, text string) {
		seen := make(map[string]bool)
		for _, u := range urls(text) {
			if host := address(u.Hostname()); host != "" && !seen[host] {
				seen[host] = true
				out = append(out, newURLRef(refs.Address, host, u.String(), a.Path, line))
			}
		}
		for _, host := range bareAddresses(text) {
			if !seen[host] {
				seen[host] = true
				out = append(out, newRef(refs.Address, host, a.Path, line))
			}
		}
	})
	return out
}

func bareAddresses(text string) []string {
	var hosts []string
	for _, re := range bareAddressPatterns {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if host := address(m[1]); host != "" {
				hosts = append(hosts, host)
			}
		}
	}
	return hosts
}

func address(host string) string {
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	return addr.WithZone("").String()
}

func (AddressCollector) Collect(_ context.Context, ref refs.Ref) refs.Report {
	rep := refs.Report{Ref: ref, Status: refs.Checked}
	addr, err := netip.ParseAddr(ref.Value)
	if err != nil {
		fail(&rep, err)
		return rep
	}
	isPublic := public(addr)
	addFact(&rep, "public", strconv.FormatBool(isPublic))
	if isPublic {
		addFinding(&rep, "IP-ADDRESS-PUBLIC")
	}
	return rep
}

func public(addr netip.Addr) bool {
	addr = addr.Unmap().WithZone("")
	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return false
	}
	return !slices.ContainsFunc(reservedRanges(), func(p netip.Prefix) bool { return p.Contains(addr) })
}
