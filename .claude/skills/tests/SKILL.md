---
name: tests
description: Use before planning, writing, changing or deleting a test.
---

# Tests

A test exists to fail when the program is wrong. Nothing else justifies one.

## A test is kept only when all of these hold

- It exercises behavior a caller relies on, through the exported API or the CLI. A change
  requested in a conversation is not a behavior.
- It asserts on the result: return values, findings, written files, responses, exit codes.
  Not on which functions were called, in what order, with what arguments.
- Its expected values come from what must be true for the caller, written without reading
  the implementation. They are literals.
- A realistic wrong implementation would fail it.
- A refactor that keeps behavior leaves it green. A test that must change when only the
  implementation changes is deleted, not updated.
- Its name states the behavior and the case.

## What gets tests

- Every finding class: one input that produces it and one near miss that does not.
- Every input format the tool reads, with malformed input.
- Everything with a security consequence: paths, size limits, redirects, what the judge may
  and may not drop.
- Every bug fix starts with the test that reproduces the bug.

Wiring, argument passing, code without branches and the standard library get none.

## Shape

- Table tests. One behavior per function, one row per case.
- Fixtures under `testdata/`. Network through `httptest`.
- A fake only for an external service, behind the interface the code already has. No mocks
  of our own types.
- No logic in a test: no loops or conditionals that compute the expectation.
- The test lands in the same commit as the change.
