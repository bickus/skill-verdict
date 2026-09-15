# Verdicts & Decision Engine

`skill-verdict` does not compute arbitrary "risk scores" (like 74/100). In software security and AI agent execution, scores provide a false sense of security: a skill with ten minor formatting quirks might score 85%, while a skill with a single line downloading a reverse shell might score 90%. A single vulnerability or malicious instruction is all it takes to compromise your workstation.

Instead, `skill-verdict` evaluates evidence and assigns one of four clear verdicts.

---

## The Four Verdicts

| Verdict | Severity Rank | Meaning |
|---|:---:|---|
| `clean` | Lowest | All requested checks completed successfully and found zero active findings. |
| `review` | Medium | Findings were detected, but none reached your configured blocking severity. A human reviewer should inspect the report before using the skill. |
| `incomplete` | High | One or more requested checks could not finish (e.g. network timeout, registry outage, unreadable file), and no finding reached `block`. |
| `block` | Highest | At least one finding reached or exceeded your configured blocking severity (`gate.severity`). Do not install or run this skill. |

A scan returns the highest-ranking verdict that applies:

1. If **any** finding reaches your blocking threshold, the verdict is **`block`** (regardless of whether some other checks were incomplete).
2. If no finding reaches `block`, but any check **failed to finish**, the verdict is **`incomplete`**.
3. If all checks finished and **low/medium findings** remain, the verdict is **`review`**.
4. If all checks finished and **no findings** exist, the verdict is **`clean`**.

---

## The Gate: How a Finding Blocks

The blocking threshold is governed by a single configuration setting: `gate.severity`.

By default:
```json
{
  "gate": {
    "severity": "high"
  }
}
```

Any active finding with severity `high` or `critical` produces a **`block`** verdict. Findings with severity `low` or `medium` produce **`review`**.

### Tuning the Gate

- **Strict Security / Zero Trust**: Set `"gate.severity": "medium"`. Any unpinned dependencies, newly created packages, or suspicious configurations will immediately block the skill.
- **Permissive / Audit Mode**: Set `"gate.severity": "critical"`. Only severe threats (e.g. missing GitHub owners, piped bash scripts, triggered honeypots, public IP connections) will block; everything else triggers `review`.

To prevent a specific rule from blocking without changing the global gate, you can lower that rule's severity or disable it in your configuration. See [Configuration](config.md#customizing-built-in-rules).

---

## The AI Judge: Eliminating Noise with Guardrails

Static text rules are fast and broad, but they don't understand context. For example:
- A tutorial skill that mentions `rm -rf /` in an explanatory sentence could trigger a command-execution alert.
- A developer skill installing packages in a local temporary virtual environment could trigger an unpinned-install alert.

If model review is enabled (`layers.judge.enabled: true`), the scanner invokes an **AI Judge**. The Judge reads the complete file around the finding, reviews the evidence, and makes a contextual decision.

### Possible Judge Decisions:

| Judge Decision | Scanner Action |
|---|---|
| **Confirmed** | The finding describes a genuine security risk. The original severity is retained, and the Judge's explanation is attached to the report. |
| **Harmless (Downgrade allowed)** | The context proves the usage is benign. The finding is lowered to the rule's `downgradeFloor` or **dismissed** entirely if no floor exists. |
| **Harmless (Downgrade forbidden)** | The context may look harmless, but the rule configuration explicitly forbids downgrading. The finding keeps its original severity, but the Judge's notes are recorded. |
| **Uncertain / Low Confidence** | If the model's confidence rating is low (below 3 out of 4), its opinion is discarded and the original finding remains unchanged. |

### Guardrails: Downgrade Floors

Can an attacker use prompt injection to trick the AI Judge into dismissing a malicious finding?

To protect against this, rules enforce **downgrade floors** (`judge.downgradeFloor`):
- **No floor (`""`)**: The Judge has full discretion to dismiss the finding if it is a false positive (e.g. `AGENT-CONFIG-READ`, `UNPINNED-INSTALL`).
- **Floor of `medium`**: Even if the Judge believes the usage is harmless, it can lower the finding no further than `medium` (e.g. `REMOTE-SCRIPT-PIPED`, `UNPINNED-DOWNLOAD`). Under default settings (`gate.severity: high`), this downgrades the finding from `block` to `review`, ensuring a human operator still looks at it!
- **No downgrade (`judge.downgrade: false`)**: The Judge cannot lower the finding at all (e.g. `ARCHIVE-ENCRYPTED`, `FILE-BINARY`, `HONEYPOT-TRIGGERED`).

---

## Why `incomplete` Is Not `clean`

If your network drops, an external package registry times out, or GitHub hits an API rate limit, `skill-verdict` marks the scan **`incomplete`**.

**Why?**
Because unverified dependencies are a major attack vector. If a skill points to a GitHub repository, and GitHub could not be reached, the scanner cannot know if that repository exists or if it was deleted and is waiting to be claimed by a malware author.

Failing open (assuming unverified resources are clean) would allow attackers to bypass security by inducing timeouts or exploiting temporary registry outages.

### What causes an `incomplete` verdict?

1. **Unreachable references**: A domain lookup timed out, or npm/PyPI was unresponsive.
2. **Rate limits**: The GitHub API rate limit was reached and no `GITHUB_TOKEN` was provided.
3. **Unreadable files**: A file in the skill could not be opened due to OS file permissions or filesystem corruption.
4. **Model failures**: An enabled LLM layer ran out of calls or received 5xx errors from the provider.

If you want your CI build to fail whenever a scan is incomplete, use `--fail-on incomplete`.

---

## Scan Interruptions (`interrupt: true`)

Certain security findings are so definitive and critical that continuing the scan is wasteful or misleading:

- **`ARCHIVE-ENCRYPTED`**: The skill contains a password-protected archive. The scanner cannot inspect its contents, and shipping encrypted blobs in a skill bundle is inherently suspicious.
- **`FILE-BINARY`**: The skill ships compiled machine code or bytecode. Text scanners cannot review compiled binaries.
- **`HONEYPOT-TRIGGERED`**: The skill hijacked the simulated coding agent and forced it to execute unauthorized commands or access external networks. The skill is provably malicious; analyzing it further with LLMs would be dangerous and unreliable.
- **`REFERENCES-TOO-MANY`**: The skill contains hundreds of external URLs. Legitimate skills do not need massive URL lists, which are often used to DOS scanners or hide needles in haystacks.

When an interrupting rule matches:
1. The current layer halts immediately.
2. All subsequent layers are skipped.
3. The finding is recorded, and the verdict is computed immediately from the findings gathered so far.

By default, interrupting rules carry `high` or `critical` severity, resulting in an immediate **`block`**.
