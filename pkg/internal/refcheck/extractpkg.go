// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"encoding/json"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const (
	registryPyPI = "pypi"
	registryNPM  = "npm"
)

var (
	pipInstall  = regexp.MustCompile(`(?:^|[\s;&|(])(?:python3?\s+-m\s+pip|pip3?|pipx|uv\s+pip)\s+install\s+(.+)$`)
	uvxRun      = regexp.MustCompile(`(?:^|[\s;&|(])uvx\s+(.+)$`)
	npmInstall  = regexp.MustCompile(`(?:^|[\s;&|(])(?:npm|pnpm|yarn)\s+(?:install|i|add)\s+(.+)$`)
	npxRun      = regexp.MustCompile(`(?:^|[\s;&|(])npx\s+(.+)$`)
	shellEnd    = regexp.MustCompile("&&|\\|\\||[;|)`#]")
	pypiPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	npmPattern  = regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
)

var pipValueFlags = []string{
	"-e", "--editable", "-r", "--requirement", "-c", "--constraint", "-i", "--index-url",
	"--extra-index-url", "--index", "--default-index", "-f", "--find-links", "-t", "--target", "--python", "--prefix", "--root",
}

var npmValueFlags = []string{
	"--before", "--cache", "--cpu", "--fetch-retries", "--fetch-retry-factor", "--fetch-retry-maxtimeout",
	"--fetch-retry-mintimeout", "--https-proxy", "--include", "--install-strategy", "--libc", "--node-options",
	"--omit", "--os", "--otp", "--prefix", "--proxy", "--registry", "--save-prefix", "--scope",
	"--script-shell", "--tag", "--userconfig", "--workspace", "-w",
}

var registryFlags = []string{"--registry", "-i", "--index-url", "--extra-index-url", "--index", "--default-index", "-f", "--find-links"}

var registryEnv = []string{
	"NPM_CONFIG_REGISTRY", "YARN_NPM_REGISTRY_SERVER", "PIP_INDEX_URL", "PIP_EXTRA_INDEX_URL", "PIP_FIND_LINKS",
	"UV_INDEX_URL", "UV_INDEX", "UV_DEFAULT_INDEX", "UV_EXTRA_INDEX_URL", "UV_FIND_LINKS",
}

func installRegistry(text string) string {
	fields := strings.Fields(text)
	for i, tok := range fields {
		tok = strings.Trim(tok, `"'`)
		key, value, joined := strings.Cut(tok, "=")
		if !joined && slices.Contains(registryFlags, key) && i+1 < len(fields) {
			value = fields[i+1]
		} else if !joined || !slices.Contains(registryFlags, key) && !slices.Contains(registryEnv, key) {
			continue
		}
		if value = strings.Trim(value, `"'`); strings.Contains(value, "://") {
			return value
		}
	}
	return ""
}

func packageValue(registry, name string) string {
	return registry + ":" + name
}

func packageRefs(a *inventory.Artifact) []refs.Ref {
	var out []refs.Ref
	base := path.Base(a.Path)
	if base == "package.json" {
		return packageJSONRefs(a)
	}
	requirements := strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt")
	eachLine(a, func(line int, text string) {
		if requirements {
			if name, ok := requirementName(text); ok {
				out = append(out, newRef(refs.Package, packageValue(registryPyPI, name), a.Path, line))
			}
			return
		}
		for _, name := range commandPackages(text, registryPyPI) {
			out = append(out, newRef(refs.Package, packageValue(registryPyPI, name), a.Path, line))
		}
		for _, name := range commandPackages(text, registryNPM) {
			out = append(out, newRef(refs.Package, packageValue(registryNPM, name), a.Path, line))
		}
	})
	return out
}

func commandPackages(text, registry string) []string {
	install, run := pipInstall, uvxRun
	if registry == registryNPM {
		install, run = npmInstall, npxRun
	}
	var names []string
	if m := install.FindStringSubmatch(text); m != nil {
		names = append(names, installArgs(m[1], registry)...)
	}
	if m := run.FindStringSubmatch(text); m != nil {
		names = append(names, firstArg(m[1], registry)...)
	}
	return names
}

func shellArgs(args string) []string {
	if loc := shellEnd.FindStringIndex(args); loc != nil {
		args = args[:loc[0]]
	}
	fields := strings.Fields(args)
	for i, f := range fields {
		fields[i] = strings.Trim(f, `"'`)
	}
	return fields
}

func installArgs(args, registry string) []string {
	var out []string
	skip := false
	for _, tok := range shellArgs(args) {
		switch {
		case skip:
			skip = false
		case registry == registryPyPI && slices.Contains(pipValueFlags, tok):
			skip = true
		case registry == registryNPM && slices.Contains(npmValueFlags, tok):
			skip = true
		case strings.HasPrefix(tok, "-"):
		default:
			if name, ok := packageName(tok, registry); ok {
				out = append(out, name)
			}
		}
	}
	return out
}

func firstArg(args, registry string) []string {
	for _, tok := range shellArgs(args) {
		if strings.HasPrefix(tok, "-") {
			continue
		}
		if name, ok := packageName(tok, registry); ok {
			return []string{name}
		}
		return nil
	}
	return nil
}

func packageName(tok, registry string) (string, bool) {
	if registry == registryNPM {
		return npmName(tok)
	}
	return pypiName(tok)
}

func pathLike(tok string) bool {
	return strings.Contains(tok, "/") || strings.HasPrefix(tok, ".") || strings.Contains(tok, ":")
}

func pypiName(tok string) (string, bool) {
	if pathLike(tok) {
		return "", false
	}
	name := strings.ToLower(tok)
	if i := strings.IndexAny(name, "[=<>!~;@"); i >= 0 {
		name = name[:i]
	}
	name = strings.ReplaceAll(name, "_", "-")
	return name, pypiPattern.MatchString(name)
}

func npmName(tok string) (string, bool) {
	if i := strings.Index(tok, "@npm:"); i > 0 {
		tok = tok[i+len("@npm:"):]
	}
	if i := strings.LastIndex(tok, "@"); i > 0 {
		tok = tok[:i]
	}
	name := strings.ToLower(tok)
	return name, npmPattern.MatchString(name)
}

func requirementName(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "-") {
		return "", false
	}
	if i := strings.IndexAny(text, " \t#"); i >= 0 {
		text = text[:i]
	}
	return pypiName(text)
}

func packageJSONRefs(a *inventory.Artifact) []refs.Ref {
	var file struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(a.Content), &file); err != nil {
		return nil
	}
	lines := packageJSONLines(a.Content)
	var out []refs.Ref
	groups := []struct {
		name string
		deps map[string]string
	}{{"dependencies", file.Dependencies}, {"devDependencies", file.DevDependencies}}
	for _, group := range groups {
		for _, key := range slices.Sorted(maps.Keys(group.deps)) {
			name, ok := dependencyName(key, group.deps[key])
			if !ok {
				continue
			}
			line := lines[group.name][key]
			if line == 0 {
				line = 1
			}
			out = append(out, newRef(refs.Package, packageValue(registryNPM, name), a.Path, line))
		}
	}
	return out
}

func packageJSONLines(content string) map[string]map[string]int {
	dec := json.NewDecoder(strings.NewReader(content))
	open, err := dec.Token()
	if err != nil || open != json.Delim('{') {
		return nil
	}
	out := make(map[string]map[string]int)
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		section, ok := tok.(string)
		if !ok {
			return nil
		}
		if section != "dependencies" && section != "devDependencies" {
			var discard json.RawMessage
			if err = dec.Decode(&discard); err != nil {
				return nil
			}
			continue
		}
		found, ok := packageObjectLines(dec, content)
		if !ok {
			return nil
		}
		out[section] = found
	}
	return out
}

func packageObjectLines(dec *json.Decoder, content string) (map[string]int, bool) {
	open, err := dec.Token()
	if err != nil {
		return nil, false
	}
	if open == nil {
		return nil, true
	}
	if open != json.Delim('{') {
		return nil, false
	}
	found := make(map[string]int)
	for dec.More() {
		offset := int(dec.InputOffset())
		var tok json.Token
		tok, err = dec.Token()
		key, ok := tok.(string)
		quote := strings.IndexByte(content[offset:], '"')
		if err != nil || !ok || quote < 0 {
			return nil, false
		}
		found[key] = strings.Count(content[:offset+quote], "\n") + 1
		var value json.RawMessage
		if err = dec.Decode(&value); err != nil {
			return nil, false
		}
	}
	_, err = dec.Token()
	return found, err == nil
}

func dependencyName(key, value string) (string, bool) {
	if rest, ok := strings.CutPrefix(value, "npm:"); ok {
		return npmName(rest)
	}
	if strings.ContainsAny(value, ":/") {
		return "", false
	}
	return npmName(key)
}
