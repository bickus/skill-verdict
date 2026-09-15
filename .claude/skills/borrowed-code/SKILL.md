---
name: borrowed-code
description: Use before copying or translating code or any other assets from another project into this repository. Covers allowed licenses and the attribution steps that go in the same change.
---

# Borrowed code

Applies to code, prompts, regex tables, Unicode tables, advisory data, test samples. A close
translation from another language counts as a copy.

## Allowed sources

Apache-2.0, MIT, BSD-2, BSD-3, ISC, CC0, public domain. Nothing else: no GPL, LGPL, AGPL,
MPL, SSPL, CC-BY-SA, and nothing without a license file.

## Steps, in the same change

1. New source: create `third_party/<source>/` with the source's `LICENSE` copied verbatim and a
   `SOURCE.md` with origin URL, version or commit, copyright holder, license name. Add a block
   for the source to `NOTICE`.
2. List every file that carries the material under "Files in this repository carrying material
   from this source" in that `SOURCE.md`. Update the list when a file is added, moved or
   removed.
3. In each such file, keep the origin copyright line under ours in the SPDX header. Use the
   holder's line as written in their license or headers.
4. Say in the commit message what was taken and from where.

## Not steps

- No inline comments marking borrowed lines. The header and `SOURCE.md` are the record.
- No copying of a whole file when a function is needed. Take the minimum.
- No relicensing of the borrowed part. It stays under its own license inside ours.
