# Verdict

- Date: 2026-09-09
- Status: Accepted

## Context

Two problems with a scanner that sums finding scores:

- Enough small findings from any source reach the threshold, and the scanner blocks a skill with no serious finding.
- The scanner cannot always inspect everything: a registry can be down, the judge can fail. The report must show that as a scanner limit and not blame the skill.

## Decision

The verdict comes from finding severity and inspection completeness. There is no score.

- A live finding at or above the configured gate severity gives block, no matter which layer produced it.
- Live findings below the gate give review.
- An incomplete inspection is its own verdict state, between review and block. The state describes the scan, not the skill.

## Consequences

- Severity is the only tuning knob. Three configuration changes keep a rule from blocking:
  - lower its severity
  - raise the gate
  - remove the rule
- A noisy rule with a high severity blocks skills. The operator owns that severity.
- Any failed step makes the verdict incomplete. The scanner never reports clean for something it did not inspect.
- Consumers act on the verdict. The findings explain it.
