# Missing repository under a young owner

- Status: Deferred

## What it is

`GITHUB-REPO-MISSING` is low, because only the owner can create the repository again and the skill already trusts the owner. A young owner account weakens that reason. The original owner may have left, and someone else may have registered the name since then. The new owner can create the repository with any content.

The scanner reports the two facts as separate findings: the low `GITHUB-REPO-MISSING` and the medium `GITHUB-OWNER-NEW` for an account created within `youngOwnerDays`. Neither finding reaches the gate on its own.

## What it needs

- A fact rule for a missing repository whose owner account is younger than `youngOwnerDays`. Its default severity is at or above the gate.
- The GitHub collector raises it from the owner and repository lookups it already makes, with no new request.
- The two separate findings stay, so the judge still sees each fact.

## Why it waits

There is no sample of a skill that uses this pattern. ADR 0015 keeps the account age as a separate finding.
