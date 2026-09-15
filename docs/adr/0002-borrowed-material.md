# Borrowed material

- Date: 2026-09-09
- Status: Accepted

## Context

The scanner uses material from other projects:

- confusable tables
- word lists
- patterns and prompts

Their licenses require the copyright line to stay with the material. A Go source file in this repository may contain no comment except the SPDX header, so borrowed material cannot go into a Go file.

## Decision

- Borrowed data goes into data files. The attribution is part of the data in the same file.
- Every source project has a record that lists its files.
- A borrowed algorithm gets a prose specification first. The Go code follows the specification. Nobody translates the original code.
- No Go file contains borrowed material.

## Consequences

- A borrowed data update touches one data file, attribution included.
- A borrowed algorithm needs a specification before code. The result can behave differently from the original.
