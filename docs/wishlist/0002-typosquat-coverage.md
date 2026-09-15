# Typosquat coverage

- Status: Deferred

## What it is

The package collector flags a name within one or two edits of a popular package, from a list of
forty-nine PyPI and thirty-one npm names borrowed from NVIDIA SkillSpector. The list covers the
most imitated packages and nothing else, and nothing refreshes it.

## What it needs

- A refresh path: a script that pulls the most downloaded packages from both registries, rewrites
  `pkg/internal/refcheck/data/popular-packages.json`, and records the date in the file.
- A larger list under the same distance rule, or a rule that scales the allowed distance with the
  name length.
- Scope-aware matching for npm, so `@scope/name` is compared on `name`.

## Why it waits

The list catches the classic imitations, and the collector already records the missing-package
fact, which is the finding that gates. A refresh path is a small script and a decision on the
source of truth; nothing in the scanner changes.
