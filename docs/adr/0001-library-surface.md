# Library surface

- Date: 2026-09-09
- Status: Accepted

## Context

The scanner is a Go library with a thin command on top. Another repository will import the library into a daemon and pass in its own LLM provider. Each importable package is an API we have to keep stable. The pipeline and the collectors still change often. Public access to them would freeze their current design.

## Decision

Public packages:

- `scan`, the entry point.
- `rules`, the rule set. The caller loads it once and gives it to the scan and to everything that shows a finding.
- `config`, `finding`, `refs`, `llm` and the packages under `llm`. They depend only on each other and the standard library.

Everything else goes under `internal/` and can change at any time.

## Consequences

- Changing a public type or one of its string values can break users of the library. The same string values also appear in rule data and in output.
- A user of the library sees the finding and the rule that reported it. How the scanner produced the finding stays hidden.
- Internal packages change without notice.
