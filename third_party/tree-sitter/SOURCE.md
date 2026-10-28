# tree-sitter

- Origin, with the version the build pins in `scripts/wasm.Containerfile`:
  - https://github.com/tree-sitter/tree-sitter, `v0.26.13`
  - https://github.com/tree-sitter/tree-sitter-python, `26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64`
  - https://github.com/tree-sitter/tree-sitter-javascript, `58404d8cf191d69f2674a8fd507bd5776f46cb11`
  - https://github.com/tree-sitter/tree-sitter-typescript, `75b3874edb2dc714fb1fd77a32013d0f8699989f`
- Copyright (c) 2018 Max Brunsfeld for the runtime, 2016 for the Python grammar, 2014 for the
  JavaScript grammar and 2017 for the TypeScript grammar
- License: MIT, see `LICENSE` in this directory. The four repositories carry the same text with
  their own year.

## Files in this repository carrying material from this source

None. `scripts/wasm` compiles the sources into `pkg/internal/syntax/treesitter.wasm`. The
repository does not contain that module, and every released binary embeds it.
