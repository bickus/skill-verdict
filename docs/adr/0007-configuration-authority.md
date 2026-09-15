# Configuration is the final authority

- Date: 2026-09-10
- Status: Accepted

## Context

The scanner ships with the author's opinions:

- which checks exist
- how severe each one is
- how far the model may lower each one
- which severity blocks a skill

All of them follow the author's threat model. The operator who runs the scan lives with the results and often has a different threat model. A fixed default makes the tool useless wherever the author guessed wrong. A fixed default also takes a security decision away from the person responsible for it.

## Decision

- Configuration can change every default the scanner ships. No check, severity, review policy or verdict threshold is exempt.
- The scanner rejects a configuration only when it cannot apply it:
  - an unknown name
  - an unparsable value
  - a value outside the accepted range
- The scanner never rejects a configuration because the author dislikes the result.
- An author who wants to keep a protection on ships it as a default. The documentation explains the reason.

## Consequences

- An operator can configure a scanner that never blocks, or one that blocks everything. Both are the operator's decision and responsibility.
- The shipped defaults are the author's judgement. The code does not enforce it.
- Validation checks form only, so error messages are about form only.
