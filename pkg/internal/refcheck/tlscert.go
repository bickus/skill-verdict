// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"crypto/tls"
	"crypto/x509"
	"strings"

	"github.com/bickus/skill-verdict/pkg/refs"
)

var validationPolicies = map[string]string{
	"2.23.140.1.1":   "EV",
	"2.23.140.1.2.1": "DV",
	"2.23.140.1.2.2": "OV",
	"2.23.140.1.2.3": "IV",
}

func certFacts(state *tls.ConnectionState, rep *refs.Report) {
	if state == nil || len(state.PeerCertificates) == 0 {
		return
	}
	leaf := state.PeerCertificates[0]
	level := validationLevel(leaf)
	if level == "" {
		return
	}
	addFact(rep, "tls_validation", level)
	if level != "DV" && len(leaf.Subject.Organization) > 0 {
		addFact(rep, "tls_organization", strings.Join(leaf.Subject.Organization, ", "))
	}
}

func validationLevel(leaf *x509.Certificate) string {
	for _, oid := range leaf.Policies {
		if level, ok := validationPolicies[oid.String()]; ok {
			return level
		}
	}
	return ""
}
