# skill-verdict

Scanner for agent skills, plugins, hooks and MCP configurations. Go, standard library,
one static binary.

## Rules

- **Commits are signed off.** Always `git commit -s`. The `.githooks/commit-msg` hook and
  the `dco` workflow reject commits that fail the rules in `CONTRIBUTING.md`.
- **No comments in code.** The only exception is the SPDX header from `CONTRIBUTING.md` on
  the first two lines of every source file. The `pre-commit` hook rejects both a missing
  header and any other comment.
- **Standard library only, static binary, no cgo.**
- **YAGNI.** Code exists for a present need. Code without a caller, code only tests call,
  and generality with a single implementation are removed, version control keeps them.
- **Caps, enforced by the hook and the linter.** 400 lines per file, 60 lines and cognitive
  complexity 15 per function.
- **`scripts/check` is the only way to build, lint or test.** Never assemble `go build`,
  `go test` or linter commands by hand. Run it once, when a unit of work is complete and
  before the commit. Not after each edit, not after documentation changes, never twice in a
  row.
- **The `tests` skill decides whether a change gets a test and what the test looks like.**
- **Use the repository skills in `.claude/skills/` whenever their description matches the
  change at hand.**
- **A decision that constrains future changes gets an ADR in `docs/adr/` before the code.**
  Check existing records before designing anything.

## Documents

| File | Contents |
|---|---|
| `README.md` | What the tool does. |
| `CONTRIBUTING.md` | License, file headers, sign-off. |
| `docs/adr/README.md` | Index of architecture decisions. |
| `pkg/internal/llmcall/README.md` | Where the model prompts live and how placeholders work. |
| `scripts/README.md` | Index of scripts. |
| `third_party/README.md` | How borrowed material is recorded. |
