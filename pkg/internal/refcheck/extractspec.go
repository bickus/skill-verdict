// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"regexp"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

var (
	specToken = regexp.MustCompile(`"kind"\s*:\s*"([\w-]+)"|(?:^|[\s{,])kind:\s*["']?([\w-]+)|"package"\s*:\s*"([^"]*)"|(?:^|[\s{,])package:\s*["']?([^\s"',}]+)|\}`)
	listEntry = regexp.MustCompile(`^\s*-\s+[\w-]+:`)
	mcpToken  = regexp.MustCompile(`"command"\s*:\s*"([^"]*)"|"args"\s*:\s*\[|"([^"]*)"|\]`)
)

type spec struct {
	registry string
	value    string
}

type specScan struct {
	kind    string
	command string
	args    bool
}

func specRefs(a *inventory.Artifact) []refs.Ref {
	var out []refs.Ref
	var s specScan
	eachLine(a, func(line int, text string) {
		for _, p := range append(s.metadata(text), s.server(text)...) {
			if name, ok := packageName(p.value, p.registry); ok {
				out = append(out, newRef(refs.Package, packageValue(p.registry, name), a.Path, line))
			}
		}
	})
	return out
}

func (s *specScan) metadata(text string) []spec {
	if listEntry.MatchString(text) {
		s.kind = ""
	}
	var out []spec
	for _, m := range specToken.FindAllStringSubmatch(text, -1) {
		switch {
		case m[1] != "" || m[2] != "":
			s.kind = m[1] + m[2]
		case m[3] != "" || m[4] != "":
			if registry := kindRegistry(s.kind); registry != "" {
				out = append(out, spec{registry, m[3] + m[4]})
			}
			s.kind = ""
		default:
			s.kind = ""
		}
	}
	return out
}

func (s *specScan) server(text string) []spec {
	var out []spec
	for _, m := range mcpToken.FindAllStringSubmatchIndex(text, -1) {
		switch {
		case m[2] >= 0:
			s.command, s.args = serverRegistry(text[m[2]:m[3]]), false
		case m[4] >= 0:
			if value := text[m[4]:m[5]]; s.args && !strings.HasPrefix(value, "-") {
				out = append(out, spec{s.command, value})
				s.command, s.args = "", false
			}
		case text[m[0]] == ']':
			s.args = false
		default:
			s.args = s.command != ""
		}
	}
	return out
}

func kindRegistry(kind string) string {
	switch strings.ToLower(kind) {
	case "node", "npm", "pnpm", "yarn", "bun":
		return registryNPM
	case "pip", "pipx", "uv", "python":
		return registryPyPI
	}
	return ""
}

func serverRegistry(command string) string {
	switch command {
	case "npx", "pnpx", "bunx":
		return registryNPM
	case "uvx":
		return registryPyPI
	}
	return ""
}
