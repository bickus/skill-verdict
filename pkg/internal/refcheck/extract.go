// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/internal/textutil"
	"github.com/bickus/skill-verdict/pkg/refs"
)

var (
	urlPattern     = regexp.MustCompile(`https?://(?:\[[^\s\]]+\][^\s<>"'` + "`" + `()\[\]{}]*|[^\s<>"'` + "`" + `()\[\]{}]+)`)
	gitSSHPattern  = regexp.MustCompile(`git@github\.com:([A-Za-z0-9][A-Za-z0-9-]{0,38})/([A-Za-z0-9._-]+)`)
	ghClonePattern = regexp.MustCompile(`\bgh\s+repo\s+clone\s+([A-Za-z0-9][A-Za-z0-9-]{0,38})/([A-Za-z0-9._-]+)`)
	ownerPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	repoPattern    = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	domainPattern  = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
)

var githubSections = map[string]bool{
	"orgs": true, "topics": true, "search": true, "features": true, "settings": true,
	"marketplace": true, "sponsors": true, "login": true, "apps": true, "about": true,
	"pricing": true, "explore": true, "site": true, "security": true, "enterprise": true,
	"events": true, "trending": true, "new": true, "notifications": true,
}

func eachLine(a *inventory.Artifact, fn func(line int, text string)) {
	for i, text := range textutil.Lines(a.Content) {
		if lead := strings.TrimLeft(text, " \t"); lead == "" || lead[0] == '#' {
			continue
		}
		fn(i+1, text)
	}
}

func newRef(kind refs.Kind, value, file string, line int) refs.Ref {
	return refs.Ref{Kind: kind, Value: value, Locations: []finding.Location{{File: file, Line: line}}}
}

func newURLRef(kind refs.Kind, value, full, file string, line int) refs.Ref {
	r := newRef(kind, value, file, line)
	if full != "" {
		r.URLs = []string{full}
	}
	return r
}

func urls(text string) []*url.URL {
	var out []*url.URL
	for _, m := range urlPattern.FindAllString(text, -1) {
		u, err := url.Parse(strings.TrimRight(m, ".,;:!?*"))
		if err != nil || u.Hostname() == "" {
			continue
		}
		out = append(out, u)
	}
	return out
}

func githubHost(host string) bool {
	return host == "github.com" || host == "www.github.com" || host == "raw.githubusercontent.com" || host == "codeload.github.com"
}

func publicDomain(host string) bool {
	return !githubHost(host) && host != "localhost" && net.ParseIP(host) == nil && domainPattern.MatchString(host)
}

func domainRefs(a *inventory.Artifact) []refs.Ref {
	var out []refs.Ref
	eachLine(a, func(line int, text string) {
		for _, u := range urls(text) {
			if host := strings.ToLower(u.Hostname()); publicDomain(host) {
				out = append(out, newURLRef(refs.Domain, host, u.String(), a.Path, line))
			}
		}
	})
	return out
}

func githubRefs(a *inventory.Artifact) []refs.Ref {
	var out []refs.Ref
	eachLine(a, func(line int, text string) {
		for _, v := range githubValues(text) {
			out = append(out, newURLRef(refs.GitHub, v.value, v.url, a.Path, line))
		}
	})
	return out
}

type githubValue struct {
	value string
	url   string
}

func githubValues(text string) []githubValue {
	var values []githubValue
	for _, u := range urls(text) {
		if !githubHost(strings.ToLower(u.Hostname())) {
			continue
		}
		if v, ok := githubPath(u.Path); ok {
			values = append(values, githubValue{value: v, url: u.String()})
		}
	}
	for _, re := range []*regexp.Regexp{gitSSHPattern, ghClonePattern} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if v, ok := githubName(m[1], m[2]); ok {
				values = append(values, githubValue{value: v})
			}
		}
	}
	return values
}

func githubPath(p string) (string, bool) {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if parts[0] == "" || githubSections[strings.ToLower(parts[0])] {
		return "", false
	}
	repo := ""
	if len(parts) > 1 {
		repo = parts[1]
	}
	return githubName(parts[0], repo)
}

func githubName(owner, repo string) (string, bool) {
	if !ownerPattern.MatchString(owner) {
		return "", false
	}
	repo = strings.TrimSuffix(repo, ".git")
	if repo == "" {
		return strings.ToLower(owner), true
	}
	if !repoPattern.MatchString(repo) {
		return "", false
	}
	return strings.ToLower(owner + "/" + repo), true
}
