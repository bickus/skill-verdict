# Verdicts

A **finding** is a possible security problem. Each finding has a severity: `low`, `medium`, `high`
or `critical`.

A scan returns the first verdict that applies:

| Verdict | When it applies |
|---|---|
| `block` | A finding reached the severity that stops use of the skill. |
| `incomplete` | At least one requested check did not finish, and no finding produced `block`. |
| `review` | At least one finding stayed below the blocking severity, and neither result above applies. |
| `clean` | Every requested check finished and nothing was found. |

The tool does not calculate a risk score. Findings explain the result; adding many minor findings
does not turn them into one serious finding.

## When a finding blocks

`gate.severity` sets the blocking severity and defaults to `high`. A finding at or above it
produces `block`. Nothing else is consulted: not the rule that found it, not whether a model saw
it, not its category.

A finding below `gate.severity` produces `review`.

To stop a rule from blocking, lower its severity or set `enabled` to `false` in the rule's entry
under its layer in `layers`, or raise `gate.severity`.

## How model review changes a finding

Built-in text rules can match harmless documentation. When enabled, model review checks each
finding in the context of its file.

| Model decision | Result |
|---|---|
| The finding is real | Keep its severity and mark it `confirmed`. |
| The finding is harmless and lowering is forbidden | Keep its severity and record the model's reason. |
| The finding is harmless and lowering is allowed | Lower it to the rule's downgrade floor, or dismiss it when no floor is set. |
| The model is not confident | Ignore the model response and leave the finding unchanged. |

A downgrade floor is the lowest severity model review may assign. Every rule ships one as a
default and the rule's entry under its layer in `layers` changes it.

Model review treats a finding from the revealed copy of a file like any other. The concealment itself is a separate finding from one of the `WORD-` rules. Model review may lower a `WORD-INVISIBLE-CHARS` finding to `low` and cannot dismiss it. Model review may dismiss a `WORD-HOMOGLYPHS` or `WORD-SPACED` finding.

Dismissed findings remain in the text report. JSON output lists them under `dismissed` only with `--verbose`.

## Scans without model review

With the model layers off, findings keep the severity their rule gives them, and the gate treats
them like any other finding. A text match at `gate.severity` blocks. Raise the gate or lower the
rule's severity if that is not wanted.

## Work that did not finish

The JSON `references` and `layers` lists record every reference and scan step. The JSON output does not list files. The text report's `incomplete` section lists the files, references and layers that did not finish.

| Item | Complete | Not complete |
|---|---|---|
| File | `readStatus` is `ok` and `contentType` is not `unknown` | `readStatus` is `failed` or `contentType` is `unknown` |
| External reference | `checked` | `not-checked` or `failed` |
| Layer | `done`, `interrupted` or `skipped`, and the `errors` list is empty | `failed`, or `done` with a file in `errors` |

One unfinished item makes the verdict `incomplete` unless a finding already produces `block`.
This does not accuse the skill of a problem. It means the requested scan did not inspect everything.

## Stopping a scan

A rule with `interrupt` set to `true` stops the scan with its first finding. The layer that found it stops at once. No later layer runs. The report marks the stopped layer `interrupted` and the later ones `skipped`. The walk rules `FILE-TOO-LARGE` to `ARCHIVE-ENCRYPTED`, `FILE-BINARY`, `FILE-INSTALLER`, `REFERENCES-TOO-MANY` and `HONEYPOT-TRIGGERED` ship with `interrupt` on. Any other rule can get it in its entry under `layers`.

A stopped scan gets its verdict from the findings recorded up to the stop. Severity decides, as in every scan. A stopped scan counts as complete, since the configuration asked for the stop. So a rule with `interrupt` on and a severity below `gate.severity` gives `review`, and the same rule at or above the gate gives `block`.

## Exit on a verdict

`--fail-on` controls the command's exit status:

| Value | Exit 1 for |
|---|---|
| `block` | `block` |
| `incomplete` | `incomplete`, `block` |
| `review` | `review`, `incomplete`, `block` |
| `never` | Nothing |
