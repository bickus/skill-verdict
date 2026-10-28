# Built-in Rules Reference

`skill-verdict` includes 75 specialized security checks designed specifically for agent skills, instructions, and MCP tool configurations.

Rather than sorting rules alphabetically, this reference groups every rule by the **real-world threat** it defends against.

---

## Threat 1: Supply Chain & Moving Targets

### The Threat:
An agent skill can look completely harmless on the day you review it. But if it installs software from an unpinned source, clones a Git branch, or points to an external package, its behavior can change overnight.

Attackers can easily compromise unpinned software by:
- Hijacking developer accounts on npm or PyPI.
- Registering abandoned GitHub usernames to take over repository clone URLs.
- Registering deleted package names (dependency confusion).
- Typosquatting popular packages with near-identical names.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `UNPINNED-INSTALL` | `high`, dismissable | static | No | Install or execution command without an exact pinned version: `pip install pkg`, `npm i pkg`, `npx pkg`, `docker pull img`, `gem install`, `cargo install`, `brew tap` and a tap formula, `clawhub install` and `openclaw skills install` or `openclaw plugins install`. Also an install entry in the `SKILL.md` frontmatter metadata and an MCP server entry that runs `npx` or `uvx`. Runs every time the skill executes. The scanner skips a registry install of the skill's own folder name. The decision layer may dismiss a place where the user runs the command by hand, the command is only documentation, or the words after the command are not a package. The decision layer confirms a place where the agent runs the command. The Judge then leaves the finding alone. |
| `UNPINNED-DEPENDENCY` | `medium`, dismissable | static | No | Dependency declared in `package.json` or `requirements.txt` with a wildcard, range, or no version pin. |
| `UNPINNED-DOWNLOAD` | `critical`, floor `medium` | static | No | Downloading content from moving targets (e.g. raw, blob or archive URLs on `main` at GitHub, GitLab, SourceHut, Codeberg or Bitbucket, release downloads, or gists) where content can be replaced at the same address. Fixed commit hashes do not match. When a person does the download by hand, the Judge lowers the finding to the floor. |
| `UNPINNED-GIT` | `high`, floor `medium` | static | No | Cloning a Git repository (`git clone`, `gh repo clone`, `go install ...@branch`, `openclaw skills install git:...`) without a pinned commit hash. Every run pulls whatever the branch currently contains. The decision layer may dismiss a place that is only documentation or that checks out a commit hash. The decision layer confirms a place where the agent does the step. When a person does the clone by hand, the Judge lowers the finding to the floor. |
| `GITHUB-OWNER-MISSING` | `critical`, floor `medium` | references | No | The referenced GitHub owner account does not exist. Anyone can register the username and ship malware under the URL the skill trusts. |
| `GITHUB-OWNER-RENAMED` | `critical`, floor `medium` | references | No | The referenced repository redirects to a new account, leaving the original owner name available for anyone to claim. |
| `GITHUB-REPO-MISSING` | `high`, floor `medium` | references | No | The GitHub owner exists, but the repository has been deleted. The owner can recreate it with arbitrary code at any time. |
| `GITHUB-OWNER-NEW` | `medium`, dismissable | references | No | The referenced GitHub owner account was created within the last 90 days. New accounts have no reputation history. |
| `PACKAGE-MISSING` | `critical`, floor `medium` | references | No | The referenced npm or PyPI package is not published. Anyone can register the package name and immediately achieve code execution (dependency confusion). |
| `PACKAGE-TYPOSQUAT` | `high`, floor `low` | references | No | Package name is 1 or 2 typos away from a popular library on the same registry (e.g. `reqeusts` or `cross-env-colors`). |
| `PACKAGE-NEW` | `medium`, dismissable | references | No | The referenced package was published for the first time within the configured age threshold (default: 90 days). |
| `PACKAGE-LOW-DOWNLOADS` | `low`, dismissable | references | No | Package had fewer than 1,000 downloads last month. Unvetted code used by few people. |
| `PACKAGE-OTHER-REGISTRY` | `high`, dismissable | static | No | Install command specifies a third-party package index or custom registry (`--index-url`, `--registry`). |

---

## Threat 2: Infrastructure Takeover & Untrusted Hosting

### The Threat:
Skills often provide documentation links or API endpoints. If an attacker purchases an expired domain, hides malicious payloads behind URL shorteners, or points an agent to anonymous pastebins, they can dynamically serve exploit code directly into the agent's context.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `DOMAIN-UNREGISTERED` | `high`, floor `medium` | references | No | A referenced domain name is not registered. Anyone can purchase it right now and serve arbitrary content under the URL the skill trusts. |
| `DOMAIN-EXPIRING` | `high`, floor `medium` | references | No | A referenced domain expires within days. Once it lapses, domain drop-catchers or attackers can seize it. |
| `DOMAIN-SHORTENER` | `critical`, floor `medium` | references | No | The URL uses a link shortener (`bit.ly`, `tinyurl.com`, `t.co`). The actual destination is concealed and can be redirected at will. |
| `DOMAIN-PASTE` | `critical`, floor `medium` | references | No | The URL points to a paste or snippet service (`pastebin.com`, `gist.github.com`). Paste content can be edited at any time without version control. |
| `DOMAIN-FILEHOST` | `critical`, floor `medium` | references | No | The URL points to an anonymous file-sharing service (`mega.nz`, `dropbox.com`, `mediafire.com`). Uploaded files can be replaced by anyone without accountability. |
| `DOMAIN-NEW` | `medium`, dismissable | references | No | A referenced domain was registered within the configured threshold (default: 365 days). New domains have no established reputation. |
| `DOMAIN-NEW-REDIRECT` | `critical`, floor `medium` | references | No | A newly registered domain immediately redirects to a different host. Common pattern for delayed malware deployment. |
| `DOMAIN-UNREACHABLE` | `high`, dismissable | references | No | Domain cannot be reached (DNS failure, invalid TLS certificate, or connection refused). Could indicate deliberate concealment from scanners. |
| `DOMAIN-HTTP-ERROR` | `high`, dismissable | references | No | The domain root responds with an HTTP 4xx or 5xx status. |
| `IP-ADDRESS-PUBLIC` | `critical`, floor `medium` | references | No | A URL, connection command, or script targets a raw public IP address instead of a registered domain name. |
| `TOOL-SERVER-UNTRUSTED` | `critical`, floor `high` | references | No | The skill connects the agent to an MCP or other tool server, and the host has at least one reference fact finding. Raised by the `decision` layer from the usage of the reference. |
| `REFERENCES-TOO-MANY` | `high`, no downgrade | references | **Yes** | Skill contains more than 100 external references. Often used to overwhelm scanners or obscure malicious needles in haystacks. |

---

## Threat 3: Arbitrary Execution & Privilege Escalation

### The Threat:
An agent skill should perform repetitive tasks within clear boundaries. Instructions that pipe remote web scripts directly into `bash`, modify sudoers to grant passwordless root, disable package manager security checks, or execute hidden commands bypass all user oversight.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `REMOTE-SCRIPT-PIPED` | `critical`, floor `high` | static | No | A shell command fetches remote content and runs it at once: `curl ... \| bash`, `wget ... \| sh`, `curl ... \| python3 -`, `sh -c "$(curl ...)"`, `bash <(curl ...)`, `source <(curl ...)`, `eval "$(curl ...)"`. A pipe into `python3 -c` or `python3 -m` does not match, because the script does not come from the download. |
| `REMOTE-CODE-AT-INSTALL` | `high`, floor `low` | discovery | No | Instructions telling the agent to download and run a script, binary, package with no fixed version, registry skill or helper that is not bundled in the skill. The scanner reviews the bundle, not what the download contains. The decision layer may dismiss a place that runs only bundled code, a package a person installs by hand, a system package, a pinned version, a lock file entry or plain documentation. The decision layer confirms a place where the agent does the step. When a person does the download by hand, the Judge lowers the finding to the floor. |
| `COMMAND-ON-LOAD` | `high`, floor `medium` | static | No | Commands configured to execute automatically the moment a coding agent loads the skill file (e.g. Claude Code `!` command syntax). |
| `ROOT-PASSWORDLESS` | `critical`, floor `medium` | static | No | Commands that modify `/etc/sudoers` or grant passwordless root/administrator privileges. |
| `ROOT-PASSWORDLESS-HIDDEN` | `critical`, floor `medium` | discovery | No | Grants passwordless root/admin privileges using obscured or joined text, downloaded scripts, or natural language prompts. |
| `PACKAGE-PROTECTION-OFF` | `critical`, dismissable | static | No | Commands disabling package manager security protections (e.g. `--allow-unauthenticated`, `--ignore-scripts`, or disabling release age minimums). |
| `PACKAGE-PROTECTION-OFF-HIDDEN`| `critical`, floor `low` | discovery | No | Disabling package manager safeguards via obfuscated commands or instructions described in prose. |
| `PACKAGE-SOURCE-ADDED` | `critical`, floor `medium` | static | No | Adding a new package repository, PPA, Homebrew tap, or custom registry. Permanently alters how all future software is installed. |
| `PACKAGE-SOURCE-ADDED-HIDDEN` | `critical`, floor `medium` | discovery | No | Adding untrusted package sources out of sight via downloaded scripts or prose instructions. |
| `ENCODED-CODE-EXECUTED` | `high`, floor `medium` | static | No | Commands that decode base64, hex, or compressed payloads and pass them directly to an interpreter (`base64 -d \| sh`). |

---

## Threat 4: Snooping, Secret Theft & Data Exfiltration

### The Threat:
Coding agents typically operate in repositories containing environment variables, cloud keys, and SSH credentials. Malicious skills quietly search the filesystem for secrets, scrape MCP server configurations, query cloud instance metadata services, or upload data to remote buckets.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `SECRETS-ENV-DUMPED` | `high`, floor `low` | static | No | Code or commands copying, enumerating or searching the whole environment (`os.environ.items()`, `Object.keys(process.env)`, `env \| grep token`). |
| `SECRETS-ENV-USED` | `critical`, dismissable | static | No | A read of one environment variable named as a secret: `_TOKEN`, `_SECRET`, `_API_KEY`, `_PASSWORD`, `_CREDENTIALS`, `_CONNECTION_STRING`, `_PAT` and the fixed names `DATABASE_URL`, `SSH_AUTH_SOCK`, `SQLCONNSTR_*`. Covers Python, JavaScript, Go, shell and PowerShell read forms. The judge dismisses the skill's own declared variables. |
| `SECRETS-DIRS-LISTED` | `high`, floor `low` | static | No | Searching or listing files in sensitive directories such as `~/.ssh`, `~/.aws`, `~/.gnupg`, or root credentials. |
| `ENV-NAME-COMPUTED` | `critical`, floor `medium` | static | No | A read of one environment variable whose name is decoded, concatenated, reversed, formatted or otherwise built at run time, so the name never appears in the code. Covers Python, JavaScript, shell, Ruby, Go, PowerShell, Rust, PHP, Java and C# read forms. |
| `AGENT-CONFIG-READ` | `medium`, dismissable | static | No | Code or instructions reading the coding agent's own private configuration directories: `.claude`, `.codex`, `.gemini`, `.continue`, and `.openclaw`, `.clawdbot` or `.moltbot` with their config file and `.env`. |
| `AGENT-MCP-CONFIG-READ` | `medium`, dismissable | static | No | Reading Model Context Protocol (MCP) server configuration files, which frequently store API tokens and tool credentials. |
| `CLOUD-METADATA-READ` | `high`, floor `low` | static | No | Addressing cloud instance metadata services (`169.254.169.254`) to harvest temporary IAM instance credentials. |
| `REMOTE-ENDPOINT-UPLOAD` | `medium`, dismissable | static | No | Posting data to remote URLs, APIs, or telemetry collection endpoints. |
| `REMOTE-ENDPOINT-UPLOAD-HIDDEN` | `high`, dismissable | discovery | No | Code or instructions sending user input, files from the machine or data from another service to a referenced host, in wording the `REMOTE-ENDPOINT-UPLOAD` patterns miss. A service receiving only its own data is dismissed. |
| `CLOUD-STORAGE-UPLOAD` | `medium`, dismissable | static | No | Uploading files directly to remote Amazon S3, Google Cloud Storage, or Azure Blob containers. |
| `EXFILTRATION-IN-PROSE` | `high`, floor `low` | discovery | No | Natural-language instructions directing the agent to remember user inputs, log sensitive context, or echo secrets in replies. |

---

## Threat 5: Agent Manipulation & Prompt Injection

### The Threat:
Adversarial skills often manipulate the AI model's cognitive state rather than running shell exploits. By using gradual deception, role-play framing, or explicit system prompt overrides, they trick the agent into violating its safety boundaries.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `HONEYPOT-TRIGGERED` | `critical`, floor `medium` | honeypot | **Yes** | A simulated coding agent given only the instruction to "load this skill" immediately attempted an unauthorized tool call (bash, write, web). The skill provably hijacks agents on load. |
| `DECEPTION-GRADUAL` | `high`, floor `low` | discovery | No | Incremental instruction sequences that appear harmless step-by-step but progressively escalate permissions or normalize malicious acts. |
| `PROMPT-INJECTION-SEMANTIC` | `medium`, floor `low` | discovery | No | Subtle prompt injection: polite reframings of ignoring safety guidelines, hypothetical scenarios, or role-play setups granting elevated permissions. *(Disabled by default)*. |
| `PROMPT-INJECTION-PARAPHRASED`| `medium`, floor `low` | discovery | No | Reformulations of known jailbreak techniques using creative synonyms, cultural framing, or encoded wording. *(Disabled by default)*. |
| `SKILL-REFERENCED` | `high`, dismissable | static | No | A path or URL to another skill (`.../SKILL.md`, `.claude/skills/<name>`). The agent loads a file the scanner never read. Kept only when this skill also prepares the ground for it, such as relaxing safety rules or granting tools the referenced skill can use. The decision layer may dismiss it when the lines only name the skill's own files or mention another skill as information. |
| `SKILL-REFERENCED-HIDDEN` | `high`, dismissable | discovery | No | An instruction to load or follow another skill in wording without a path, together with preparation the unseen skill can act on. |
| `REMOTE-INSTRUCTIONS-FOLLOWED` | `high`, dismissable | discovery | No | Instructions telling the agent to open a URL and follow what it finds there (a prompt, rules, a config, a list of steps). The scanner never reads that page, so the reviewed skill is not what actually runs. Plain documentation lookups are dismissed. |
| `INSTRUCTIONS-FORWARDED` | `high`, dismissable | discovery | No | The skill tells the agent to put instructions into what it hands to other agents, sessions or skills, so the instructions spread past the agent that installed the skill. Passing on data, the user's own request, or the prompt of a subagent the task starts is dismissed. |

---

## Threat 6: Evasion, Obfuscation & Concealed Payloads

### The Threat:
To evade basic keyword searches, regex filters, and human audits, malicious authors conceal text using zero-width characters, lookalike Cyrillic characters, or password-protected archives.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `WORD-INVISIBLE-CHARS` | `high`, floor `low` | reveal | No | Non-printing zero-width characters or directional formatting inserted inside words to bypass regex scanners (e.g. `c\u200Burl`). |
| `WORD-HOMOGLYPHS` | `medium`, dismissable | reveal | No | A word that mixes Latin letters with letters of another script that print like ASCII letters (e.g. Cyrillic `а` or Greek `ο`). The decision layer may dismiss it after reading the lines around it. |
| `LINK-IN-COMMENT` | `high`, floor `medium` | static | No | A URL inside an HTML comment in a Markdown file, outside code blocks. The rendered page hides it from the person who reads the skill, and the agent reads it as plain text. |
| `ARCHIVE-ENCRYPTED` | `critical`, no downgrade | walk | **Yes** | The skill bundle ships an encrypted/password-protected archive. The scanner cannot inspect its contents, and shipping encrypted files in a skill bundle is inherently dangerous. |
| `ARCHIVE-PASSWORD-GIVEN` | `critical`, floor `medium` | discovery | No | Instructions providing a password or extraction code to open an external or bundled archive. Bypasses file scanners. |
| `ARCHIVE-RAR` | `high`, floor `medium` | walk | No | The skill bundle ships a `.rar` archive, which requires third-party extractors and cannot be inspected as text. As a bonus - `.rar` files are rarely used in "normal operations" but are loved by malicious groups around the world in certain countries. |
| `ARCHIVE-RAR-MENTIONED` | `medium`, floor `low` | static | No | The skill instructions tell the user or agent to download and unpack a `.rar` file. |

---

## Threat 7: Opaque Bundles, Binary Blobs & Symlink Traps

### The Threat:
Agent skills should be composed of readable documentation, prompts, and source code. Shipping compiled binary programs, native installers, or disguised PDFs prevents human and automated review. Similarly, malicious symlinks can trick an agent into reading files outside the skill directory.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `FILE-BINARY` | `high`, no downgrade | walk | **Yes** | The skill ships compiled machine code or bytecode (ELF, Windows PE, Mach-O, Java `.class`, Python `.pyc`, WebAssembly). Unreviewable without reverse-engineering. |
| `FILE-INSTALLER` | `high`, no downgrade | walk | **Yes** | The skill ships system packages or installers (`.msi`, `.pkg`, `.deb`, `.rpm`). |
| `FILE-PDF` | `high`, no downgrade | walk | **Yes** | The skill ships a PDF file. PDFs cannot be inspected as plain text and can embed malicious scripts or invisible instructions. |
| `FILE-TOO-LARGE` | `high`, no downgrade | walk | **Yes** | A single file exceeds `layers.walk.maxFileBytes` (default: 5 MB). |
| `BUNDLE-TOO-LARGE` | `high`, no downgrade | walk | **Yes** | Cumulative skill size exceeds `layers.walk.maxBundleBytes` (default: 64 MB). |
| `BUNDLE-TOO-MUCH-TEXT` | `high`, no downgrade | walk | **Yes** | Cumulative text files exceed `layers.walk.maxTextBytes` (default: 400 KB). |
| `BUNDLE-TOO-MANY-FILES` | `high`, no downgrade | walk | **Yes** | Total file count exceeds `layers.walk.maxFiles` (default: 50 files). |
| `BUNDLE-TOO-DEEP` | `high`, no downgrade | walk | **Yes** | Directory nesting depth exceeds `layers.walk.maxDepth` (default: 5 subdirectories). |
| `SYMLINK-OUT-OF-SKILL` | `high`, dismissable | static | No | A symbolic link points to a target outside the skill directory. |
| `SYMLINK-TO-SECRETS` | `critical`, floor `medium` | static | No | A symbolic link points directly to sensitive system files (`.env`, `.ssh`, `/etc`, or home directories). |

---

## Threat 8: Insecure Configurations & Dangerous Deserialization

### The Threat:
Skills that configure infrastructure or frameworks can silently weaken your security posture by disabling TLS certificate checks, running containers in privileged mode, or deserializing untrusted objects.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `SETTING-UNSAFE` | `medium`, dismissable | static | No | Code or settings that disable TLS validation (`verify=False`), bypass authentication, disable sandboxing, or enable permissive debug modes. |
| `CONTAINER-TRUST-DISABLED` | `medium`, dismissable | static | No | Pulling container images with Content Trust disabled or using insecure, unauthenticated registries. |
| `KUBERNETES-PRIVILEGED` | `medium`, dismissable | static | No | Kubernetes pod definitions that run privileged containers, mount host filesystems, or share the host network namespace. |
| `DESERIALIZE-PHP` | `high`, floor `low` | static | No | Calling PHP `unserialize()` on untrusted input, which can trigger object injection and remote code execution. |
| `DESERIALIZE-RUBY-MARSHAL` | `high`, floor `low` | static | No | Calling Ruby `Marshal.load()` on untrusted data. |
| `DESERIALIZE-RUBY-YAML` | `medium`, dismissable | static | No | Calling unsafe YAML loaders (`YAML.load`, `Psych.load`) that instantiate arbitrary Ruby objects. |
