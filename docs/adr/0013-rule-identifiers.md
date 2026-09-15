# Rule identifiers

- Date: 2026-09-14
- Status: Accepted

## Context

A rule id appears where its title does not: in a configuration override, in a JSON report a
program consumes, in a bug report, in a conversation. The first rule set numbered its rules
per layer or per category, in several forms. An id such as `SC13` told a reader nothing, and
two ids of the same rule set followed different forms. A reader of the rules table found
related rules scattered, because the ordering key carried no meaning.

## Decision

- An id is uppercase words joined by hyphens. Each word is plain English.
- An id names what the rule found: first the subject, then the condition. `PACKAGE-MISSING`,
  `DOMAIN-NEW`, `FILE-TOO-LARGE`.
- The subject word is the same for every rule about the same kind of thing. Rules about one
  kind of thing therefore sort together.
- A condition may lead the id only when it forms a noun phrase with several subjects and all
  of those rules share one remedy. `UNPINNED-GIT` and `UNPINNED-DEPENDENCY` both mean pin it.
- The same condition takes the same word everywhere: `NEW`, `MISSING`, `UNPINNED`,
  `TOO-MANY`, `TOO-LARGE`.
- A model rule that restates a text rule takes the text rule's id and the suffix `-HIDDEN`.
- An id carries no layer, no category, no kind and no number.
- A rule whose meaning changes gets a new id. The old id is never reused.

## Consequences

- A reader understands a finding, an override or a report line without the rule set at hand.
- The rules table reads in groups when sorted by id.
- An id is a stable contract. Changing one breaks every configuration that overrides it, so
  a rename is a deliberate change with a record, never a side effect of other work.
- Ids are longer than numbers. A report line grows by a few words.
