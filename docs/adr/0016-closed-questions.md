# Model answers closed questions

- Date: 2026-10-28
- Status: Accepted

## Context

A rule matches text or a fact and starts a finding. For many findings the right severity depends on a reading of the passage: who runs the command, where a value goes, what the skill does with a reference. A model can read the passage.

A model that reads the passage together with a paragraph of policy and then decides whether the finding stands applies the policy with its own judgment. Two runs on the same skill end in different verdicts. A passage written for the model can make it drop a finding. The policy is prose, and a change to it is a change in wording with an effect nobody can predict.

## Decision

A rule that a model may lower or dismiss states, as rule data:

- one or more closed questions about the passage, each with a fixed set of answers
- an outcome table over the answers: which combinations confirm the finding, which dismiss it, which give it another severity

The model picks answers. Code applies the table. A combination the table does not list leaves the finding at its severity. A dismissal needs a confident answer. A confirmation does not.

An answer that the passage is not what the rule describes dismisses the finding. Every other dismissing answer ends at the rule's floor, as ADR 0006 requires.

The text the model reads is a slice that code builds from the finding: the instruction a line is part of, or the path a value takes through the code. The slice marks where the code stopped following a value. A finding whose slice the scanner cannot build or cannot send keeps its severity.

A dismissal follows from answers about what the passage does. What the passage says about itself, that it is safe, unused or only documentation, answers nothing.

A model that judges in free text judges only findings a model raised, and only against the definition of the finding's class.

## Consequences

- A policy change is a data change. A new rule that a model may lower comes with its questions and its table.
- A finding the model cannot place with confidence stays. More findings reach the reader, and none disappears on a guess.
- The rule author decides in advance which passages count as benign. The model cannot add a reason.
- A pattern that misfires is dismissed by the model even on a rule with a floor. The floor bounds policy, not matching.
- A model with no tool loop can answer, since the slice carries what the question needs.
