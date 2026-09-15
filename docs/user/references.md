# External References & Supply Chain

A skill file can appear completely clean while silently directing your AI agent to download malware from an abandoned domain, an expired GitHub username, or an unpinned package.

This vector — known as **SkillJacking** — exploits the fact that agent skills are deeply integrated with the wider internet. `skill-verdict` inspects not just the static text of a skill, but the live status of the external resources it relies on.

---

## What the Scanner Extracts

During scanning, `skill-verdict` parses all text files and extracts four categories of external references:

| Kind | What Is Detected | Examples |
|---|---|---|
| **Domains** | Hostnames extracted from HTTP/HTTPS URLs. | `https://example.com/docs`, `http://api.service.io` |
| **GitHub** | GitHub repositories, SSH clone URLs, and `gh repo clone` commands. | `github.com/org/repo`, `git@github.com:user/tool.git` |
| **Packages** | Package install commands (`pip`, `pipx`, `uv`, `npm`, `pnpm`, `yarn`, `npx`), `package.json` dependencies, and `requirements.txt`. | `pip install requests`, `npm i express` |
| **IP Addresses** | Raw IPv4/IPv6 addresses in URLs, `host:port` pairs, or network commands (`curl`, `nc`, `ssh`, etc.). | `http://198.51.100.4:8080`, `nc 203.0.113.5 4444` |

References are deduplicated across the entire skill so each external resource is evaluated only once per scan.

---

## Domain Checks & Takeover Detection

When a domain is detected, `skill-verdict` evaluates whether the domain is legitimate, abandoned, or suspicious:

1. **Registration & RDAP Lookup**:
   - The scanner queries public RDAP (Registration Data Access Protocol) servers to discover:
     - **Domain Age**: Is the domain brand-new (`DOMAIN-NEW`)? New domains are frequently registered for one-off malicious campaigns.
     - **Expiration Date**: Is the domain about to expire (`DOMAIN-EXPIRING`)? If a skill depends on an expiring domain, an attacker can register it the moment it lapses and control whatever the agent downloads.
     - **Unregistered Domains**: Does the domain exist at all (`DOMAIN-UNREGISTERED`)? If a skill references an unregistered domain, anyone can buy it right now and serve payloads to your agent.
2. **Safe Root HTTP Probe**:
   - The scanner sends a safe `GET` request to the **root** of the domain (e.g. `https://example.com/`).
   - **Crucial Security Rule**: The scanner **never** requests the specific subpath mentioned in the skill (e.g. `https://example.com/api/v1/trigger?token=xyz`). Requesting subpaths could trigger webhooks, delete resources, or alert an attacker.
   - The root request uses standard browser headers (Chrome on Windows) to verify that the server is alive and to track redirects.
3. **Suspicious Host Identification**:
   - **URL Shorteners (`DOMAIN-SHORTENER`)**: Detects `bit.ly`, `tinyurl.com`, `t.co`, etc. Legitimate skills should never hide destinations behind link shorteners.
   - **Paste & Snippet Sites (`DOMAIN-PASTE`)**: Detects `pastebin.com`, `gist.github.com`, etc. Code behind paste sites can be modified at any time without version control or review.
   - **File Sharing Services (`DOMAIN-FILEHOST`)**: Detects anonymous file lockers and temporary file hosts (`mega.nz`, `dropbox.com`, `mediafire.com`).
4. **Cross-Domain Redirects (`DOMAIN-NEW-REDIRECT`)**:
   - If a newly registered domain redirects to a different registered domain, it is flagged as a high-risk obfuscation tactic.

---

## GitHub Checks

Referencing a GitHub repository without pinning a commit hash exposes you to repository hijacking:

1. **Missing Owners (`GITHUB-OWNER-MISSING`)**:
   - If the GitHub account that originally owned the repository has been deleted, **anyone can register that exact username** on GitHub and recreate the repository with malicious code. This is an immediate critical risk.
2. **Renamed Owners (`GITHUB-OWNER-RENAMED`)**:
   - If an account renamed itself, GitHub automatically redirects traffic to the new name. However, the old username is now free to be claimed by anyone.
3. **Missing Repositories (`GITHUB-REPO-MISSING`)**:
   - If the account exists but the repository was deleted, the owner could recreate it with arbitrary code at any time.
4. **New Owner Accounts (`GITHUB-OWNER-NEW`)**:
   - Flags accounts created within the last 90 days.

---

## Package Registry Checks (npm & PyPI)

Package dependencies in skills are verified against live registry data:

1. **Missing Packages (`PACKAGE-MISSING`)**:
   - If a skill tells the agent to `pip install my-helper`, but `my-helper` is not published on PyPI, an attacker can register the package name and immediately achieve remote code execution on your system (dependency confusion).
2. **Typosquatting (`PACKAGE-TYPOSQUAT`)**:
   - Compares package names against popular libraries. If a skill requires `reqeusts` instead of `requests`, or `chalk-color` instead of `chalk`, it is flagged for typosquatting.
3. **New & Low-Download Packages (`PACKAGE-NEW`, `PACKAGE-LOW-DOWNLOADS`)**:
   - Brand-new packages and packages with negligible download counts represent unvetted code.
4. **Non-Standard Registries (`PACKAGE-OTHER-REGISTRY`)**:
   - Flags install commands specifying `--index-url` or `--registry` pointing to untrusted third-party servers.

---

## Network Safety & Privacy Cautions

### 1. IP Address Disclosure

> **CAUTION:** When `skill-verdict` verifies external domains, it sends an HTTPS request to the domain root. This means **your IP address will appear in the server access logs** of the owner of that domain.

If you are analyzing suspected malware or need total anonymity, do not run with default network settings.

### 2. Running Completely Offline

To disable all network requests and perform an air-gapped local scan:

```json
{
  "layers": {
    "references": {"enabled": false},
    "honeypot": {"enabled": false},
    "discovery": {"enabled": false},
    "judge": {"enabled": false}
  }
}
```

With `references.enabled: false`, zero network sockets will be opened.

### 3. Avoiding GitHub API Rate Limits

Anonymous requests to the GitHub API are capped at 60 per hour per IP. If you scan multiple skills, you will quickly encounter rate limits, leading to `incomplete` scans.

Set a GitHub personal access token in your environment:

```sh
export GITHUB_TOKEN=ghp_yourTokenHere
```

A fine-grained token with **zero repository permissions** (public read-only) increases your allowance to 5,000 requests per hour.

---

## Tuning Reference Checks

You can fine-tune timeouts, request pacing, and age thresholds in `config.json`:

```json
{
  "layers": {
    "references": {
      "timeout": "10s",
      "budget": "120s",
      "concurrency": 8,
      "requestDelay": "1s",
      "retryDelays": ["2s", "5s"],
      "youngDomainDays": 365,
      "expiryDays": 14,
      "youngOwnerDays": 90,
      "newPackageDays": 90,
      "lowDownloads": 1000,
      "maxReferences": 100
    }
  }
}
```

| Setting | Default | Description |
|---|---|---|
| `budget` | `120s` | Maximum total time allocated for all external checks in a single skill. |
| `timeout` | `10s` | Timeout for an individual HTTP request. |
| `concurrency` | `8` | Maximum number of concurrent outbound host connections. |
| `requestDelay` | `1s` | Rate-limiting pause between consecutive requests to the same host. |
| `youngDomainDays` | `365` | Domains younger than this are flagged as `DOMAIN-NEW`. |
| `expiryDays` | `14` | Domains expiring within this window trigger `DOMAIN-EXPIRING`. |
| `maxReferences` | `100` | Skills referencing more items than this trigger `REFERENCES-TOO-MANY` and halt the scan. |
