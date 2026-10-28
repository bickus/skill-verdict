# Built-in Rules Reference

`skill-verdict` has 78 security checks for agent skills, their instructions and MCP tool configurations.

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
| `UNPINNED-INSTALL` | `high`, dismissable | static | No | Install or execution command without an exact pinned version: `pip install pkg`, `npm i pkg`, `npx pkg`, `docker pull img`, `gem install`, `cargo install`, `brew tap` and a tap formula, `clawhub install` and `openclaw skills install` or `openclaw plugins install`. Also an install entry in the `SKILL.md` frontmatter metadata and an MCP server entry that runs `npx` or `uvx`. Runs every time the skill executes. The scanner skips a registry install of the skill's own folder name. The decision layer confirms a place where the agent runs the command or the agent platform applies the entry. The layer dismisses a place where a person runs the command by hand or where the command is only documentation. The layer also dismisses words after the command that are not a package, and a command with an exact version or a digest. |
| `UNPINNED-DEPENDENCY` | `medium`, no downgrade | static | No | Dependency declared in `package.json` or `requirements.txt` with a wildcard, range, or no version pin. The finding stays only when another file of the skill names the manifest or runs an installer of the same ecosystem, such as `pip install -r` for `requirements.txt` or `npm ci` for `package.json`. |
| `UNPINNED-DOWNLOAD` | `critical`, floor `medium` | static | No | Downloading content from moving targets (e.g. raw, blob or archive URLs on `main` at GitHub, GitLab, SourceHut, Codeberg or Bitbucket, release downloads, or gists) where content can be replaced at the same address. Fixed commit hashes do not match. The decision layer confirms a URL that the agent fetches, installs from, runs, follows or sends data to. A download that a person does by hand, a citation and documentation go to the floor. |
| `UNPINNED-GIT` | `high`, floor `medium` | static | No | Cloning a Git repository (`git clone`, `gh repo clone`, `go install ...@branch`, `openclaw skills install git:...`) without a pinned commit hash. Every run pulls whatever the branch currently contains. The decision layer confirms a clone that the agent does from a repository the skill names. A clone that a person does by hand and documentation go to the floor. The layer dismisses a match that is not a clone, a clone pinned to a commit hash and a repository that the user names. |
| `GITHUB-OWNER-MISSING` | `critical`, floor `medium` | references | No | The referenced GitHub owner account does not exist. Anyone can register the username and ship malware under the URL the skill trusts. |
| `GITHUB-REPO-MOVED` | `low`, dismissable | references | No | The repository is now under another owner and GitHub redirects the old address there. Only the old owner can take the old address back, and the skill already trusts that owner. |
| `GITHUB-OWNER-REPLACED` | `critical`, floor `medium` | references | No | The account that holds the referenced owner name is younger than the repository and than the account the repository moved to. Someone registered the old name after a rename, and GitHub still redirects the old address. |
| `GITHUB-REPO-MISSING` | `medium`, dismissable | references | No | The GitHub owner has no repository under that name. A deleted repository and an owner name registered again by a stranger look the same to the scanner. |
| `GITHUB-OWNER-NEW` | `medium`, dismissable | references | No | The referenced GitHub owner account was created within the last 90 days. New accounts have no reputation history. |
| `PACKAGE-MISSING` | `critical`, floor `medium` | references | No | The referenced npm or PyPI package is not published. Anyone can register the package name and immediately achieve code execution (dependency confusion). The decision layer confirms a place where the agent or the agent platform installs the package. An install by a person and documentation go to the floor. The layer dismisses a placeholder such as `your-package`. |
| `PACKAGE-TYPOSQUAT` | `high`, floor `low` | references | No | Package name is 1 or 2 typos away from a popular library on the same registry (e.g. `reqeusts` or `cross-env-colors`). The decision layer confirms a place where the agent or the agent platform installs the package. An install by a person goes to `medium`, documentation to the floor. The layer dismisses a placeholder. |
| `PACKAGE-NEW` | `medium`, dismissable | references | No | The referenced package was published for the first time within the configured age threshold (default: 90 days). The decision layer confirms a place where the agent or the agent platform installs the package. The layer dismisses an install by a person, documentation and a placeholder. |
| `PACKAGE-LOW-DOWNLOADS` | `low`, dismissable | references | No | Package had fewer than 1,000 downloads last month. Unvetted code used by few people. The decision layer confirms a place where the agent or the agent platform installs the package. The layer dismisses an install by a person, documentation and a placeholder. |
| `PACKAGE-OTHER-REGISTRY` | `high`, floor `medium` | static | No | Install command specifies a third-party package index or custom registry (`--index-url`, `--registry`). The decision layer confirms an install by the agent from a registry that neither the vendor of the package nor the user runs. A registry of the vendor, a mirror that the user configures, an install by a person and documentation go to the floor. |

---

## Threat 2: Infrastructure Takeover & Untrusted Hosting

### The Threat:
Skills often provide documentation links or API endpoints. If an attacker purchases an expired domain, hides malicious payloads behind URL shorteners, or points an agent to anonymous pastebins, they can dynamically serve exploit code directly into the agent's context.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `DOMAIN-UNREGISTERED` | `high`, floor `medium` | references | No | A referenced domain name is not registered. Anyone can purchase it right now and serve arbitrary content under the URL the skill trusts. |
| `DOMAIN-EXPIRING` | `high`, floor `medium` | references | No | A referenced domain expires within days. Once it lapses, domain drop-catchers or attackers can seize it. |
| `PLATFORM-SITE-MISSING` | `critical`, floor `medium` | references | No | A host on a hosting platform answers that no site, app or bucket exists under its name. Anyone with an account on the platform can create one and serve content under the URL the skill trusts. The scanner trusts only the platform's own answer: its page for a missing site, or a missing DNS record. The scanner knows the answers of `s3.amazonaws.com`, `surge.sh`, `azurewebsites.net`, `fly.dev`, `pages.dev` and `workers.dev`. |
| `DOMAIN-SHORTENER` | `critical`, floor `medium` | references | No | The URL uses a link shortener (`bit.ly`, `tinyurl.com`, `t.co`). The actual destination is concealed and can be redirected at will. The decision layer decides the finding from what the skill does with the URL, see [External references](references.md#what-the-skill-does-with-a-reference). |
| `DOMAIN-PASTE` | `critical`, floor `medium` | references | No | The URL points to a paste or snippet service (`pastebin.com`, `gist.github.com`). Paste content can be edited at any time without version control. The decision layer decides the finding from what the skill does with the URL. |
| `DOMAIN-FILEHOST` | `critical`, floor `medium` | references | No | The URL points to an anonymous file-sharing service (`mega.nz`, `dropbox.com`, `mediafire.com`). Uploaded files can be replaced by anyone without accountability. The decision layer decides the finding from what the skill does with the URL. |
| `DOMAIN-NEW` | `medium`, dismissable | references | No | A referenced domain was registered within the configured threshold (default: 365 days). New domains have no established reputation. |
| `DOMAIN-NEW-REDIRECT` | `critical`, floor `medium` | references | No | A newly registered domain immediately redirects to a different host. Common pattern for delayed malware deployment. The decision layer decides the finding from what the skill does with the URL. |
| `DOMAIN-UNREACHABLE` | `low`, dismissable | references | No | The scanner cannot connect to the domain. The owner of the domain controls its DNS, its TLS and its server. |
| `DOMAIN-HTTP-ERROR` | `low`, dismissable | references | No | The domain root responds with an HTTP 4xx or 5xx status. API hosts, content networks and hosts that serve only paths answer this way. Only the owner of the domain can change the answer. |
| `IP-ADDRESS-PUBLIC` | `critical`, floor `medium` | references | No | A URL, connection command, or script targets a raw public IP address instead of a registered domain name. The public DNS resolvers of Google, Cloudflare, Quad9 and OpenDNS give no finding. The decision layer decides the finding from what the skill does with the address. |
| `TOOL-SERVER-UNTRUSTED` | `critical`, no downgrade | references | No | The skill connects the agent to an MCP or other tool server, and the host has at least one reference fact finding above `low`. Raised by the `decision` layer from the usage of the reference. |
| `REFERENCES-TOO-MANY` | `high`, no downgrade | references | **Yes** | Skill contains more than 100 external references. Often used to overwhelm scanners or obscure malicious needles in haystacks. |

---

## Threat 3: Arbitrary Execution & Privilege Escalation

### The Threat:
An agent skill should perform repetitive tasks within clear boundaries. Instructions that pipe remote web scripts directly into `bash`, modify sudoers to grant passwordless root, disable package manager security checks, or execute hidden commands bypass all user oversight.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `REMOTE-SCRIPT-PIPED` | `critical`, floor `high` | static | No | A shell command fetches remote content and runs it at once: `curl ... \| bash`, `wget ... \| sh`, `curl ... \| python3 -`, `sh -c "$(curl ...)"`, `bash <(curl ...)`, `source <(curl ...)`, `eval "$(curl ...)"`. A pipe into `python3 -c` or `python3 -m` does not match, because the script does not come from the download. The decision layer confirms a command that the agent runs. A command that a person runs by hand goes to the floor. The layer dismisses a mention in documentation or a changelog. |
| `REMOTE-CODE-AT-INSTALL` | `high`, floor `low` | discovery | No | Instructions telling the agent to download and run a script, binary, package with no fixed version, registry skill or helper that is not bundled in the skill. The scanner reviews the bundle, not what the download contains. The discovery model records the step only when the agent or a person is told to do it. The decision layer dismisses a place that runs only bundled code. The layer confirms a place where the agent does the step. Most other answers lower the finding to the floor, see [Detection](detection.md#layer-6-decision). |
| `COMMAND-ON-LOAD` | `high`, no downgrade | static | No | Claude Code runs `` !`command` `` and every fenced block opened with ```` ```! ```` in `SKILL.md` and in command files when it loads the file. A read-only command such as `git status` or `ls` is `medium`. The same syntax in any other file runs nothing and gives no finding. |
| `ROOT-PASSWORDLESS` | `critical`, floor `medium` | static | No | Commands that modify `/etc/sudoers` or grant passwordless root/administrator privileges. The decision layer confirms a grant. A removal, a check and an example of what not to do go to the floor. |
| `ROOT-PASSWORDLESS-HIDDEN` | `critical`, floor `medium` | discovery | No | Grants passwordless root/admin privileges using obscured or joined text, downloaded scripts, or natural language prompts. The discovery model records it only when the agent or a person is told to do it. |
| `PACKAGE-PROTECTION-OFF` | `critical`, dismissable | static | No | Commands disabling package manager security protections (e.g. `--allow-unauthenticated`, `--ignore-scripts`, or disabling release age minimums). The decision layer confirms a protection that the agent, the agent platform or a person turns off for every package. An allow list of packages whose scripts may run goes to `medium`. The layer dismisses a setting that turns a protection on, an explanation and documentation. |
| `PACKAGE-PROTECTION-OFF-HIDDEN`| `critical`, floor `low` | discovery | No | Disabling package manager safeguards via obfuscated commands or instructions described in prose. The discovery model records it only when the agent or a person is told to do it. |
| `PACKAGE-SOURCE-ADDED` | `critical`, floor `medium` | static | No | Adding a new package repository, PPA, Homebrew tap, or custom registry. Permanently alters how all future software is installed. The decision layer confirms a source of an unknown owner that the agent or the agent platform adds. The same step by a person goes to `medium`. A source of the OS vendor or of the package vendor, and documentation, go to the floor. |
| `PACKAGE-SOURCE-ADDED-HIDDEN` | `critical`, floor `medium` | discovery | No | Adding untrusted package sources out of sight via downloaded scripts or prose instructions. The discovery model records it only when the agent or a person is told to do it. |
| `ENCODED-CODE-EXECUTED` | `high`, floor `medium` | static | No | Commands that decode base64, hex, or compressed payloads and pass them directly to an interpreter (`base64 -d \| sh`). The decision layer confirms decoded code that runs and code that hides what it does. Decoded data and a literal that nothing decodes go to the floor. |

---

## Threat 4: Snooping, Secret Theft & Data Exfiltration

### The Threat:
Coding agents typically operate in repositories containing environment variables, cloud keys, and SSH credentials. Malicious skills quietly search the filesystem for secrets, scrape MCP server configurations, query cloud instance metadata services, or upload data to remote buckets.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `SECRETS-ENV-DUMPED` | `high`, floor `low` | static | No | Code or commands copying, enumerating or searching the whole environment (`os.environ.items()`, `Object.keys(process.env)`, `env \| grep token`). The decision layer confirms code that lists, prints, sends or writes all variables. Handing the environment to a child process goes to `medium`. The layer dismisses a read of named variables. |
| `SECRETS-ENV-USED` | `critical`, dismissable | static | No | A read of one environment variable named as a secret: `_TOKEN`, `_SECRET`, `_API_KEY`, `_PASSWORD`, `_CREDENTIALS`, `_CONNECTION_STRING`, `_PAT` and the fixed names `DATABASE_URL`, `SSH_AUTH_SOCK`, `SQLCONNSTR_*`. Covers Python, JavaScript, Go, shell and PowerShell read forms. The decision layer confirms a value that goes to another host, a file or a log. The layer dismisses the skill's own credential that goes only to its service or nowhere, and a placeholder. A credential of another product that goes nowhere visible keeps the severity. |
| `SECRETS-DIRS-LISTED` | `high`, floor `low` | static | No | Searching or listing files in sensitive directories such as `~/.ssh`, `~/.aws`, `~/.gnupg`, or root credentials. The decision layer confirms a listing of the home or of a credential folder. The skill's own folder and a path that the user gives go to the floor. |
| `ENV-NAME-COMPUTED` | `critical`, floor `medium` | static | No | A read of one environment variable whose name is decoded, concatenated, reversed, formatted or otherwise built at run time, so the name never appears in the code. Covers Python, JavaScript, shell, Ruby, Go, PowerShell, Rust, PHP, Java and C# read forms. The decision layer confirms a built name that holds a secret or that the code does not explain. A harmless name with a plain reason goes to the floor. |
| `AGENT-CONFIG-READ` | `medium`, dismissable | static | No | Code or instructions reading the coding agent's own private configuration directories: `.claude`, `.codex`, `.gemini`, `.continue`, and `.openclaw`, `.clawdbot` or `.moltbot` with their config file and `.env`. The decision layer confirms a read whose result is sent or logged, and a read by a skill that is not for managing agent settings. The layer dismisses a read that stays on the machine in a skill for managing those settings. |
| `AGENT-MCP-CONFIG-READ` | `medium`, dismissable | static | No | Reading Model Context Protocol (MCP) server configuration files, which frequently store API tokens and tool credentials. The decision layer decides it as `AGENT-CONFIG-READ`, and dismisses prose that reads no file. |
| `CLOUD-METADATA-READ` | `high`, floor `low` | static | No | Addressing cloud instance metadata services (`169.254.169.254`) to harvest temporary IAM instance credentials. The decision layer confirms a read of the credential paths. Instance details and an example in a file for people go to the floor. |
| `REMOTE-ENDPOINT-UPLOAD` | `medium`, dismissable | static | No | Posting data to remote URLs, APIs, or telemetry collection endpoints. |
| `REMOTE-ENDPOINT-UPLOAD-HIDDEN` | `high`, dismissable | discovery | No | Code or instructions sending user input, files from the machine or data from another service to a referenced host, in wording the `REMOTE-ENDPOINT-UPLOAD` patterns miss. The discovery model records it only when data from the machine or from another service goes to a host that is not the service the skill is for. |
| `CLOUD-STORAGE-UPLOAD` | `medium`, dismissable | static | No | Uploading files directly to remote Amazon S3, Google Cloud Storage, or Azure Blob containers. The decision layer confirms an upload to a bucket that the skill names, and an upload of data that the user did not choose. The layer dismisses an upload of the task output or of files the user chose to a bucket the user configures. |
| `EXFILTRATION-IN-PROSE` | `high`, floor `low` | discovery | No | Natural-language instructions directing the agent to send user inputs, session content or secrets to an outside service, or to write them to a file that another agent or session reads. Data that stays in the reply, in the conversation or in a local file of the skill is not recorded. |

---

## Threat 5: Agent Manipulation & Prompt Injection

### The Threat:
Adversarial skills often manipulate the AI model's cognitive state rather than running shell exploits. By using role-play framing or explicit system prompt overrides, they trick the agent into violating its safety boundaries.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `HONEYPOT-TRIGGERED` | `critical`, floor `medium` | honeypot | **Yes** | A simulated coding agent given only the instruction to "load this skill" immediately attempted an unauthorized tool call (bash, write, web). The skill provably hijacks agents on load. |
| `PROMPT-INJECTION-SEMANTIC` | `medium`, floor `low` | discovery | No | Subtle prompt injection: polite reframings of ignoring safety guidelines, hypothetical scenarios, or role-play setups granting elevated permissions. *(Disabled by default)*. |
| `PROMPT-INJECTION-PARAPHRASED`| `medium`, floor `low` | discovery | No | Reformulations of known jailbreak techniques using creative synonyms, cultural framing, or encoded wording. *(Disabled by default)*. |
| `SKILL-REFERENCED` | `high`, dismissable | static | No | A path or URL to another skill (`.../SKILL.md`, `.claude/skills/<name>`). The agent loads a file the scanner never read. The decision layer dismisses the finding when the text only names the skill's own files or mentions another skill as information. The layer confirms it when the agent changes the files of the other skill. The layer also confirms it when the agent loads the other skill and this skill prepares something for it, such as relaxing safety rules or granting tools. When this skill prepares nothing, loading the other skill lowers the finding to `low`. |
| `SKILL-REFERENCED-HIDDEN` | `high`, dismissable | discovery | No | An instruction to load or follow another skill in wording without a path, together with preparation the unseen skill can act on. The discovery model records it only when this skill relaxes safety rules, grants tools or writes files or settings for the other skill. |
| `REMOTE-INSTRUCTIONS-FOLLOWED` | `high`, dismissable | discovery | No | Instructions telling the agent to open a URL and follow what it finds there (a prompt, rules, a config, a list of steps). The scanner never reads that page, so the reviewed skill is not what actually runs. The discovery model records it only for a page the skill names. A page the user gives and the content the task works on are not recorded. |
| `INSTRUCTIONS-FORWARDED` | `high`, dismissable | discovery | No | The skill tells the agent to put instructions into what it hands to other agents, sessions or skills, so the instructions spread past the agent that installed the skill. The discovery model records it only when the instructions go to another agent, another session or another skill. Instructions for a subagent the task starts or for the user are not recorded. |

---

## Threat 6: Evasion, Obfuscation & Concealed Payloads

### The Threat:
To evade basic keyword searches, regex filters, and human audits, malicious authors conceal text using zero-width characters, lookalike Cyrillic characters, or password-protected archives.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `WORD-INVISIBLE-CHARS` | `high`, floor `low` | reveal | No | Non-printing zero-width characters or directional formatting inserted inside words to bypass regex scanners (e.g. `c\u200Burl`). The decision layer confirms a disguised word. A notation, text in a language that uses those characters and test data go to the floor. |
| `WORD-HOMOGLYPHS` | `medium`, dismissable | reveal | No | A word that mixes Latin letters with letters of another script that print like ASCII letters (e.g. Cyrillic `а` or Greek `ο`). The decision layer dismisses the finding for a notation, for text in a language that uses the other script and for test data. The layer confirms a disguised word. |
| `LINK-IN-COMMENT` | `high`, floor `medium` | static | No | A URL inside an HTML comment in a Markdown file, outside code blocks. The rendered page hides it from the person who reads the skill, and the agent reads it as plain text. The decision layer confirms a comment that tells the agent to follow the URL or that holds text for the agent. A note for the author, a credit or a license goes to the floor. |
| `ARCHIVE-ENCRYPTED` | `critical`, no downgrade | walk | **Yes** | The skill bundle ships an encrypted/password-protected archive. The scanner cannot inspect its contents, and shipping encrypted files in a skill bundle is inherently dangerous. |
| `ARCHIVE-PASSWORD-GIVEN` | `critical`, floor `medium` | discovery | No | Instructions providing a password or extraction code to open an external or bundled archive. Bypasses file scanners. The discovery model records it only when the agent or a person is told to use the password. |
| `ARCHIVE-RAR` | `high`, no downgrade | walk | No | The skill bundle ships a `.rar` archive, which requires third-party extractors and cannot be inspected as text. As a bonus - `.rar` files are rarely used in "normal operations" but are loved by malicious groups around the world in certain countries. |
| `ARCHIVE-RAR-MENTIONED` | `medium`, floor `low` | static | No | The skill names a `.rar` file. The decision layer confirms a RAR file that the skill downloads or tells someone to download. A file that the user provides and a list of supported formats go to the floor. |

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
| `SYMLINK-OUT-OF-SKILL` | `high`, no downgrade | static | No | A symbolic link points to a target outside the skill directory. |
| `SYMLINK-TO-SECRETS` | `critical`, no downgrade | static | No | A symbolic link points directly to sensitive system files (`.env`, `.ssh`, `/etc`, or home directories). |

---

## Threat 8: Insecure Configurations & Dangerous Deserialization

### The Threat:
Skills that configure infrastructure or frameworks can silently weaken your security posture by disabling TLS certificate checks, running containers in privileged mode, or deserializing untrusted objects.

### Rules:

| Rule ID | Severity & Judge Policy | Layer | Interrupt? | What It Detects |
|---|---|:---:|:---:|---|
| `SETTING-UNSAFE` | `medium`, dismissable | static | No | Code or settings that disable TLS validation (`verify=False`), bypass authentication, disable sandboxing, or enable permissive debug modes. The decision layer confirms a setting that the skill applies or tells the user to apply. The layer dismisses a test fixture and an example that nobody is told to apply. |
| `CONTAINER-TRUST-DISABLED` | `medium`, dismissable | static | No | Pulling container images from a registry over plain HTTP or with an unknown certificate authority, or turning off image signature checks that the user turned on. The decision layer confirms a step of the agent, the agent platform or a person. The layer dismisses documentation. |
| `KUBERNETES-PRIVILEGED` | `medium`, dismissable | static | No | Kubernetes pod definitions that run privileged containers, mount host filesystems, or share the host network namespace. The decision layer decides it as `SETTING-UNSAFE`. |
| `DESERIALIZE-PHP` | `high`, floor `low` | static | No | Calling PHP `unserialize()` on untrusted input, which can trigger object injection and remote code execution. The decision layer confirms data from outside the skill. Data of the skill itself and a safe variant of the call go to the floor. |
| `DESERIALIZE-RUBY-MARSHAL` | `high`, floor `low` | static | No | Calling Ruby `Marshal.load()` on untrusted data. The decision layer confirms data from outside the skill. Data of the skill itself and a safe variant of the call go to the floor. |
| `DESERIALIZE-RUBY-YAML` | `medium`, dismissable | static | No | Calling unsafe YAML loaders (`YAML.load`, `Psych.load`) that instantiate arbitrary Ruby objects. The decision layer confirms data from outside the skill. The layer dismisses data of the skill itself and a safe variant of the call. |
