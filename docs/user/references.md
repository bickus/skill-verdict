# External references

A skill can look harmless while sending users to a domain, repository or package that an attacker
controls. skill-verdict checks those external names instead of relying only on the files in the
skill.

Reference checks run whenever `layers.references.enabled` is true, with or without the model layers.

## What the scanner recognizes

| Kind | Recognized text | Ignored |
|---|---|---|
| Domain | Host name from a URL | IP addresses, `localhost` and GitHub itself |
| GitHub | Repository URLs, SSH clone names and `gh repo clone` commands | GitHub pages that are not owners or repositories |
| Package | Install commands for pip, pipx, uv, npm, pnpm, Yarn and npx; Python requirement files; `package.json` dependencies | Flags, URLs and local paths |
| Address | IPv4 or IPv6 address as the host of a URL, as `host:port`, or as the target of `nc`, `ncat`, `netcat`, `socat`, `ssh`, `scp`, `sftp`, `telnet`, `curl`, `wget`, `ftp`, `tftp` or `rsync` | Addresses in plain text and version numbers |

Blank lines and lines whose first non-space character is `#` are skipped during reference
extraction. References are combined across the skill so the same external name is checked once. A
scan of several skills also reuses a result when they name the same reference.

## Facts

A fact is a value that the responsible service returned, for example a domain registration date or whether a package exists. Some facts produce findings, listed under [Findings](#findings). Every fact goes into the reference brief, see [The brief](#the-brief).

## Domain checks

The scanner uses RDAP, the public domain-registration lookup system, to find the registered domain,
registration dates and registrar. It then sends one HTTPS request to the host's root and follows
redirects. It does not request the path found in the skill.
A request to that path could trigger whatever the path does, so the scanner never sends one.
The request to the root sends the same headers as Chrome on Windows when a user opens the page. A site cannot tell the scanner from a visitor by its headers.

If a redirect ends on a different registered domain, that destination is checked once as well.

## Address checks

An address gets no lookup. The scanner never connects to it and needs no network for this check. The report records whether the address is public in the `public` fact. A public address produces `IP-ADDRESS-PUBLIC`. Any other address produces no reference finding.
Hosts on shared services, such as `github.io`, skip registration checks because the service's
registration says nothing about the individual site.

Domain results can contain these facts:

| Field | Meaning |
|---|---|
| `registrable` | Registered domain that owns the host name. |
| `registered`, `expires` | Whether it is registered and the registration dates. |
| `age_days` | Days since registration. |
| `registrar` | Registration provider. |
| `privacy` | Registration details are hidden by a privacy service. |
| `platform` | Shared hosting service used by the host. |
| `final_url`, `redirect_cross_domain` | Destination of a redirect that left the registered domain, and the domain it reached. |
| `http_status` | Status of the root answer when it is 4xx or 5xx. |
| `connection_error` | Error text of a connection that the host refused or dropped. |
| `tls_validation` | Validation level of the certificate the host presented: `DV`, `OV`, `IV` or `EV`. |
| `tls_organization` | Organization a certificate authority verified, present for OV, IV and EV certificates. |

A domain registered within `youngDomainDays` produces `DOMAIN-NEW`. Three host lists produce a finding on their own: a URL shortener gives `DOMAIN-SHORTENER`, a paste or snippet site gives `DOMAIN-PASTE`, and a file sharing service gives `DOMAIN-FILEHOST`.

A 4xx or 5xx status from the root produces `DOMAIN-HTTP-ERROR`. A rate limit answer does not count, see [Request pacing](#request-pacing).

A host the scanner cannot reach produces `DOMAIN-UNREACHABLE`. The finding needs one of these errors:

- the name does not resolve
- the TLS handshake fails or the certificate is not valid
- the server refuses, resets or closes the connection

A timeout gives a `failed` lookup and no finding, because a slow or offline scanner causes timeouts too.

## GitHub checks

The scanner asks the GitHub API whether the owner and repository exist. It records the owner's
type, age and repository count, and the repository's stars, age, last update, fork status and
archive status.

An owner name that redirects to another account is treated as renamed. An owner created within `youngOwnerDays` produces `GITHUB-OWNER-NEW`.

## Package checks

The scanner asks `packages.ecosyste.ms` for the package's first publication date, latest version and last month's download count. Two answers send the scanner to npm or PyPI for an existence check:

- the statistics service does not know the package
- the statistics service does not answer

Only the registry's answer decides that a package is missing.

An install command that names another registry or index with an option such as `--registry` or `--index-url` is still checked against the default registry. The brief notes at that location which registry the command installs from, and `PACKAGE-OTHER-REGISTRY` reports the command.

It also compares the package name with a built-in list of popular packages. A name one or two edits away from a popular one produces `PACKAGE-TYPOSQUAT`. A package first published within `newPackageDays` produces `PACKAGE-NEW`. Fewer than `lowDownloads` downloads last month produces `PACKAGE-LOW-DOWNLOADS`.

## Findings

Every finding below needs an answer from the responsible service. For `DOMAIN-UNREACHABLE` that answer is the host refusing the connection. A timeout or a malformed response creates none. Three rules need no request:

- `IP-ADDRESS-PUBLIC` follows from the address itself
- `PACKAGE-TYPOSQUAT` follows from the package name
- `REFERENCES-TOO-MANY` follows from the number of references

| Rule | Severity | Meaning |
|---|---|---|
| `IP-ADDRESS-PUBLIC` | critical | A URL, host and port, or network command targets a public IP address. |
| `DOMAIN-SHORTENER` | critical | The host is a URL shortener. |
| `DOMAIN-PASTE` | critical | The host is a paste or snippet site. |
| `DOMAIN-FILEHOST` | critical | The host is a file sharing service. |
| `DOMAIN-NEW` | medium | The domain registration is younger than `youngDomainDays`. |
| `DOMAIN-NEW-REDIRECT` | critical | A domain registered within `youngDomainDays` redirects to a different registered domain. |
| `DOMAIN-EXPIRING` | high | Registration expires within `expiryDays`. |
| `DOMAIN-UNREGISTERED` | high | The domain is available for anyone to register. |
| `DOMAIN-HTTP-ERROR` | high | The host root answers with a 4xx or 5xx status. |
| `DOMAIN-UNREACHABLE` | high | The host refuses or drops the connection, or its name does not resolve. |
| `GITHUB-OWNER-MISSING` | critical | The GitHub owner name does not exist. |
| `GITHUB-OWNER-RENAMED` | critical | The repository resolves to a different owner name, leaving the original name available. |
| `GITHUB-OWNER-NEW` | medium | The GitHub owner account is younger than `youngOwnerDays`. |
| `GITHUB-REPO-MISSING` | high | The owner exists but the repository does not. |
| `PACKAGE-LOW-DOWNLOADS` | low | The package had fewer than `lowDownloads` downloads last month. |
| `PACKAGE-MISSING` | critical | The npm or PyPI package does not exist. |
| `PACKAGE-NEW` | medium | The first release of the package is younger than `newPackageDays`. |
| `PACKAGE-TYPOSQUAT` | high | The package name is one or two edits away from a popular package. |
| `REFERENCES-TOO-MANY` | high | The skill has more distinct references than `maxReferences`. |

`REFERENCES-TOO-MANY` stops the scan by default, before any lookup. A skill rarely needs that many references, and a long list can hide a malicious reference among valid ones. The count covers all kinds of references together and includes domains reached through a redirect. With `interrupt` off on this rule, the scanner checks the first `maxReferences` references and marks the rest `not-checked`.

Model review sees every finding in this table and reads the brief for the facts behind it. The model may lower a finding to the floor in [Rules](rules.md) and no further. The model may dismiss a finding whose rule has no floor. See [Verdicts](verdicts.md) for what the severity means for the verdict.

## The brief

The reference check writes `references.brief.md`, a Markdown artifact of kind `brief` with origin `references`. The model layers list it with the skill's files and read it like any other file. The brief has one section per reference kind and one heading per reference with:

- the status and, for a failed lookup, the error text
- `derived: true` when the scanner reached the reference through a redirect
- every file and line where the skill uses the reference, and the text of that line
- every full URL the skill wrote for the reference, one `url:` line each
- every fact the services returned, one `key: value` line each

The brief has no findings and no opinions. Rule IDs stay in the findings list.

## Checks that do not finish

A reference is `not-checked` in these cases:

- the time budget runs out
- an API rate limit is still there after the last retry
- the skill has more than `maxReferences` references and `interrupt` is off on `REFERENCES-TOO-MANY`

A reference is `failed` in these cases:

- a request times out
- the network on the scanner side fails
- the scanner cannot read the response

Either result makes the scan `incomplete` because the external name was not verified. A domain host that refuses the connection gives `DOMAIN-UNREACHABLE` and counts as checked.

| Setting | Default | Purpose |
|---|---|---|
| `layers.references.timeout` | `10s` | Time allowed for one request. |
| `layers.references.budget` | `120s` | Total time allowed for one skill. |
| `layers.references.maxReferences` | `100` | More distinct references in one skill produce `REFERENCES-TOO-MANY`. |
| `layers.references.concurrency` | `8` | Hosts with a request in flight at once. |
| `layers.references.requestDelay` | `1s` | Pause between two requests to one host. |
| `layers.references.retryDelays` | `["2s", "5s"]` | Pauses before each retry. |
| `layers.references.maxBodyBytes` | `1048576` | Maximum response size in bytes. |
| `layers.references.maxRedirects` | `5` | Redirects followed. |

## Request pacing

Requests to one host go out one at a time, `requestDelay` apart. Requests to different hosts run in parallel, up to `concurrency` hosts at once.

One run sends each URL once. Two repositories of one GitHub owner share the owner lookup.

The scanner repeats a request after each pause in `retryDelays` when the answer was status 429, status 403 with a rate limit header, a 5xx status or a network error. When the last attempt still gets a rate limit, the scanner closes the host for the rest of the run. Every later reference on that host is `not-checked` with the reason `rate limit`.

## GitHub API limits

Without a token, GitHub allows 60 requests per hour from one address. A reference may require one request for the owner and another for the repository.

Set a token in the environment variable named by `layers.references.githubTokenEnv`, which is
`GITHUB_TOKEN` by default. A token with no scopes is enough for public repositories.

## Network safety

The scanner refuses to connect to any address that is not public. This applies to the
first request and every redirect. A host that resolves to such an address is marked
`failed`.

URLs with an IP address or `localhost` are not treated as external references. With
`layers.references.enabled` set to `false` the scanner creates no reference checks and makes no network
requests.
