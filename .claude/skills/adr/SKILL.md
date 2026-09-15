---
name: adr
description: Use before writing or changing an architecture decision record in docs/adr.
---

# Architecture Decision Records

An ADR records a durable decision, not a snapshot of its implementation. Copy
`docs/adr/template.md`. File name `NNNN-short-title.md`, numbered in order.

Include only:

- **Context:** the problem or constraint that makes the decision necessary.
- **Decision:** the general rule future work follows, scoped to the problem, not to its first use.
- **Consequences:** the tradeoffs that follow directly from the decision.

Leave out implementation details, paths, tuning values, test results, discarded experiments
and speculative edge cases. Date and status are metadata, not narrative.

Before finishing, ask:

- Would an ordinary implementation change force this ADR to change?
- Would another use of the same decision require a duplicate ADR?

If either answer is yes, the ADR is too specific. Correct wording in place. Delete content
that does not help a reader apply the decision.

Add the record's row to `docs/adr/README.md`.
