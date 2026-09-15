# Walk layer and interrupting rules

- Date: 2026-09-12
- Status: Accepted

## Context

The walk reads the files of a skill before any other step, and what the walk meets is evidence about the skill. Each of these hides content from review, and a skill can use each of them on purpose:

- a file too large for anyone to review
- a bundle of hundreds of files
- a directory tree deeper than any skill needs
- an asset in a form that is not text

A scanner that treats these as technical bounds cuts or skips the content and reports the scan as incomplete. The form of the skill never becomes a finding.

Some findings also make the rest of a scan pointless. Nobody can review the bundle. The reference checks and the model requests then only cost time and money, and the skill gets more chances to mislead the model. The operator has to say which findings stop a scan.

## Decision

The walk is a layer. The walk layer runs first and has no switch, since there is no scan without it. The layer reads every file in full. The layer never unpacks a container and never judges a container by the files inside it. The layer's settings are the thresholds its rules compare against. Each of these is a finding of a rule of the walk layer:

- a file, a count, a size or a depth over its threshold
- a file whose name matches the rule

Every rule has an `interrupt` setting that the operator owns. Findings go through one recording point in the pipeline. When a recorded finding comes from a rule with `interrupt` on, three things happen:

- the running layer stops at its next check
- no later layer runs
- the pipeline records the stopped layer and every layer it did not run

The verdict follows the recorded findings by severity like any other. An interrupted scan is complete, since the operator asked for the stop.

## Consequences

- A skill's form can block on its own. The thresholds, the severities and the switches are operator settings.
- A program packed inside an archive gets no finding from the walk.
- No layer and no rule knows how a stop works. A rule of any layer can interrupt.
- An operator who turns `interrupt` off on a walk rule accepts that the scanner reads the whole bundle into memory.
- A stopped scan leaves work undone on purpose. The report says which rule stopped it.
