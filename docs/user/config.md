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
    "provider": "openai-completions",
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
| `provider` | `openai-completions` | Wire protocol: `openai-completions` (Chat Completions API) or `openai-responses` (Responses API). |
| `baseUrl` | `https://openrouter.ai/api/v1` | Base API endpoint. The scanner appends `/chat/completions` or `/responses` to it. Works with OpenAI, OpenRouter, Groq, Ollama, vLLM, LiteLLM, etc. |
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
    "provider": "openai-completions",
    "baseUrl": "http://localhost:11434/v1",
    "model": "llama3.1",
    "keyEnv": "SKILL_VERDICT_API_KEY"
  }
}
```
*(When running against local servers that require no key, the environment variable can be empty.)*

### Example: Responses API

The Responses API serves reasoning models that keep state between turns. A ChatGPT subscription token works against `https://chatgpt.com/backend-api/codex`, an OpenCode Go key against `https://opencode.ai/zen/go/v1`.

```json
{
  "llm": {
    "provider": "openai-responses",
    "baseUrl": "https://opencode.ai/zen/go/v1",
    "model": "gpt-5.6-luna",
    "reasoningEffort": "medium"
  }
}
```

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
    "ast": {"enabled": true},
    "static": {"enabled": true},
    "references": {"enabled": true},
    "decision": {"enabled": false},
    "honeypot": {"enabled": true, "calls": 10},
    "discovery": {"enabled": true, "calls": 24},
    "judge": {"enabled": true, "calls": 15}
  }
}
```

*(Note: `layers.walk` has no switch because file discovery and limits are required for all scans.)*

### Layer Call Budgets (`calls`)

The `calls` parameter sets the maximum number of model requests a layer may make per skill. If a model layer exhausts its call budget before completing, the layer fails and the scan verdict becomes `incomplete`.

### Decision Layer (`layers.decision`)

The decision layer has its own model settings, separate from `llm`, because a decision model takes a state and questions and returns probabilities, not text. The layer is off by default. It needs the `ast` and `static` layers. See [Detection](detection.md) for what it dismisses and when.

```json
{
  "layers": {
    "decision": {
      "enabled": true,
      "baseUrl": "https://openrouter.ai/api/alpha",
      "model": "~typesafe/jev-latest",
      "keyEnv": "SKILL_VERDICT_DECISION_API_KEY",
      "keyFile": "",
      "timeout": "1m0s",
      "maxStateChars": 60000,
      "maxRequestChars": 120000,
      "confirmProbability": 0.3,
      "dismissProbability": 0.8,
      "inputPrice": 0.042
    }
  }
}
```

| Setting | Meaning |
|---|---|
| `baseUrl`, `model` | The scanner posts to `<baseUrl>/decisions`. |
| `keyEnv`, `keyFile` | Where the API key comes from. The configuration never holds the key. |
| `maxStateChars` | Largest text the layer sends. The layer cuts a longer text at this size. |
| `maxRequestChars` | Largest request, text and questions together. The layer does not send a larger request, and its questions get no answer. |
| `confirmProbability` | Lowest summed probability of the options a `confirm` row names. Default `0.3`. |
| `dismissProbability` | Lowest summed probability of the options a `severity`, `dismiss` or `misfire` row names. A usage answer raises a rule and enters the reference brief only at this probability or above. Default `0.8`. |
| `inputPrice` | USD per million input tokens, used for the cost in the report. |

Each probability is above `0` and at most `1`, and `confirmProbability` is at most `dismissProbability`.

A rule takes part when its rule data has a `decision` block with questions and an outcome table. See [Detection](detection.md#layer-6-decision) for the format. The block is rule data, so the configuration cannot change it. To change a table, copy the rule into a file in your rules directory and edit it there. Set `judge.downgrade` to `false` and the model no longer lowers the findings of the rule. The layer then only confirms them:

```json
{
  "layers": {
    "static": {
      "rules": {
        "REMOTE-ENDPOINT-UPLOAD": {"judge": {"downgrade": false}}
      }
    }
  }
}
```

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

#### 2. Keep piped scripts at their severity:
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
      "judge": {"downgrade": false},
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
| `files` | No | File patterns to match (e.g. `["*.py", "*.sh"]`). Empty matches all files. Case-insensitive. A pattern with a `/` matches the last folders and the file name of the path, so `commands/*.md` matches `plugins/p/commands/run.md`. |
| `patterns` | Yes (static) | Array of RE2 regular expressions. Use `(?i)` for case-insensitive matching. |
| `exclude` | No | RE2 patterns that suppress a match if present on the same line. |
| `lowered` | No | RE2 patterns of commands that end at `medium`: commands that only read, and scripts shipped in the skill folder. A load-time command that matches one of them in full, or a fenced block whose every line does, gets severity `medium`. Only `COMMAND-ON-LOAD` uses it. |
| `installers` | No | Groups of install commands, one group per package ecosystem. A group lists the manifests it installs in `files` and RE2 patterns of the commands in `patterns`. A finding in a manifest stays only when another file of the skill names the manifest or matches a pattern of the manifest's group. Only `UNPINNED-DEPENDENCY` uses it. |
| `judge.downgrade` | No | Boolean: whether a model may lower this finding. The decision layer lowers a rule with a `decision` block, the AI Judge lowers a discovery rule. Default `false`. |
| `judge.downgradeFloor`| No | Lowest severity a model may assign. `""` allows total dismissal. |
| `decision` | No | Closed questions for the decision model and the outcome table over the answers. See [Detection](detection.md#layer-6-decision). |
| `usages` | No | A reference rule only, instead of `decision`. Lists the usages of a reference that confirm the finding. The scanner turns it into the usage question. A user action lowers the finding to `medium`, a citation or documentation dismisses it. |
| `fields` | No | A discovery rule only. Closed questions that the discovery model answers for every finding it reports. Each field has a `question`, its `options` with one sentence each, and `accept`, the options the rule takes. The scanner refuses a finding with an answer outside `accept`. See [Detection](detection.md#layer-8-model-discovery). |
| `interrupt` | No | Boolean: if `true`, the first finding from this rule halts the scan immediately. |
