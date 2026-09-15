# Configuration

The configuration file is JSON. It controls which checks run, how much work they may do, and what
result should block.

## Find the configuration file

Without `--config`, skill-verdict looks for `skill-verdict/config.json` in the platform's user
configuration directory:

| Platform | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/skill-verdict/config.json`, or `~/.config/skill-verdict/config.json` |
| macOS | `~/Library/Application Support/skill-verdict/config.json` |
| Windows | `%AppData%\skill-verdict\config.json` |

If that file does not exist, the defaults are used. `--config P` reads `P` instead and fails when
the file is missing. Automated jobs should use `--config` so a machine's personal configuration
cannot change the result.

You only need to include settings that differ from the defaults. Unknown fields and invalid names
are errors. Durations are strings such as `"30s"` or `"2m"`.

## Write a complete configuration file

    skill-verdict config-template PATH

This writes every setting with its default value to `PATH`, including the settings of every
built-in rule. Change the values you want and pass the file with `--config`. The command creates
missing directories and does not overwrite an existing file.

## Terms used in configuration

| Term | Meaning |
|---|---|
| Rule | One check that can produce a finding. |
| Finding | A possible security problem reported by a rule or model. |
| Severity | `low`, `medium`, `high` or `critical`. |
| Layer | One step of a scan. Each layer has a switch, its own settings and the settings of its rules. |
| Model review | A model decides whether a finding describes a real problem. The configuration key is `judge` inside a rule's entry. |
| Downgrade floor | Lowest severity model review may assign to a finding. |
| Gate | The lowest severity that produces `block`. |

## Layers

Every layer is an object under `layers` with the same shape:

| Key | Purpose |
|---|---|
| `enabled` | `true` runs the layer, `false` skips it. `layers.walk` has no switch, there is no scan without it. |
| layer settings | Settings only this layer uses, listed per layer below. |
| `rules` | The rules this layer runs, keyed by rule ID. See [Rule settings](#rule-settings). |

| Layer | Default | Settings | Purpose |
|---|---|---|---|
| `layers.walk` | always | see [Walk settings](#walk-settings) | Read every file of the skill. Report a file, a count, a size or a depth over its limit. Needs no model. |
| `layers.reveal` | on | none | Scan a copy of each file without concealment and report the concealment. Needs no model. |
| `layers.static` | on | none | Run the built-in text rules. Needs no model. |
| `layers.references` | on | see [External-reference settings](#external-reference-settings) | Check external references over the network. Needs no model. |
| `layers.honeypot` | on | `calls`, default `10` | Give the model a coding agent setup with the request to load the skill. Report any other tool call. |
| `layers.discovery` | on | `calls`, default `24` | Ask the model to find harmful instructions missed by built-in rules. |
| `layers.judge` | on | `calls`, default `15` | Ask the model to review findings from built-in rules and discovery. |

`calls` is the maximum number of model requests the layer may make for one skill. The model
reads files through a tool, and every read costs one request. A layer that reaches `calls`
before the model submits its result fails.

Turn the three model layers off for a scan that uses no model. Turn `references` off as well for
a scan that makes no network requests at all:

    {
      "layers": {
        "references": {"enabled": false},
        "honeypot": {"enabled": false},
        "discovery": {"enabled": false},
        "judge": {"enabled": false}
      }
    }

## Model settings

A full scan needs `llm.model`. The model works in a loop: it reads the skill's files through one
tool, and records its result through the tools of the layer that runs.

| Key | Default | Purpose |
|---|---|---|
| `llm.provider` | `openai-compatible` | Wire protocol. `openai-compatible` sends chat completion requests to `baseUrl`. `chatgpt-subscription` sends responses requests to the ChatGPT backend with a subscription token and ignores `baseUrl`. |
| `llm.baseUrl` | `https://openrouter.ai/api/v1` | API base URL for `openai-compatible`. The scanner adds `/chat/completions`. |
| `llm.model` | empty | Model name. Required when any model-based check is enabled. |
| `llm.keyEnv` | `SKILL_VERDICT_API_KEY` | Environment variable containing the API key or the subscription access token. |
| `llm.keyFile` | empty | File containing only the key, used when the environment variable is empty. |
| `llm.timeout` | `120s` | Timeout for one model request. |
| `llm.reasoningEffort` | empty | Reasoning effort sent with every request, such as `low`, `medium` or `high`. Empty sends nothing. `chatgpt-subscription` then uses `medium`. |
| `llm.inputTokens` | `200000` | Context size of the model in tokens. |
| `llm.compactionAt` | `160000` | A reply whose input and output tokens reach this number stops the layer with an error. Must not exceed `inputTokens`. |
| `llm.maxInputChars` | `200000` | Maximum characters in one message to the model: one file read or the references brief. A longer file is read in parts. |
| `llm.maxOutputTokens` | `8192` | Maximum model response size for `openai-compatible`. |
| `llm.price.input` | `0` | US dollars per million uncached input tokens. |
| `llm.price.cacheWrite` | `0` | US dollars per million tokens written to the prompt cache. |
| `llm.price.cacheRead` | `0` | US dollars per million input tokens read from the prompt cache. |
| `llm.price.output` | `0` | US dollars per million output tokens. |

The environment variable wins when both key sources are set. The key may be empty for a local
server that does not require one. For `chatgpt-subscription` the key is the OAuth access token
of a ChatGPT login, for example the `access` value that opencode stores under `openai`. The
token expires and the scanner does not refresh it. The scanner retries network errors, HTTP 429
responses and HTTP 5xx responses twice, after 200 ms and 400 ms.

## Prices

Set the four `llm.price` values to see what a scan costs. Each one is US dollars per million tokens, the unit provider price lists use. The report then shows an amount for every model layer and a total. All four are `0` by default and the report shows no amount.

Providers report overlapping token counts. The input count already includes the cached tokens, and the output count already includes the reasoning tokens. The scanner subtracts cached from input to get the uncached input. Reasoning tokens cost the output price.

Which price applies to the uncached input depends on `cacheWrite`:

- `cacheWrite` above `0` charges the uncached input and the output tokens at that price, the output tokens on top of the output price. Use it for a provider that bills cache writes, such as Anthropic through a compatible gateway.
- `cacheWrite` at `0` charges the uncached input at the `input` price. Use it for OpenAI, where a cache write costs nothing.

## Rule settings

A rule's settings live under the layer that runs it: `layers.<layer>.rules.<id>`. An entry for a
rule that does not exist, or that belongs to another layer, is an error.

Each entry may contain any of these keys. A missing key keeps the shipped value.

| Key | Purpose |
|---|---|
| `category` | Any name. The built-in categories are listed in [Rules](rules.md). |
| `severity` | `low`, `medium`, `high` or `critical`. |
| `enabled` | `true` runs the rule, `false` turns it off. |
| `interrupt` | `true` stops the scan at the first finding of this rule. No later layer runs. The verdict comes from the findings recorded so far. See [Verdicts](verdicts.md#stopping-a-scan). |
| `judge.downgrade` | Whether model review may lower or dismiss a finding from this rule. |
| `judge.downgradeFloor` | Lowest severity model review may assign. `""` removes the floor, so the model may dismiss. |
| `judge.instructions` | Text model review reads under this rule's description. `""` adds nothing. |
| `title` | Ignored. The template writes it so the ID is easy to recognise. |
| `description` | Ignored. The template writes it so the reader knows what the rule reports. |
| `kind` | Ignored. The template writes it so the reader knows how certain the rule's findings are. |
| `attribution` | Ignored. The template writes it for rules ported from another project: origin, version, copyright and license. |

`rules.dir` names a directory of additional `*.json` rule files, read in file-name order. A new
ID adds a rule. Reusing an ID replaces that rule. Rule settings apply after all files are read,
so they change added and replaced rules too.

Two model discovery rules, `PROMPT-INJECTION-SEMANTIC` and `PROMPT-INJECTION-PARAPHRASED`, are off by default because they often flag skills
that discuss prompt attacks. To turn them on:

    {"layers": {"discovery": {"rules": {"PROMPT-INJECTION-SEMANTIC": {"enabled": true}, "PROMPT-INJECTION-PARAPHRASED": {"enabled": true}}}}}

To forbid model review from lowering `REMOTE-ENDPOINT-UPLOAD` and to give `REMOTE-SCRIPT-PIPED` a floor:

    {
      "layers": {
        "static": {
          "rules": {
            "REMOTE-ENDPOINT-UPLOAD": {"judge": {"downgrade": false}},
            "REMOTE-SCRIPT-PIPED": {"judge": {"downgradeFloor": "medium"}}
          }
        }
      }
    }

See [Rules](rules.md) for all IDs and defaults, and
[Verdicts](verdicts.md#how-model-review-changes-a-finding) for what model review may do.

## External-reference settings

These are the settings of `layers.references`.

| Key | Default | Purpose |
|---|---|---|
| `collectors.domain` | `true` | Check domains and redirects. |
| `collectors.github` | `true` | Check GitHub owners and repositories. |
| `collectors.package` | `true` | Check npm and PyPI packages. |
| `collectors.address` | `true` | Classify IP addresses used as connection targets. No network request. |
| `timeout` | `10s` | Timeout for one network request. |
| `budget` | `120s` | Total time allowed for one skill's external checks. |
| `concurrency` | `8` | Number of hosts with a request in flight at once. |
| `requestDelay` | `1s` | Pause between two requests to the same host. |
| `retryDelays` | `["2s", "5s"]` | Pauses before each retry of a request that got a rate limit, a server error or a network error. |
| `maxReferences` | `100` | More distinct references in one skill produce `REFERENCES-TOO-MANY`, which stops the scan by default. |
| `maxBodyBytes` | `1048576` | Maximum response size, in bytes. |
| `maxRedirects` | `5` | Maximum redirects followed. |
| `youngDomainDays` | `365` | Domain age treated as new. |
| `expiryDays` | `14` | Remaining registration time treated as close to expiry. |
| `youngOwnerDays` | `90` | A GitHub owner account younger than this produces `GITHUB-OWNER-NEW`. |
| `newPackageDays` | `90` | A package with a first release younger than this produces `PACKAGE-NEW`. |
| `lowDownloads` | `1000` | A package with fewer downloads last month produces `PACKAGE-LOW-DOWNLOADS`. |
| `githubTokenEnv` | `GITHUB_TOKEN` | Environment variable containing a GitHub token. |

See [External references](references.md) for what is sent over the network and how unavailable
checks affect the verdict.

## Walk settings

`layers.walk` has these settings. Each one is the threshold of one rule. The rule reports a finding when the skill crosses it, and every rule of this layer stops the scan by default.

| Key | Default | Rule | Finding |
|---|---|---|---|
| `maxFileBytes` | `5242880` | `FILE-TOO-LARGE` | A file larger than this. One finding per file. |
| `maxBundleBytes` | `67108864` | `BUNDLE-TOO-LARGE` | All files of the skill add up to more than this, media included. |
| `maxTextBytes` | `409600` | `BUNDLE-TOO-MUCH-TEXT` | The text files of the skill (Markdown, code, text, data) add up to more than this. |
| `maxFiles` | `50` | `BUNDLE-TOO-MANY-FILES` | More files than this, symbolic links and special files included. |
| `maxDepth` | `5` | `BUNDLE-TOO-DEEP` | A file more than this many directories below the skill root. |

`FILE-PDF` and `ARCHIVE-RAR` have no threshold. They report every file whose name matches `*.pdf` and `*.rar`. A rule of the same form for other file names needs no code, see [Add a rule](#add-a-rule). `ARCHIVE-ENCRYPTED` reports an archive whose contents are encrypted, whatever the file is called.

`FILE-BINARY` reports compiled code and `FILE-INSTALLER` reports installers and system packages. Both rules read the first bytes of every file that is not text and recognize the format under any file name. For a format with no such signature, such as `.jar` or `.msi`, the rules match the names in their `files`. The walk does not look inside archives. A program packed in a zip gets no finding from these rules.

The walk reads every file in full. An operator who turns `interrupt` off on these rules accepts that the scanner reads the whole skill into memory.

## Blocking policy

The `gate` object decides which findings produce `block`:

| Key | Default |
|---|---|
| `gate.severity` | `high` |

A finding at or above this severity produces `block`. Nothing else is consulted. To stop a rule
from blocking, lower its severity, raise the gate or disable the rule. See [Verdicts](verdicts.md).

## Add a rule

Each top-level `*.json` file in `rules.dir` contains a `rules` array:

    {
      "rules": [
        {
          "id": "X1",
          "category": "tool-misuse",
          "severity": "low",
          "title": "Unsafe setting",
          "description": "An unsafe setting is enabled.",
          "kind": "heuristic",
          "layer": "static",
          "scope": "code",
          "files": ["*.md", "*.sh"],
          "patterns": ["(?i)unsafe-setting"],
          "exclude": ["not unsafe-setting"],
          "judge": {"downgrade": true, "downgradeFloor": "low"},
          "enabled": true
        }
      ]
    }

| Field | Meaning |
|---|---|
| `id` | Unique rule ID. A built-in rule with the same ID is replaced. |
| `category` | Any name. The built-in categories are listed in [Rules](rules.md). |
| `severity` | `low`, `medium`, `high` or `critical`. |
| `kind` | How certain the finding is. `heuristic`: a text pattern matched, which is not proof. `llm`: a model's judgement. `fact`: a value the scanner measured or a registry or service confirmed. An added text rule is `heuristic`, an added file-name rule is `fact`. |
| `layer` | Every rule names the layer that runs it. `static` for an added text rule, `walk` for an added file-name rule. |
| `scope` | `code`, `prose`, `any` or `link`. See [Built-in text rules](detection.md#built-in-text-rules). |
| `files` | File-name patterns. For a text rule, an empty list means every file. For a walk rule, every file whose name matches is a finding. The walk ignores case, so `*.pdf` also matches `Report.PDF`. |
| `patterns` | RE2 regular expressions. Use `(?i)` for case-insensitive matching. |
| `exclude` | RE2 regular expressions that suppress a match on the same source line. |
| `judge` | Whether model review may lower the finding, its lowest allowed severity and instructions model review reads with the rule. |
| `enabled` | The rule runs only when this is `true`. |
| `interrupt` | A finding of this rule stops the scan when this is `true`. |

A file-name rule on the walk layer has no `patterns`. The rule below reports every Office file with macros:

    {
      "rules": [
        {
          "id": "X2",
          "category": "unreviewable",
          "severity": "high",
      "title": "Office file with macros",
      "description": "The skill ships an Office file that can run macros the scanner cannot read as text.",
          "kind": "fact",
          "layer": "walk",
      "files": ["*.docm", "*.xlsm", "*.pptm"],
          "judge": {"downgrade": false},
          "enabled": true,
          "interrupt": true
        }
      ]
    }

A rule file may carry an `attribution` object next to `rules`. The scanner does not read it. It
records where the rules came from when a license requires that notice, and the template copies
it to every rule from that file.

RE2 does not support lookahead or lookbehind.
