**skill-verdict** is a security scanner for agent skills. A skill is not just a directory with markdown and supporting files - any external references mentioned there are the equal part of a skill. The scanner checks those files and their external dependencies and references without executing the skill's code.

## Why skill-verdict?

A skill can contain nothing malicious and still expose you to an attack. An attacker can take over an abandoned dependency and replace its contents or a lot of other approaches when the skill's files stay unchanged, but the agent receives the attacker's instructions or code.

**skill-verdict** checks external resources for signs of takeover, replacement or a potential for that. A finding does not require an existing malicious payload. The scanner looks up:

- Domain registration data, redirects chain, certificate details, "category" of a domain - for every mentioned link. Would you trust downloading a file for work from a random file-sharing service or via a bit.ly link?
- GitHub accounts and repositories, their age, popularity, and, obviously, existence.
- npm and PyPI packages as well as a wide variety of shady or very unsafe ways of working with package managers.

Skills often need external documentation or software. A link alone does not make a skill malicious, but documentation hidden behind bit.ly, instructions in github gist or a random package installed with @latest - at least are vulnerable.

No score is calculated because a single vulnerability is enough to compromise your environment. But you control the checks and decision severity.

## Why so many "unpinned XXX" findings?

The scanner reports every install that has no fixed version or a code execution from a "moving target":

- A dependency with no version.
- An install command with no version.
- A clone of a branch.
- A download link with replaceable content behind it.

You'd say that it generates noise findings, but let me try to convince you with some data.

Between September 2025 and September 2026, official releases of well-known packages on their official registries shipped malware in at least nine separate incidents[^1]. Some of the affected packages:

- axios, about 100 million downloads a week.
- keyv and flat-cache, which sit under eslint. The same wave poisoned 444 packages with about 2 billion combined monthly installs.
- The official SDKs of Zapier, PostHog and Postman.
- LiteLLM on PyPI.
- Trivy and KICS, which are security scanners.
- 32 packages in Red Hat's own npm namespace.

None of these were typosquats. Some of the poisoned releases carried valid signatures. Each poisoned version stayed public for hours to days before removal. Before playing a roulette - be honest with yourself and assess what might be the damage from you being one of millions of people affected by such attacks.

An unpinned install takes the newest release on every run - roulette, but fast and no actions from you.

A version pinned to a hash stays the same after your review (which you should do, at least with AI agents) - safe, but manually updating is on you.

## What it checks?

- Attempts to read or send sensitive data.
- Instructions that weaken security settings or redirect the agent away from the user's request.
- Commands that download and execute code.
- Text concealment through invisible characters or altered spelling.
- Unregistered domains or domains approaching expiry.
- GitHub accounts that no longer exist or have changed names.
- GitHub repositories that no longer exist.
- Missing packages or package names that resemble popular ones.
- Certain download and instruction links whose owners can replace the content at the same address.
- Compiled programs and installers.
- Encrypted archives.
- Unsupported file formats and files over scan limits.

Optional model checks look for harmful intent beyond fixed text patterns. Model review can lower or dismiss findings according to your configuration - we perfectly understand that a noisy report with a ton of false-positives is not worth even reading, so you'd better run it with the model reviewer level to get rid of most of false-positives.

An optional honeypot gives a model the tools of a coding agent. None of that tools are doing anything, and none of your installed agents or environments are used for such an "experiment". But if a model attempts to execute anything while being told not to - it's either an extremely dumb model or an interesting prompt injection technique that's not detectable by any static checks. And if so - we can't trust any further model analysis of the skill so it's not performed and a skill is immediately marked for blocking.

## Install

Download the binary for your platform from [Releases](https://github.com/bickus/skill-verdict/releases). Check it against `SHA256SUMS` and put it on `PATH`. Or install with [Go](https://go.dev/doc/install) 1.26 or later:

```sh
go install github.com/bickus/skill-verdict/pkg/cmd/skill-verdict@latest
skill-verdict --version
```

The result is a single static binary with no runtime or shared libraries to install. See [Install](docs/user/install.md) for platform builds from a checkout.

N.B.: If you copy-pasted and executed that "go install ...@latest" - you should trigger your internal trigger on any @latest and avoid using them :)

## Run a scan

Generate a starting configuration file:

```sh
skill-verdict config-template config.json
```

This file lists every setting and rule with its default value. For a full scan with AI-assisted review, set your model in `config.json`:

- `llm.model`: your model name (e.g., `gpt-5.6-luna`, `google/gemini-3.8-flash`).
- `llm.baseUrl`: your OpenAI-compatible API endpoint (defaults to OpenRouter, works with OpenAI, Ollama, vLLM, etc.). See [Model settings](docs/user/config.md#configuring-llm-providers).

Set your API key in your environment. Setting a GitHub token is optional but recommended to avoid hitting GitHub API rate limits during external reference checks:

```sh
export SKILL_VERDICT_API_KEY=your-api-key
export GITHUB_TOKEN=your-github-token
```

Now scan a skill:

```sh
skill-verdict scan --config config.json ./path/to/skill
```

Scan multiple skills in one run:

```sh
skill-verdict scan --config config.json ./path/to/skill "./my skills/other"
```

Or scan every skill located inside a directory:

```sh
skill-verdict scan --config config.json ./path/to/skills
```

Want an instant, air-gapped offline scan with zero network calls and zero AI tokens? Turn off the four network/model layers (`references`, `honeypot`, `discovery`, and `judge`) in your config. See [Configuration](docs/user/config.md) for how to tune every layer, and [Rules](docs/user/rules.md) for the full catalog of checks.

## Understand the result

Every scan produces a clear terminal report (or JSON for machines). Each finding shows:

- **Severity**: `low`, `medium`, `high`, or `critical`.
- **Location**: the exact file and line number in the skill.
- **Evidence**: the specific text, command, or external fact that triggered the alert.
- **AI Judge Analysis**: if model review is enabled, the Judge explains whether the finding is a genuine risk or harmless context, and whether it was upheld, lowered, or dismissed.

The scan concludes with one of four overall verdicts:

| Verdict | Meaning |
|---|---|
| `clean` | All requested checks completed with zero active findings. (Remember: clean means no known red flags were found, not a guarantee of safety). |
| `review` | Findings were detected, but none reached your blocking threshold. A human should look over the report. |
| `incomplete` | Some checks could not finish (e.g. network timeout, rate limit, or unreadable file) and no blocking flaw was found yet. Treat this with caution — unverified dependencies are not clean. |
| `block` | A finding reached or exceeded your configured blocking severity. Do not install or run this skill. |

By default, `high` and `critical` findings produce a `block` verdict. You can adjust this threshold (`gate.severity`) in [Configuration](docs/user/config.md#the-gate-and-blocking-policy).

See [Verdicts](docs/user/verdicts.md) for full details on how decisions are reached.

## Use in automation

Use `skill-verdict` in pre-commit hooks, CI pipelines, or automated agent onboarding scripts:

```sh
skill-verdict scan --config config.json --format json --fail-on review ./path/to/skills
```

- `--fail-on block` (default): Exits with code `1` only when a skill is blocked.
- `--fail-on review`: Strict enforcement — exits with code `1` on anything other than `clean`.
- `--fail-on incomplete`: Fails if any check could not finish, preventing silent blind spots.
- Command errors (invalid arguments, missing config) exit with code `2`.

See [Usage](docs/user/usage.md) for complete CLI options, output fields, and automation examples.

## Privacy and limits

You have complete control over what `skill-verdict` sends over the wire:

- Offline scans make no network requests.
- External checks send referenced names to lookup services. The scanner also requests domain roots to check availability and redirects. The scanner does not fetch linked documentation or download referenced code.
  - **CAUTION:** Requesting domain roots for mentioned links might expose your IP to owners of such domains. Make sure to disable the references check if this is important for you.
- Model checks send skill content and scan context to your configured model provider.

See [External references](docs/user/references.md) for full details on network behavior.

Keep in mind: `skill-verdict` does not execute skills or inspect the inside of encrypted archives. Heuristic rules and models can both miss zero-day techniques or flag legitimate code. Always review the evidence in the report, re-scan periodically (since external dependencies change over time), and restrict your agent's operational permissions.

## Documentation

| Guide | What you will find |
|---|---|
| [Install](docs/user/install.md) | Binaries, verification, building from source, and scan modes. |
| [Usage](docs/user/usage.md) | Command line usage, reading reports, and CI/automation recipes. |
| [Configuration](docs/user/config.md) | Fine-tuning rules, model endpoints, limits, and the Gate policy. |
| [Verdicts](docs/user/verdicts.md) | How the four verdicts work, the AI Judge, and why incomplete scans matter. |
| [Detection](docs/user/detection.md) | How the inspection pipeline works from walk limits to honeypot traps. |
| [External references](docs/user/references.md) | Deep dive into supply chain checks, domain takeovers, and network safety. |
| [Rules](docs/user/rules.md) | Complete catalog of security checks organized by threat category. |

## Research and contributions

AIR's [SkillJacking](https://www.air.security/blog-posts/skilljacking) research inspired the external checks. The researchers demonstrated takeovers through abandoned dependencies in legitimate skills.

The repository includes an isolated benchmark runner for [MalSkillBench](https://github.com/lxyeternal/MalSkillBench), a benchmark of malicious agent skills by Wenbo Guo, Wei Zeng, Chengwei Liu, Xiaojun Jia, Yijia Xu, Lei Tang, Yong Fang and Yang Liu ([paper](https://arxiv.org/abs/2606.07131)). See [Scripts](scripts/README.md) for its use and attribution.

Contributions require a DCO sign-off. See [CONTRIBUTING.md](CONTRIBUTING.md) for submission requirements.

## License

Licensed under [Apache License 2.0](LICENSE). See [NOTICE](NOTICE) and [third_party](third_party/README.md) for third-party acknowledgments.

[^1]: Sources, one or two per incident:
    - [axios post-mortem](https://github.com/axios/axios/issues/10636) and the [CISA alert](https://www.cisa.gov/news-events/alerts/2026/04/20/supply-chain-compromise-impacts-axios-node-package-manager), March 2026.
    - [Microsoft Threat Intelligence on ChainDrop](https://www.microsoft.com/en-us/security/blog/2026/08/04/chaindrop-supply-chain-compromise-anatomy-self-propagating-worm), keyv, flat-cache and 444 packages, August 2026.
    - [PostHog post-mortem](https://posthog.com/blog/nov-24-shai-hulud-attack-post-mortem) and [Unit 42](https://unit42.paloaltonetworks.com/npm-supply-chain-attack) on Shai-Hulud 2.0, Zapier, PostHog, Postman and AsyncAPI SDKs, November 2025.
    - [Trivy advisory GHSA-69fq-xp46-6x23](https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23) and [SANS](https://www.sans.org/blog/axios-npm-supply-chain-compromise-malicious-packages-remote-access-trojan) on the TeamPCP wave, Trivy, KICS, LiteLLM and Telnyx, March 2026.
    - [Red Hat RHSB-2026-006](https://access.redhat.com/security/vulnerabilities/RHSB-2026-006), 32 packages in the @redhat-cloud-services namespace, June 2026.
