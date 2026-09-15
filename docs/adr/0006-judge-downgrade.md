# Judge downgrade

- Date: 2026-09-09
- Status: Accepted

## Context

Static rules match text, not meaning. In a benign skill most matches are false positives, and a verdict that cannot drop a false positive is useless. Letting a language model drop findings fixes that, but the skill under review is input to that same model. A malicious skill can talk the model into dropping anything. The model needs limits.

## Decision

A language model judges the findings, within these limits:

- Each rule declares whether the model may lower its findings.
- Each rule declares a floor, the lowest severity the model may assign.
- The model may lower a finding down to the floor and no further. The model may dismiss a finding only when the rule declares no floor.
- Configuration can change both settings for any rule.
- A dismissed finding stays in the report with a dismissed mark. The verdict ignores it.
- The gate sees the severity after judgement.
- When the model is not confident, the finding keeps its rule severity.

## Consequences

- The floor limits the damage from a manipulated or wrong judge. The rule author sets the default, the operator can change it.
- Every decision of the model is in the report.
- A finding the judge did not see keeps its rule severity. A failed judge makes the scan incomplete, so a missing judge cannot look like a clean skill.
