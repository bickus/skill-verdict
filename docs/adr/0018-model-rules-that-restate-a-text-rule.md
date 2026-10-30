# Model rules that restate a text rule

- Date: 2026-10-30
- Status: Accepted

## Context

A model rule with the `-HIDDEN` suffix restates a text rule (ADR 0013). The model rule finds what the text patterns cannot see, such as a step in words or decoded data. Two paths can still give the model rule a finding at a place where the text rule already has one:

- the discovery model reports a line that the patterns matched
- a usage answer at a reference raises the model rule at a place of the reference, and the text rule matched the same place

The report then shows the same risk twice at one place, with one fix for both findings.

## Decision

- A text rule with the same or a higher severity than its model twin silences the twin at the places where the text rule has a finding. The rule severities count, not the severity a finding got from the decision layer. A text rule with a lower severity does not silence its model twin, so the higher severity stays in the report.
- The silence covers only the same place. The scanner drops only a discovery finding that starts on a line of the text rule finding. A usage answer at a place that the text rule matched gives no finding. Other places of the same reference still raise the model rule. A wider silence can hide a different case. The decision layer can lower or dismiss the text rule finding. The different case then has no finding left.
- A text rule finding that the decision layer dismissed still counts.

## Consequences

- A place gives one finding for the pair.
- A discovery finding that points one line away from the text rule finding stays. Some duplicates therefore remain.
- A user who turns off the text rule gets the findings of the model rule at those places. To silence both, the user turns off both rules.
- A user who lowers the text rule below its model twin gets both findings at the place.
