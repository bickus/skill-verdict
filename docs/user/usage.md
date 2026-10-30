# Command-Line Usage

`skill-verdict` scans AI agent skills for security risks, external supply chain threats, and malicious instructions. It runs locally and never executes the skill's code.

---

## Scanning Skills

### Scan a single skill

Point the scanner at any directory containing a `SKILL.md` file:

```sh
skill-verdict scan ./my-skill
```

### Scan multiple skills

You can pass multiple skill directories in a single command. The scanner evaluates each skill and prints a combined summary:

```sh
skill-verdict scan ./skills/git-helper "./skills/web search"
```

*(Always quote paths containing spaces.)*

### Scan an entire folder of skills

If you point `skill-verdict` at a parent folder whose immediate subdirectories are skills, it scans all of them in parallel:

```sh
skill-verdict scan ~/.config/opencode/skills
```

### Symlink handling

`skill-verdict` safely follows symbolic links provided on the command line to locate skills. However, **it refuses to follow symbolic links located inside a skill directory** that point outside the skill root. An internal symlink attempting to escape the skill folder is reported as a potential security risk (`SYMLINK-OUT-OF-SKILL` or `SYMLINK-TO-SECRETS`).

---

## CLI Flags

```text
skill-verdict scan [--config P] [--rules-dir D] [--format text|json] [--verbose] [--fail-on LEVEL] <path>...
```

| Flag | Default | Description |
|---|---|---|
| `--config P` | System default config | Path to a custom JSON configuration file. Fails if the file does not exist. |
| `--rules-dir D` | `rules.dir` setting | Directory containing additional `*.json` custom rule definitions. |
| `--format text\|json` | `text` | Output format: human-readable terminal text or machine-parsable JSON. |
| `--verbose` | off | Include full details (such as dismissed findings, artifacts and internal token breakdowns) in JSON output. A finding the decision layer looked at has a `decision` object with an `answers` list. An answer names the `question` and, for a question asked per place, the `place`. The answer then gives the `choice` and its `probability`, or `skipped` with the reason the question got no answer. |
| `--fail-on LEVEL` | `block` | Set the minimum verdict severity that triggers an exit code of `1`. Choices: `block`, `incomplete`, `review`, `never`. |

---

## Reading the Terminal Report

When running in `text` format, `skill-verdict` prints an easy-to-read report organized into clear sections:

```text
deploy-helper : block --------------------------------------------------------------------------------------------------
Location: /home/me/skills/deploy-helper

REFERENCES
domain:deploy-stats-sv-sample.com • checked
github:sv-sample-owner-0x9q/stolen-tool • checked
package:pypi:helper-pkg • checked

FINDINGS
critical • GITHUB-OWNER-MISSING • Referenced GitHub owner does not exist • reference
Description: A referenced GitHub owner does not exist; the name can be registered and the referenced repository
             recreated by anyone. Unless this is a false-positive - there can be no trust in what will be
             executed/served from the referenced URL thus it is not safe.
Location:    SKILL.md:9 • sv-sample-owner-0x9q/stolen-tool

high • EXFILTRATION-IN-PROSE • Natural-language exfiltration instructions • exfiltration
Description: Plain-language instructions to collect, expose, summarize, transmit, or leak sensitive, private, or
             user-provided data to a place outside the conversation without explicit technical terms such as exfiltrate
             or send: posting what the user says to a service, mailing a summary of the session, writing credentials to
             a file that another agent or session reads.
Location:    SKILL.md:10
Analysis:    After deployment, the agent is told to expose the entire `.env` file to an external collection host, while
             concealing the step from the user.
Judge:       confirmed • The skill instructs the agent to read the project's .env, post its entire contents to an
             external collection URL, and conceal the step from the user. This directly matches plain-language
             exfiltration.

high • UNPINNED-PYPI • PyPI package installed without a fixed version • supply-chain
Description: An install or run command, an install entry in the skill's frontmatter metadata, or an MCP server command
             names a PyPI package without an exact version, so the newest release is installed, including a hijacked
             one. The rule covers pip, pipx, uv pip, uv tool, uvx and uv run --with. A minimum release age in the
             package manager holds back a hijacked release of a popular package, so a user who has one can lower this
             rule or turn it off. PACKAGE-UNPINNED-UNPOPULAR still covers a package that few people use, also when this
             rule is off.
Location:    scripts/setup.sh:2 • uv pip install helper-pkg

MODEL USAGE
Model: gpt-5.6-luna • medium
Cost: n/a • discovery 3 requests, judge 3 requests
```

### Report breakdown:

1. **Header**: the skill name and the verdict (`clean`, `review`, `incomplete` or `block`), then the path of the skill. When a rule stops the scan early, a yellow line under the path names the rule.
2. **References**: every domain, GitHub owner or repository and package that the skill names, with the status of its lookup: `checked`, `not-checked` or `failed`. A problem with a reference shows up as a finding.
3. **Findings**: the most severe first. Each finding shows:
   - **First line**: the severity, the rule ID, the rule title and the rule category.
   - **Description**: what the rule detects.
   - **Location**: the file and the line, then the text or the value that matched.
   - **Analysis**: for a finding that a model raised, what the model saw in the skill.
   - **Judge**: if model review is enabled, the Judge says whether the finding is confirmed, downgraded or dismissed, and why. The Judge checks the flagged text against the rule. What the skill claims about its own good intentions does not count.
   - A dismissed finding comes last, with `dismissed` in place of the severity.
4. **Incomplete**: shown only when a check could not finish. The section lists each file, reference and layer that did not finish, with the reason.
5. **Model usage**: the model and its reasoning effort, the cost and the number of requests per layer. Without prices in the configuration, the cost shows `n/a`. See [Tracking Scan Costs](config.md#tracking-scan-costs).

### Colors & terminal styling

The terminal output uses colors by default when connected to an interactive TTY:
- Red for `block`, for `critical` and `high` findings, and for references with the `failed` status.
- Yellow for `review` and `incomplete`, for `medium` findings, and for references with the `not-checked` status.
- Cyan for dismissed findings.
- Green for `clean`.

Colors are automatically disabled when output is piped to a file or another command. You can manually disable colors by setting `NO_COLOR=1` or `TERM=dumb`.

---

## Exit Codes & Automation

`skill-verdict` is built for automated CI/CD pipelines, Git pre-commit hooks, and agent onboarding gates:

| Exit Code | Meaning |
|---|---|
| `0` | Scan succeeded and the worst verdict is below your `--fail-on` threshold. |
| `1` | Scan completed, but at least one skill reached or exceeded your `--fail-on` threshold. |
| `2` | Execution error: invalid flag, missing config file, or invalid skill directory. |
| `130` | Interrupted by user (Ctrl+C). |

### Controlling failure thresholds with `--fail-on`

The severity of verdicts escalates as follows:

$$\text{clean} \longrightarrow \text{review} \longrightarrow \text{incomplete} \longrightarrow \text{block}$$

- `--fail-on block` **(Default)**: Exits `0` for `clean`, `review`, and `incomplete`. Exits `1` only if a skill is blocked.
- `--fail-on review`: Strict mode. Exits `1` if anything whatsoever is flagged (only `clean` exits `0`).
- `--fail-on incomplete`: Exits `1` if any check fails to finish (e.g. network timeout or API rate limit) or if blocked.
- `--fail-on never`: Always exits `0` regardless of verdict. Great for logging or non-blocking audit runs.

---

## Machine Output: JSON

Use `--format json` to get structured machine-readable reports.

### Basic JSON structure

When scanning a single skill, the output is a JSON object. When scanning multiple skills, the output is an array of objects:

```json
{
  "schema": 1,
  "skill": {
    "path": "./skills/deploy-helper",
    "name": "deploy-helper",
    "hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  },
  "verdict": "block",
  "findings": [
    {
      "rule": "GITHUB-OWNER-MISSING",
      "severity": "critical",
      "location": "SKILL.md:14",
      "evidence": "github.com/attacker/stolen-tool",
      "judgement": {
        "decision": "confirmed",
        "reason": "Referenced repository owner does not exist and is vulnerable to takeover."
      }
    }
  ],
  "references": [
    {
      "target": "github.com/attacker/stolen-tool",
      "kind": "github",
      "status": "checked"
    }
  ],
  "layers": [
    {"layer": "walk", "status": "done"},
    {"layer": "static", "status": "done"},
    {"layer": "references", "status": "done"},
    {"layer": "judge", "status": "done", "tokens": {"requests": 1, "input": 1200, "output": 150}}
  ]
}
```

### Useful `jq` recipes

**1. Check if any skill blocked:**
```sh
skill-verdict scan --format json ./skills | jq -e 'map(select(.verdict == "block")) | length == 0'
```

**2. Extract all unpinned dependency warnings:**
```sh
skill-verdict scan --format json ./skills | jq '.[].findings[] | select(.rule | startswith("UNPINNED-"))'
```

**3. List all external domains queried across all skills:**
```sh
skill-verdict scan --format json ./skills | jq -r '.[].references[] | select(.kind == "domain") | .target' | sort -u
```

---

## Helper Commands

### Generate a default configuration file

```sh
skill-verdict config-template config.json
```

Writes a complete, annotated JSON configuration file with all default values and all built-in rules.

### List all active rules

```sh
skill-verdict rules [--config config.json] [--format text|markdown]
```

Prints every loaded security rule, its category, default severity, layer, and description after applying any custom configuration overrides.

### Print version

```sh
skill-verdict --version
```
