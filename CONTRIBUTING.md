# Contributing

## License

Contributions are accepted under the Apache License 2.0 in `LICENSE`. Every source file
starts with:

```
// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors
```

A file that carries material from a third party keeps that party's copyright line under
ours and is listed in `third_party/<source>/SOURCE.md`. The procedure is in
`.claude/skills/borrowed-code/SKILL.md`.

## Sign-off

Every commit carries a `Signed-off-by:` line with the same name and email as the commit
author, which certifies the `DCO` in this repository. `git commit -s` adds it. The `dco`
workflow rejects pull requests with unsigned commits and commits that carry an AI tool as
co-author or an AI session link.

Enable the local hooks once after cloning. `commit-msg` enforces the sign-off rules,
`pre-commit` checks the staged files and runs `scripts/check`:

```
git config core.hooksPath .githooks
```
