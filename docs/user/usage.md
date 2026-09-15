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
| `--verbose` | off | Include full details (such as dismissed findings and internal token breakdowns) in JSON output. |
| `--fail-on LEVEL` | `block` | Set the minimum verdict severity that triggers an exit code of `1`. Choices: `block`, `incomplete`, `review`, `never`. |

---

## Reading the Terminal Report

When running in `text` format, `skill-verdict` prints an easy-to-read report organized into clear sections:

```text
[BLOCK] ./skills/deploy-helper

References:
  github.com/attacker/stolen-tool (github: owner does not exist)
  https://bit.ly/3xY9zA (domain: url shortener)

Findings:
  CRITICAL  GITHUB-OWNER-MISSING  Referenced GitHub owner does not exist
    Location: SKILL.md:14
    Evidence: github.com/attacker/stolen-tool
    Judge:    Confirmed. The repository points to an abandoned username that can be re-registered by an attacker.

  HIGH      UNPINNED-INSTALL  Package install without pinned version
    Location: scripts/setup.sh:3
    Evidence: pip install helper-pkg
    Judge:    Downgraded to LOW. Package is installed in an isolated venv for testing, but unpinned versions remain risky.

Model usage:
  Model: gpt-4o (medium)
  Cost:  $0.0042 (3 requests)
```

### Report breakdown:

1. **Header & Verdict**: Shows the skill name/path and final verdict (`CLEAN`, `REVIEW`, `INCOMPLETE`, or `BLOCK`).
2. **References**: Real-time status of external domains, GitHub accounts/repos, and npm/PyPI packages mentioned in the skill.
3. **Findings**: Sorted from most severe to least severe:
   - **Severity**: `LOW`, `MEDIUM`, `HIGH`, or `CRITICAL`.
   - **Rule ID & Title**: Built-in rule that detected the issue.
   - **Location**: Exact file and line number (`SKILL.md:14`).
   - **Evidence**: The raw snippet or value that triggered the alert.
   - **Judge**: If AI review is enabled, the Judge explains whether the finding is a true risk, harmless documentation context, and why it was confirmed, downgraded, or dismissed.
4. **Incomplete**: Only shown if a check could not finish (e.g. network timeout or file read failure).
5. **Model usage**: Token counts, API requests, and estimated monetary cost (if prices are set in config).

### Colors & terminal styling

The terminal output uses colors by default when connected to an interactive TTY:
- Red for `BLOCK` and `CRITICAL`/`HIGH` findings.
- Yellow for `REVIEW` and `MEDIUM` findings.
- Cyan for `INCOMPLETE` scans.
- Green for `CLEAN` verdicts.

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
