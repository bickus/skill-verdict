# Usage

## Scan skills

    skill-verdict scan [--config P] [--rules-dir D] [--format text|json] [--verbose] [--fail-on LEVEL] <path>...

Each `<path>` can be:

- One skill directory containing a regular `SKILL.md` file.
- A directory whose immediate subdirectories are skills.

The scanner checks several paths in one run and writes one report. Put the flags before the first path. Quote a path that contains spaces, for example `"./my skills"`. One missing path or one path without skills stops the command. The scanner then checks none of the skills.

The scanner follows a symbolic link passed as `<path>`. It does not follow symbolic links inside
a skill because they may point outside the directory being checked.

### Scan modes

What a scan does is set in the configuration file, not by flags. Turn the model checks off with
`layers`, and the network checks off with `layers.references.enabled`. See
[Configuration](config.md#layers). Enabled text rules run in every mode.

### Flags

| Flag | Default | Purpose |
|---|---|---|
| `--config P` | User configuration file | Read configuration from `P`. The command fails if `P` does not exist. |
| `--rules-dir D` | `rules.dir` | Load additional rule files from `D`. |
| `--format text\|json` | `text` | Choose human-readable or JSON output. |
| `--verbose` | Off | Write every field of the JSON report. |
| `--fail-on LEVEL` | `block` | Exit 1 when the result reaches `LEVEL`. Accepted values are `block`, `incomplete`, `review` and `never`. |

Ctrl-C stops the scan at once:

- The skill in progress gets no report. No other skill starts.
- Text reports of the skills that finished stay in the output.
- A scan of more than one skill still ends with the summary line. The line counts the skills that did not finish as `skipped`. Model usage in the summary covers only the finished skills.
- The command writes no JSON report.
- The command exits with code 130.

### Colors

The text report uses color when it writes to a terminal. A redirect to a file or a pipe gives the plain text. `NO_COLOR` and `TERM=dumb` also turn color off. Color repeats what the text already says, so a report without color is complete.

### Exit codes

| Code | Meaning |
|---|---|
| 0 | The scan finished and its worst verdict is below `--fail-on`. |
| 1 | The scan finished and its worst verdict reached `--fail-on`. |
| 2 | The command could not run because an argument, configuration, path or output was invalid. |
| 130 | Ctrl-C stopped the scan. |

Verdicts increase in this order: `clean`, `review`, `incomplete`, `block`. Therefore,
`--fail-on review` accepts only `clean`, while `--fail-on never` always exits 0 after a completed
command.

## Read text output

Each skill starts with a line that names the skill and its verdict, then a line with the path of the skill. A scan of more than one skill ends with a line that counts the skills per verdict.

A stopped scan adds one line under the path, for example `interrupted by FILE-PDF in walk`. `FILE-PDF` is the rule that stopped the scan and `walk` is the layer it stopped.

The blocks for one skill come in this order:

- `References`, one line per external reference with its status, and the reason when the check did not finish.
- `Findings`, one block per finding, the most serious first, dismissed findings last.
- `Incomplete`, the work the scan did not do. The block appears only with the verdict `incomplete`.
- `Model usage`, when a model ran. `Model` names the model and its reasoning effort. `Cost` gives the price of the scan from the prices in the configuration, or `n/a` without prices, and the number of requests of each model layer. The token counts are in the JSON output.

The first line of a finding has its severity, or `dismissed`, then the rule ID, the title of the rule and its category. The lines under it are:

- `Description`, the description of the rule.
- `Location`, the file and line inside the skill, then the text that caused the finding. A finding about a whole file has no line number.
- `Analysis`, the text the model returned about its own finding. The line appears only when model review did not judge the finding.
- `Judge`, the decision of model review and its reason.

## Read JSON output

One scanned skill produces one JSON object. Scanning a directory of skills produces an array.
Each object has this shape:

    {
      "schema": 1,
      "skill": {},
      "verdict": "review",
      "findings": [],
      "references": [],
      "layers": [],
      "rules": {}
    }

`skill` contains the path, name and a SHA-256 hash of the regular files in the skill. The hash does not include symbolic links or special files. A scan that a rule stopped during the walk has no hash.

### Finding fields

| Field | Meaning |
|---|---|
| `rule` | ID of the rule that reported the problem. Its title, category, kind and description are under `rules`. |
| `severity` | Current severity after any model review. |
| `location` | File and line, such as `SKILL.md:6`, or a range, such as `SKILL.md:6-9`. A finding about a whole file has only the file name. A file name ending in `.revealed` is the copy of that file with concealment removed. |
| `evidence` | Text or external value that caused the finding. |
| `judgement` | Model decision and reason. |

### Rule fields

`rules` maps the ID of every rule in `findings` to these fields:

| Field | Meaning |
|---|---|
| `title` | Short name of the rule. |
| `category` | Security category, such as `exfiltration` or `supply-chain`. |
| `kind` | How certain the rule's findings are. `heuristic`: a text pattern matched, which is not proof. `llm`: a model's judgement. `fact`: a value the scanner measured or a registry or service confirmed. |
| `description` | What the rule detects. |

`references` contains the status and facts for each external reference. See
[External references](references.md).

`layers` lists every scan step, also the steps that did not run after a stop. A model layer has `tokens`: the number of requests and the input, cached, output and reasoning tokens summed over them. Prices in the configuration add `cost` to each model layer and a top level `cost` for the whole result. See [Verdicts](verdicts.md#work-that-did-not-finish) for what makes the verdict `incomplete`.

### Layer fields

| Field | Meaning |
|---|---|
| `layer` | Name of the scan step. |
| `status` | `done`, `failed`, `interrupted` or `skipped`. `interrupted` is the step a rule stopped, `skipped` is a step after it. |
| `error` | Why the whole step failed. |
| `interrupt` | ID of the rule that stopped the step. |
| `errors` | One entry per file the step gave up on, with the error text. |

## List rules

    skill-verdict rules [--config P] [--rules-dir D] [--format text|markdown]

This prints every loaded rule after the rule settings under `layers` and additional rule files
are applied.
The Markdown form is used for the [rule reference](rules.md).

## Write a configuration template

    skill-verdict config-template <path>

This writes the complete default configuration to `<path>`: every setting with its default
value and the settings of every built-in rule. Missing parent directories are created. The
command fails when `<path>` already exists. Edit the file and pass it with `--config`. See
[Configuration](config.md#write-a-complete-configuration-file).

## Print the version

    skill-verdict --version

Released binaries print their module version. A binary built without version information prints
`devel`.
