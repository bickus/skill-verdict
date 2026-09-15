---
name: dependencies
description: Use before adding an import outside the Go standard library or a go.mod requirement. Decides between the standard library, vendoring the small part, and adding a module.
---

# Dependencies

Default: Go standard library. No cgo. One static binary for Linux, Windows and macOS.

## Before adding a module

Answer in order. The first yes decides.

1. Does the standard library do it, even with more code on our side? Use the standard
   library.
2. Is the needed part small, stable, and does it not need upstream fixes to stay correct?
   Vendor that part: copy the minimum into our tree following the `borrowed-code` skill. A
   reference to the origin and version goes in `SOURCE.md`, not a runtime dependency.
3. Does it need to track a moving target: a wire format that changes, a parser with security
   fixes, a database driver? Add the module. State the reason in the commit message.

## When a module is added

- Pin the version in `go.mod`, commit `go.sum`.
- No `replace` to a fork without a written reason.
- Pure Go only. A module that needs cgo is rejected.
- One module per need. A second module for the same need replaces the first.
