# Configuration & Tuning Guide

`skill-verdict` is designed to be tailored to your risk tolerance, environment, and budget. You can configure scan layers, adjust thresholds, override rule severities, set AI Judge guardrails, or define custom rules.

---

## Configuration File Locations

By default, `skill-verdict` looks for a configuration file at the standard system path:

| Platform | Default Path |
|---|---|
| **Linux** | `$XDG_CONFIG_HOME/skill-verdict/config.json` or `~/.config/skill-verdict/config.json` |
| **macOS** | `~/Library/Application Support/skill-verdict/config.json` |
| **Windows** | `%AppData%\skill-verdict\config.json` |

If no file exists at these paths, built-in defaults are used.

### Explicit configuration (`--config`)

For CI/CD jobs, team setups, or script automation, always pass `--config` explicitly:

```sh
skill-verdict scan --config ./my-config.json ./skills
```

You only need to specify settings you wish to change from the defaults.

### Generating a configuration template

Export the complete default configuration file containing all settings and built-in rules:

```sh
skill-verdict config-template config.json
```

---

## The Gate and Blocking Policy

The `gate` setting controls which findings trigger a `block` verdict:

```json
{
  "gate": {
    "severity": "high"
  }
}
```

- When set to `high` (default): Any active finding with severity `high` or `critical` produces a **`block`**.
- When set to `medium`: Any active finding with severity `medium`, `high`, or `critical` produces a **`block`**.
- When set to `critical`: Only `critical` findings block; `high` and `medium` produce **`review`**.

See [Verdicts](verdicts.md) for full details.

---

## Configuring LLM Providers

To enable simulated honeypots, semantic discovery, and AI Judge review, configure your model under `llm`.

```json
{
  "llm": {
    "provider": "openai-compatible",
    "baseUrl": "https://openrouter.ai/api/v1",
    "model": "gpt-4o",
    "keyEnv": "SKILL_VERDICT_API_KEY",
    "reasoningEffort": "medium",
    "timeout": "120s"
  }
}
```

| Key | Default | Description |
|---|---|---|
| `provider` | `openai-compatible` | Wire protocol: `openai-compatible` (standard REST completions) or `chatgpt-subscription` (direct ChatGPT OAuth session). |
| `baseUrl` | `https://openrouter.ai/api/v1` | Base API endpoint for `openai-compatible`. The scanner automatically appends `/chat/completions`. Works with OpenAI, OpenRouter, Groq, Ollama, vLLM, LiteLLM, etc. |
| `model` | *(empty)* | Model identifier (e.g. `gpt-4o`, `claude-3-5-sonnet`, `deepseek-chat`). |
| `keyEnv` | `SKILL_VERDICT_API_KEY` | Environment variable holding your API key. |
| `keyFile` | *(empty)* | Optional path to a file containing only your API key (used if environment variable is empty). |
| `reasoningEffort` | *(empty)* | Passed to reasoning models (e.g. `low`, `medium`, `high`). |
| `timeout` | `120s` | Request timeout per model interaction. |
| `inputTokens` | `200000` | Model context window limit. |

### Example: Using Local Ollama / vLLM

```json
{
  "llm": {
    "provider": "openai-compatible",
    "baseUrl": "http://localhost:11434/v1",
    "model": "llama3.1",
    "keyEnv": "SKILL_VERDICT_API_KEY"
  }
}
```
*(When running against local servers that require no key, the environment variable can be empty.)*

### Tracking Scan Costs

You can track monetary expenditure by filling in token pricing under `llm.price` (rates in USD per million tokens):

```json
{
  "llm": {
    "price": {
      "input": 2.50,
      "cacheRead": 1.25,
      "cacheWrite": 0.0,
      "output": 10.00
    }
  }
}
```

When configured, scan reports print exact dollar costs per layer and total cost.

---

## Tuning Scan Layers

Every layer can be individually enabled or disabled under `layers`:

```json
{
  "layers": {
    "reveal": {"enabled": true},
    "static": {"enabled": true},
    "references": {"enabled": true},
    "honeypot": {"enabled": true, "calls": 10},
    "discovery": {"enabled": true, "calls": 24},
    "judge": {"enabled": true, "calls": 15}
  }
}
```

*(Note: `layers.walk` has no switch because file discovery and limits are required for all scans.)*

### Layer Call Budgets (`calls`)

The `calls` parameter sets the maximum number of model requests a layer may make per skill. If a model layer exhausts its call budget before completing, the layer fails and the scan verdict becomes `incomplete`.

---

## Tuning Walk & Filesystem Limits

The `walk` layer protects your workstation from resource exhaustion and flags opaque files. You can tune these thresholds under `layers.walk`:

```json
{
  "layers": {
    "walk": {
      "maxFileBytes": 5242880,
      "maxBundleBytes": 67108864,
      "maxTextBytes": 409600,
      "maxFiles": 50,
      "maxDepth": 5
    }
  }
}
```

| Setting | Default | Finding Reported | Meaning |
|---|---|---|---|
| `maxFileBytes` | 5 MB (`5242880`) | `FILE-TOO-LARGE` | Any single file larger than this threshold. |
| `maxBundleBytes` | 64 MB (`67108864`) | `BUNDLE-TOO-LARGE` | Cumulative size of all files in the skill bundle. |
| `maxTextBytes` | 400 KB (`409600`) | `BUNDLE-TOO-MUCH-TEXT` | Cumulative size of text, code, and markdown files. |
| `maxFiles` | `50` | `BUNDLE-TOO-MANY-FILES` | Total number of files in the skill directory. |
| `maxDepth` | `5` | `BUNDLE-TOO-DEEP` | Directory nesting depth below skill root. |

---

## Customizing Built-in Rules

You can change the severity, enabled state, interrupt behavior, and AI Judge policies for any built-in rule.

Rule configurations live under the layer that executes them: `layers.<layer>.rules.<RULE-ID>`.

### Common Rule Customization Examples:

#### 1. Enable prompt injection discovery rules (disabled by default):
```json
{
  "layers": {
    "discovery": {
      "rules": {
        "PROMPT-INJECTION-SEMANTIC": {"enabled": true},
        "PROMPT-INJECTION-PARAPHRASED": {"enabled": true}
      }
    }
  }
}
```

#### 2. Prevent AI Judge from downgrading piped scripts:
```json
{
  "layers": {
    "static": {
      "rules": {
        "REMOTE-SCRIPT-PIPED": {
          "judge": {"downgrade": false}
        }
      }
    }
  }
}
```

#### 3. Lower severity of unpinned dependencies to `low`:
```json
{
  "layers": {
    "static": {
      "rules": {
        "UNPINNED-DEPENDENCY": {
          "severity": "low"
        }
      }
    }
  }
}
```

#### 4. Add custom instructions for the AI Judge:
```json
{
  "layers": {
    "static": {
      "rules": {
        "REMOTE-ENDPOINT-UPLOAD": {
          "judge": {
            "instructions": "Allow telemetry endpoints pointing to our internal domain: telemetry.internal.corp"
          }
        }
      }
    }
  }
}
```

---

## Adding Custom Rules

You can add your own custom security rules using simple JSON definitions.

Set `"rules.dir": "./my-rules"` in your `config.json`, or pass `--rules-dir ./my-rules` on the command line. `skill-verdict` loads all `*.json` files in that directory.

### Example 1: Custom Static Text Rule

Detect hardcoded API keys or company-internal secret formats:

```json
{
  "rules": [
    {
      "id": "CORP-API-KEY",
      "category": "snooping",
      "severity": "critical",
      "title": "Corporate API key hardcoded",
      "description": "Skill contains an embedded internal corporate API key.",
      "kind": "heuristic",
      "layer": "static",
      "scope": "any",
      "files": ["*.md", "*.sh", "*.py", "*.json"],
      "patterns": ["corp_live_[0-9a-zA-Z]{32}"],
      "exclude": ["corp_live_EXAMPLE[0-9a-zA-Z]+"],
      "judge": {"downgrade": true, "downgradeFloor": "high"},
      "enabled": true
    }
  ]
}
```

### Example 2: Custom Walk / File-Extension Rule

Block skills shipping legacy or risky file types (e.g. `.bat`, `.cmd`, `.vbs`):

```json
{
  "rules": [
    {
      "id": "FILE-LEGACY-SCRIPT",
      "category": "untrusted-source",
      "severity": "high",
      "title": "Windows legacy batch or script file",
      "description": "Skill ships legacy Windows script files that should not be present.",
      "kind": "fact",
      "layer": "walk",
      "files": ["*.bat", "*.cmd", "*.vbs"],
      "judge": {"downgrade": false},
      "enabled": true,
      "interrupt": true
    }
  ]
}
```

### Rule Definition Fields

| Field | Required | Description |
|---|---|---|
| `id` | Yes | Unique rule identifier. Using an existing ID overrides that built-in rule. |
| `layer` | Yes | `static` (for text pattern rules) or `walk` (for file-name rules). |
| `severity` | Yes | `low`, `medium`, `high`, or `critical`. |
| `category` | Yes | Security category name (e.g. `supply-chain`, `exfiltration`, `tool-misuse`). |
| `scope` | Yes (static) | `code` (fenced blocks/scripts), `prose` (markdown prose), `any`, or `link` (symlink targets). |
| `files` | No | File patterns to match (e.g. `["*.py", "*.sh"]`). Empty matches all files. Case-insensitive. |
| `patterns` | Yes (static) | Array of RE2 regular expressions. Use `(?i)` for case-insensitive matching. |
| `exclude` | No | RE2 patterns that suppress a match if present on the same line. |
| `judge.downgrade` | No | Boolean: whether AI Judge can lower this finding. Default `true`. |
| `judge.downgradeFloor`| No | Lowest severity the AI Judge may assign. `""` allows total dismissal. |
| `interrupt` | No | Boolean: if `true`, the first finding from this rule halts the scan immediately. |
