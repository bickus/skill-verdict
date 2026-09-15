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

Write the default configuration to a file:

```sh
skill-verdict config-template config.json
```

The file lists every setting with its default value. Every layer and every rule is on. Before the first scan, set the model:

- `llm.model` is the model name.
- `llm.baseUrl` is its API endpoint. Any OpenAI-compatible API works, including a local server. See [Model settings](docs/user/config.md#model-settings).

The API key comes from the environment. A GitHub token is optional and increases the GitHub request allowance of the external checks:

```sh
export SKILL_VERDICT_API_KEY=your-api-key
export GITHUB_TOKEN=your-github-token
```

`llm.keyEnv` and `layers.references.githubTokenEnv` change the variable names.

Scan one skill:

```sh
skill-verdict scan --config config.json ./path/to/skill
```

Scan several skills in one run. Quote a path that contains spaces:

```sh
skill-verdict scan --config config.json ./path/to/skill "./my skills/other"
```

Scan every skill directly inside a directory:

```sh
skill-verdict scan --config config.json ./path/to/skills
```

A scan without a model or network needs four switches off: set `enabled` to `false` for `references`, `honeypot`, `discovery` and `judge` under `layers`. [Configuration](docs/user/config.md) describes every layer and model setting and how to change the severity of a rule. [Rules](docs/user/rules.md) lists every built-in rule.

## Understand the result

Each finding includes:

- Its severity.
- The evidence that triggered it.
- The location in the skill.
- AI Judge model might change/dismiss/uphold a finding - with explanation.

Final skill verdict are as below:

| Verdict | Meaning |
|---|---|
| `clean` | The requested checks completed without an active finding. A clean result does not guarantee safety. |
| `review` | Findings need attention, but none reached your blocking severity. |
| `incomplete` | Some requested checks could not finish. A blocking finding takes precedence. |
| `block` | A finding reached your configured blocking severity. |

By default, high and critical findings produce `block`. You can [configure](docs/user/config.md) this for yourself if default settings are not suitable for you.

See [Verdicts](docs/user/verdicts.md) for the full policy.

## Use in automation

JSON output and configurable exit codes let a script or CI job act on the result:

```sh
skill-verdict scan --config config.json --format json --fail-on review ./path/to/skills
```

`--fail-on review` exits with code `1` for any verdict other than `clean`. The default is `--fail-on block`, which exits with code `1` only for `block`. Command errors exit with code `2`. See [Usage](docs/user/usage.md) for output fields and all options.

## Privacy and limits

The enabled checks determine what the scanner sends over the network:

- Offline scans make no network requests.
- External checks send referenced names to lookup services. The scanner also requests domain roots to check availability and redirects. The scanner does not fetch linked documentation or download referenced code.
  - **CAUTION:** Requesting domain roots for mentioned links might expose your IP to owners of such domains. Make sure to disable the references check if this is important for you.
- Model checks send skill content and scan context to your configured model provider.

See [External references](docs/user/references.md) for the services the scanner contacts.

The scanner provides no sandbox for skill execution and does not inspect archive contents. The honeypot can miss attacks that wait for a particular task. Text rules and models can both miss problems or flag legitimate behavior. Review the evidence in the report.

External resources can change after a scan. Recheck skills before use. Limit your agent's permissions to the task's requirements.

## Documentation

| Page | What you will find |
|---|---|
| [Install](docs/user/install.md) | Installation, platform builds and scan requirements. |
| [Usage](docs/user/usage.md) | Commands, output fields and exit codes. |
| [Configuration](docs/user/config.md) | Model setup, scan limits and rule customization. |
| [Verdicts](docs/user/verdicts.md) | How findings and unfinished checks determine the result. |
| [Detection](docs/user/detection.md) | What the scanner reads and how the checks work. |
| [External references](docs/user/references.md) | Domain, GitHub and package checks and network behavior. |
| [Rules](docs/user/rules.md) | Every built-in rule and its default severity. |

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
