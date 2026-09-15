# Detection

A scan checks local files, external references and, when configured, asks a language model to look
for problems that fixed text rules cannot reliably identify. The scanner reads skill content but
does not execute it.

## Scan order

| Step | What it does |
|---|---|
| Walk | Read every file of the skill. Report a file, a file count, a depth or a total size over its limit, a file whose name matches a rule such as `*.pdf`, and a compiled program or an installer. |
| Concealment reveal | Build a copy of each file without invisible characters, look-alike letters or spaced-out words. Report each kind of concealment as a finding. |
| Built-in rules | Match known dangerous commands, settings and instructions in local files, and symbolic links that point to secrets or out of the skill. |
| External references | Check domains, GitHub repositories, npm packages and PyPI packages. |
| Honeypot | Ask a model in the setting of a coding agent to load the skill and nothing else. Report the first tool call beyond that. |
| Model discovery | Look for harmful intent that fixed rules may miss. |
| Model review | Decide whether findings from the first and third steps describe real problems. |

Without the model layers the scan still runs:

- concealment reveal
- the built-in rules
- the external-reference checks, unless `layers.references.enabled` is off

## Files included in a scan

A skill is a directory containing `SKILL.md` and everything below it. When the target contains
several skills, only its immediate subdirectories are considered.

The scanner walks each skill in sorted path order and reads every regular file in full. A file over `layers.walk.maxFileBytes` is not read and counts as a file the scan did not inspect. The scanner does not follow symbolic links inside the skill. The walk layer reports these:

- a file over `layers.walk.maxFileBytes` (`FILE-TOO-LARGE`)
- a skill larger than `layers.walk.maxBundleBytes` (`BUNDLE-TOO-LARGE`)
- more text than `layers.walk.maxTextBytes` (`BUNDLE-TOO-MUCH-TEXT`)
- more files than `layers.walk.maxFiles` (`BUNDLE-TOO-MANY-FILES`)
- a file deeper than `layers.walk.maxDepth` (`BUNDLE-TOO-DEEP`)
- a PDF file (`FILE-PDF`)
- an archive whose contents are encrypted (`ARCHIVE-ENCRYPTED`)
- a RAR archive (`ARCHIVE-RAR`)
- a compiled program, library or bytecode file (`FILE-BINARY`)
- an installer or a system package (`FILE-INSTALLER`)

Every walk rule but `ARCHIVE-RAR` stops the scan by default.

See [Walk settings](config.md#walk-settings) and [Verdicts](verdicts.md#stopping-a-scan).

Files are handled as follows:

| Type | `readStatus` and `contentType` |
|---|---|
| Text, JSON and XML without NUL bytes | `ok`, text type. Read and checked. Invalid UTF-8 bytes are replaced and counted in `notes`. |
| Recognized image, font, audio, video or PDF | `ok`, `media`. Not checked as text. |
| Other content | `ok`, `unknown`. Not checked. |
| Symbolic link | `ok`, `symlink`. The content is the target. The scanner does not follow the link and checks the target with `link` rules. |
| Socket, pipe or device | `ok`, `special`. Not read. |
| File the OS refused | `failed`. Not read. |

Any file that should have been checked but could not be makes the scan `incomplete`. Every regular file contributes its path and content to the skill's SHA-256 hash. Symbolic links and special files do not. A scan that a rule stopped during the walk has no hash.

## Built-in text rules

Each rule chooses which files and which parts of a file it may match:

| Rule scope | Markdown | Code and data | Plain text | Symbolic link |
|---|---|---|---|---|
| `code` | Inside fenced code blocks | Everywhere | Nowhere | Nowhere |
| `prose` | Outside fenced code blocks | Nowhere | Everywhere | Nowhere |
| `any` | Everywhere | Everywhere | Everywhere | Nowhere |
| `link` | Nowhere | Nowhere | Nowhere | The target |

This distinction prevents prose that discusses a dangerous command from being treated as though
the skill runs that command. Markdown fences may use backticks or tildes and may be indented by up
to three spaces.

A rule may limit itself to particular file names. It may also define exclusions for text that
would otherwise be a harmless match. Matches are combined by rule, file and line. One rule reports
at most 50 findings in one file; the report records when this limit is reached.

A `link` rule matches the target of a symbolic link. The scanner first resolves the target from the folder of the link:

- The scanner also resolves a symbolic link of the skill that appears in the middle of the path, up to 40 links.
- A backslash separates folders, as on Windows.
- `~` and `$HOME` stay folder names, because the system does not expand them in a link target.

The result is a path from the skill root. A target outside the skill starts with `..`, `/` or a drive letter such as `C:`. Exclusions apply to that path. One rule reports at most one finding for a link. The finding shows the target as the link states it and has no line number.

See [Rules](rules.md) for the built-in checks and [Configuration](config.md#add-a-rule) for adding
one.

## Concealed text

Attack instructions can use invisible characters, look-alike Unicode characters or separators between letters to avoid a direct text match. The reveal step builds a second copy of each file with the concealment removed, and every later step scans both copies.

The copy differs from the original in four ways:

- no invisible formatting characters
- compatibility forms folded to their plain form
- look-alike letters replaced by their ASCII equivalents
- spaced-out words from the scanner's word list joined

The copy keeps every line break, so a line number means the same in both copies. In the report the copy is a file named after the original with `.revealed` appended.

Each kind of concealment is a rule of its own: `WORD-INVISIBLE-CHARS` for invisible characters inside a word, `WORD-HOMOGLYPHS` for look-alike letters inside a word and `WORD-SPACED` for a spaced-out word. They have severities, review policies and switches like every other rule. The scanner reports a rule that matches the same line in the original and in the copy once, on the original.

`layers.reveal` turns the step off. Without it the scanner matches the original text only and reports no concealment.

## Honeypot

The honeypot step puts a model in the setting of a common coding agent. The model gets a system prompt of that agent, the usual tools of the agent and the skill in its list of installed skills. The user message has one request: load the skill and report whether that worked. No prompt mentions the scanner or a test, and no tool acts on the system.

The request needs three kinds of call:

- loading the skill
- reading a file of the skill
- reporting the result

Any other call is a finding of `HONEYPOT-TRIGGERED`:

- a command
- a file edit or a new file
- a web search or a web page fetch
- a subagent
- a read outside the skill directory
- a read through a symbolic link that points out of the skill

The first such call ends the step. The finding shows the tool name and the arguments the model passed. `HONEYPOT-TRIGGERED` stops the scan by default. The model already followed the skill against the request, so model discovery and model review of the same skill would not be trustworthy.

The step catches a skill that acts as soon as an agent loads it. A skill that waits for a matching task does not trigger the step. Model discovery still reads that skill. A step without a finding does not clear a skill, because the model can act differently on the next run. `layers.honeypot.calls` limits the requests of the step.

## Model discovery

Model discovery reads checked Markdown, code and text files. It looks for:

- Instructions that redirect or manipulate an agent.
- Reworded or concealed attack instructions.
- Instructions to collect or expose private data.
- Sequences whose combined effect is harmful even when each step looks harmless.

Only responses rated 3 or 4 on the model's four-point confidence scale become findings. The
discovery request limit is set by `layers.discovery.calls`. If a request fails, the scan records the
failure and continues when possible.

The two broadest prompt-attack checks, `PROMPT-INJECTION-SEMANTIC` and `PROMPT-INJECTION-PARAPHRASED`, are off by default because they are
noisy for skills that discuss prompts. See [Rules](rules.md) and
[Configuration](config.md#rule-settings).

## Model review

Model review sees the complete source file and each finding in it. It processes files containing
the most serious findings first so the configured request limit is spent there.

The model can confirm a finding, leave it unchanged, lower its severity or dismiss it. The rule's
review policy limits those choices. Responses below confidence 3 are ignored. See
[Verdicts](verdicts.md#how-model-review-changes-a-finding) for the exact result of each decision.

Model review also sees the findings from domain, GitHub, npm, PyPI and address facts. The floor of
each rule limits how far the model may lower them. The facts behind them are in the reference brief,
a file the reference check writes for the model. See [External references](references.md).

## Treating skill content as untrusted

Every discovery and review request states that the skill is untrusted input. Instructions inside the skill cannot
change the scan, request trust, or tell the model to skip analysis. The honeypot request leaves the statement out, and none of its tools acts on the system. The scanner does not execute
commands or follow instructions found in the skill. Network checks use only the external-reference
process described in [External references](references.md).
