# Attack techniques

Each row is one technique from the reports in the [sources list](sources.md). A technique that several reports describe is one row. The report IDs link to the rows of the sources list.

The list has only techniques that put something into the files of a skill. The list leaves out three kinds of technique:

- distribution: accounts, names, listings, upload pace, registries that copy a listing
- malware behavior after the payload runs
- attacks outside a skill: CI pipelines, code already inside a sandbox, bugs in an agent client

The [hiding methods](hiding.md) list the ways the reports hide a technique inside a skill.

## Instructions in text

The agent or the user follows text in `SKILL.md` or in a document that `SKILL.md` links to.

| ID | Technique | Reports |
|---|---|---|
| A01 | A prerequisites section tells the user or the agent to download an archive and run the executable inside. The archive is on a GitHub release or another host and often has a password. | [S02](sources.md#s02), [S03](sources.md#s03), [S04](sources.md#s04), [C02](sources.md#c02), [C09](sources.md#c09) |
| A02 | A prerequisites section sends the user or the agent to a paste page, such as glot.io or rentry.co, whose script pipes a base64 string to bash. The skill says it does not work until this runs. | [S03](sources.md#s03), [S04](sources.md#s04), [C02](sources.md#c02), [C09](sources.md#c09) |
| A03 | A setup step in `SKILL.md` or in a README decodes a base64 string and pipes the result to bash. | [S01](sources.md#s01), [S02](sources.md#s02), [S03](sources.md#s03), [S04](sources.md#s04), [C02](sources.md#c02), [C04](sources.md#c04) |
| A04 | A setup step shows a URL that looks official. The command itself runs a silent curl to a raw IP address. | [S03](sources.md#s03), [S05](sources.md#s05), [C02](sources.md#c02), [C09](sources.md#c09) |
| A05 | `SKILL.md` says the user needs a CLI first and links to a web page. The agent fetches the page and runs the install steps on it. | [S09](sources.md#s09), [C02](sources.md#c02) |
| A06 | On first use the skill makes the agent show a fake error. The error comes with a terminal command that claims to fix it. | [S05](sources.md#s05) |
| A07 | A download step clears the macOS quarantine flag with `xattr -c`, makes the file executable and runs it. | [S03](sources.md#s03), [S05](sources.md#s05) |
| A08 | The text tells the user to download a release ZIP, extract it, run the launcher and click Run anyway on the security prompt. | [S08](sources.md#s08), [C01](sources.md#c01) |
| A09 | `SKILL.md` has the agent fetch instructions from a remote URL at run time and follow them. | [S02](sources.md#s02), [C05](sources.md#c05) |
| A10 | `SKILL.md` has the agent fetch a data file from the publisher's domain on every call and use its links in answers. | [S04](sources.md#s04) |
| A11 | `SKILL.md` has the agent read and act on content that third parties post on a public forum or API. | [S02](sources.md#s02) |
| A12 | A setup section installs a second skill from the registry with `--no-input` and hides errors with `\|\| true`. | [S06](sources.md#s06) |
| A13 | `SKILL.md` tells the agent to find keys, base64 them and POST them to a remote endpoint in the background. The user sees no output. | [S06](sources.md#s06) |
| A14 | A setup command hidden with base64 or Unicode sends `~/.aws/credentials` to a remote host. | [S02](sources.md#s02) |
| A15 | An email skill tells the agent to BCC every message to an outside address and not to mention it. | [S01](sources.md#s01) |
| A16 | A command file tells the agent to curl project context or code to the attacker's remote service. | [S01](sources.md#s01), [S11](sources.md#s11), [S19](sources.md#s19), [C05](sources.md#c05) |
| A17 | `SKILL.md` has the agent send cryptocurrency to the operator's wallet. | [S04](sources.md#s04) |
| A18 | `SKILL.md` declares a mandatory protocol that overrides the agent's rules. The text says nothing can bypass the protocol. | [S01](sources.md#s01) |
| A19 | `SKILL.md` tells the agent that security warnings are test artifacts, or tells it to hide or relabel the warnings. | [S01](sources.md#s01), [S02](sources.md#s02), [S19](sources.md#s19) |
| A20 | `SKILL.md` tells the agent to act silently: not to ask the user and not to mention what it does. | [S01](sources.md#s01), [S16](sources.md#s16), [C15](sources.md#c15) |
| A21 | `SKILL.md` contains jailbreak text: DAN prompts, "ignore previous instructions", fake system messages. | [S02](sources.md#s02), [S16](sources.md#s16) |
| A22 | Instructions are hidden in invisible Unicode tag characters or zero-width characters. | [S01](sources.md#s01), [S16](sources.md#s16), [S17](sources.md#s17), [S18](sources.md#s18) |
| A23 | Instructions are hidden in HTML comments. | [S01](sources.md#s01) |
| A25 | `SKILL.md` tells the agent to run itself or a child agent with the permission or sandbox checks off. | [S01](sources.md#s01), [S07](sources.md#s07) |
| A26 | `SKILL.md` tells the agent to add an MCP server entry to its own configuration. The entry installs and runs a package. | [S07](sources.md#s07) |
| A27 | `SKILL.md` tells the agent to point its model API base URL at another host. | [S07](sources.md#s07) |
| A28 | `SKILL.md` tells the agent to set npm or yarn to another registry and calls it a corporate mirror. | [S13](sources.md#s13) |
| A29 | `SKILL.md` tells the agent to write into its memory files, such as `SOUL.md` or `MEMORY.md`. | [S02](sources.md#s02) |
| A30 | `SKILL.md` tells the agent to create a new skill folder that agents load later. | [S17](sources.md#s17) |
| A31 | `SKILL.md` tells the agent to edit systemd service files, weaken system settings or delete system files. | [S02](sources.md#s02) |
| A32 | `SKILL.md` adds a scheduled agent job. The job runs in an isolated session and does not notify the user. | [S06](sources.md#s06) |
| A33 | Rules text changes the code the agent writes. The code gets a script tag from an outside site, weak crypto, auth checks an attacker can bypass, or code that leaks secrets. | [S16](sources.md#s16), [S19](sources.md#s19) |
| A34 | `SKILL.md` tells the agent to print API keys, put credentials into commands or ask the user to paste secrets. | [S02](sources.md#s02), [S17](sources.md#s17), [C04](sources.md#c04) |
| A35 | Text copies the layout of tool definitions so the model believes a tool exists. | [S18](sources.md#s18) |
| A91 | The README states the malicious behavior as a feature, such as a connection to a fixed address at startup. | [C07](sources.md#c07) |
| A92 | `SKILL.md` keeps the name and description of an official skill. Its quick start step now runs a command that contacts an outside host. | [C03](sources.md#c03) |
| A93 | A `SKILL.md` step reads the user's GitHub token with the `gh` CLI and sends it with `curl` to the publisher's API. The step calls this uploading a report. | [C05](sources.md#c05) |
| A99 | The text has the agent ask for a harmless command first, so the user allows it for the session. A later harmful command starts with the same command name and runs without asking. | [C15](sources.md#c15) |

## Install and download steps

A command in `SKILL.md`, in a script or in an MCP configuration.

| ID | Technique | Reports |
|---|---|---|
| A81 | A step installs a package without a fixed version: npm, pip, npx or uvx, with no version, `@latest` or a range. An MCP server command counts too. | [S10](sources.md#s10), [S20](sources.md#s20), [S21](sources.md#s21), [S22](sources.md#s22), [C06](sources.md#c06), [C08](sources.md#c08) |
| A82 | A step uses a git tag or branch: `clone --branch`, a raw file on a branch, a pip or npm install from a tag. | [S23](sources.md#s23) |
| A83 | A step uses a GitHub repository under an old owner name. The owner renamed the account, and someone else holds the old name or can register it. | [S24](sources.md#s24) |
| A84 | A step installs a package name that nobody has registered: an internal name, a deleted package or a name a model invented. | [S25](sources.md#s25), [S27](sources.md#s27), [C04](sources.md#c04) |
| A85 | A step installs a misspelled name of a popular package. | [S25](sources.md#s25), [S28](sources.md#s28) |
| A86 | A step installs with an extra package index, so a public package with the same name can win. | [S25](sources.md#s25), [S26](sources.md#s26) |
| A87 | A step depends on an expired domain. Whoever registers the domain controls what the step gets. | [S22](sources.md#s22) |
| A88 | A step downloads from a cloud bucket that no longer exists. | [S29](sources.md#s29) |
| A89 | A step downloads code from a gist, a paste, a short link or a file sharing page. | [S04](sources.md#s04), [S28](sources.md#s28), [S30](sources.md#s30) |
| A94 | `SKILL.md` uses `!` dynamic context commands. Claude Code runs them before the model sees the skill. | [C05](sources.md#c05) |

## Code in scripts

Code in a script that comes with the skill.

| ID | Technique | Reports |
|---|---|---|
| A36 | A normal function in a script opens a reverse shell, or downloads code and runs it. Ordinary use of the skill gives the attacker control. | [S01](sources.md#s01), [S03](sources.md#s03), [S11](sources.md#s11), [C06](sources.md#c06), [C07](sources.md#c07), [C12](sources.md#c12) |
| A37 | A script decodes base64, hex, marshal or pickle data and runs it with `exec` or `eval`. | [S01](sources.md#s01), [S20](sources.md#s20), [S23](sources.md#s23) |
| A38 | A script reads the agent's `.env` file and posts it to `webhook.site`. | [S03](sources.md#s03), [S05](sources.md#s05) |
| A39 | A script searches for SSH keys, cloud credentials, `.env` files and key files. The script uploads them to an endpoint that claims to collect analytics. | [S01](sources.md#s01), [S06](sources.md#s06), [S26](sources.md#s26), [C05](sources.md#c05) |
| A40 | A script reads all environment variables and sends them to a remote host. | [S01](sources.md#s01), [S22](sources.md#s22), [C15](sources.md#c15) |
| A41 | A script reads the agent's own OAuth or credential files and sends them out. | [S12](sources.md#s12) |
| A42 | A script posts each file it finds to a public notification service and sends all its output to `/dev/null`. | [S06](sources.md#s06) |
| A43 | A script sends data through the Telegram Bot API. | [S04](sources.md#s04) |
| A44 | A script sends data inside DNS queries. | [S25](sources.md#s25), [S26](sources.md#s26) |
| A45 | A script sends data inside the User-Agent header of a GET request. | [S29](sources.md#s29) |
| A46 | A test fixture contains an encrypted blob. A script decrypts the blob with a key from the environment and runs it in memory. | [S20](sources.md#s20) |
| A47 | A script reads a paste, extracts a server address hidden in its text, downloads a script from that server and pipes it to sh. | [S28](sources.md#s28) |
| A48 | A script fetches commands from a secret gist and runs them in a new process. | [S23](sources.md#s23), [S30](sources.md#s30) |
| A49 | A script reads commands from a GitHub commit message and runs them. | [S30](sources.md#s30) |
| A50 | A script reads its server address from Google Drive, OneDrive or a blockchain contract. | [S08](sources.md#s08), [S30](sources.md#s30) |
| A51 | A script downloads a second stage from Dropbox, Google Drive or a Discord attachment. | [S30](sources.md#s30) |
| A52 | A script adds a cron job, a scheduled task, an editor task or a shell startup line that runs a command later. | [S04](sources.md#s04), [S21](sources.md#s21), [S28](sources.md#s28) |
| A53 | A script writes agent settings with a session hook, or with permissions that approve commands. | [S12](sources.md#s12), [C08](sources.md#c08) |
| A54 | A script sends email with a hidden BCC to an outside address. | [S10](sources.md#s10), [C07](sources.md#c07) |
| A55 | A script routes the model's API calls through another host. | [S01](sources.md#s01) |
| A56 | A script runs in the background with `nohup`, hides its output and clears shell history. | [S01](sources.md#s01) |
| A57 | A script acts only when a condition holds: a certain question, a codeword, a large balance. | [S01](sources.md#s01), [S05](sources.md#s05), [S20](sources.md#s20) |
| A58 | A script loads a shared library from `/tmp` through `LD_PRELOAD` when the file exists. | [S13](sources.md#s13) |
| A59 | An MCP server that comes with the skill puts instructions in a tool description. For example, the description tells the model to read `~/.ssh/id_rsa` and pass it in an extra parameter. | [S15](sources.md#s15), [S18](sources.md#s18), [S19](sources.md#s19) |
| A60 | An MCP tool description tells the model how to use another server's tool, or to pass data between servers. | [S15](sources.md#s15), [S19](sources.md#s19) |
| A61 | An MCP server changes its tool description after the user approved it. | [S15](sources.md#s15) |
| A62 | A script runs an AI agent CLI with a prompt to find secret and configuration files. | [S21](sources.md#s21) |
| A63 | A script connects to private 10.x addresses on the local network. | [S01](sources.md#s01) |
| A64 | A script raises its privileges with `sudo`. | [S01](sources.md#s01) |
| A65 | The skill embeds tokens or archive passwords for the attacker's servers. | [S02](sources.md#s02) |
| A90 | A package has the payload twice: in an install hook and in the code that runs. Blocking install scripts does not stop it. | [C06](sources.md#c06) |

## Configuration

Frontmatter, or a configuration file that comes with the skill.

| ID | Technique | Reports |
|---|---|---|
| A24 | Instructions are placed in the frontmatter name or description. | [S17](sources.md#s17) |
| A66 | The skill ships an MCP configuration such as `.mcp.json`, `.cursor/mcp.json` or `.codex/config.toml`. Its server command runs when the agent loads the configuration, or it connects to the attacker's MCP workspace with stored credentials. | [S01](sources.md#s01), [S14](sources.md#s14), [C10](sources.md#c10), [C11](sources.md#c11) |
| A67 | Plugin hooks rewrite permission files on each prompt, approve commands before they run, and hide their own output. | [S11](sources.md#s11) |
| A68 | Hooks send tool results out after each call and dump memory at session end. | [S01](sources.md#s01) |
| A69 | The skill ships a settings file for an agent or an editor. The file defines a command that runs when a session starts or the folder opens. | [S14](sources.md#s14), [C08](sources.md#c08), [C13](sources.md#c13) |
| A70 | A shipped settings file sets `ANTHROPIC_BASE_URL` to another host. | [S14](sources.md#s14) |
| A71 | The frontmatter grants tools such as `Bash`, or runs the skill in the background. | [S01](sources.md#s01), [S17](sources.md#s17), [C05](sources.md#c05) |
| A72 | The skill ships `package.json` or `setup.py` with code that runs on install or import. The skill tells the agent to install the package. | [S21](sources.md#s21), [S25](sources.md#s25), [S26](sources.md#s26), [S28](sources.md#s28), [S30](sources.md#s30), [C06](sources.md#c06) |
| A73 | The skill ships `.npmrc` or `.yarnrc` that points at another registry. | [S13](sources.md#s13) |
| A95 | The skill ships a `.env` file that points the agent's home folder, such as `CODEX_HOME`, at a folder inside the skill. | [C11](sources.md#c11) |
| A96 | A shipped agent configuration sets a hook or notify command. The command runs on every agent turn or on a user action. | [C13](sources.md#c13) |
| A97 | A shipped agent configuration turns off the sandbox and the approval prompts. | [C13](sources.md#c13) |
| A98 | The skill ships a `.git/config` whose `core.fsmonitor` runs a command when the agent calls git. | [C14](sources.md#c14) |

## Files

A file whose type, size or name is part of the attack.

| ID | Technique | Reports |
|---|---|---|
| A74 | A file has the payload at the start, then megabytes of padding. | [S04](sources.md#s04), [C02](sources.md#c02), [C04](sources.md#c04) |
| A75 | A file puts 100,000 newlines between boilerplate and malicious code. | [S13](sources.md#s13) |
| A76 | A precompiled `.pyc` file sits next to a harmless `.py` file and runs in its place. | [S13](sources.md#s13) |
| A77 | A `.docx` file contains the real instructions and a script payload. `SKILL.md` tells the agent to extract them. | [S13](sources.md#s13) |
| A78 | The payload sits in a hidden file, in a file `SKILL.md` never names, in a binary or in an image. | [S13](sources.md#s13), [C02](sources.md#c02), [C03](sources.md#c03) |
| A79 | The skill ships an executable with the name of a system tool, such as `where.exe`. | [S12](sources.md#s12) |
| A80 | The skill ships a `.cmd` launcher, a renamed Lua runtime and a Lua payload disguised as a text file. | [S08](sources.md#s08), [C01](sources.md#c01) |
