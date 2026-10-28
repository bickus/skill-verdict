# tree-sitter-powershell

- Origin, with the version the build pins in `scripts/wasm.Containerfile`:
  https://github.com/airbus-cert/tree-sitter-powershell, `da65ba3acc93777255781b447f5e7448245df4bf`
- Copyright (c) 2023 Airbus CERT
- License: MIT, see `LICENSE` in this directory.

## Files in this repository carrying material from this source

None. `scripts/wasm` compiles the sources into `pkg/internal/syntax/treesitter.wasm`. The
repository does not contain that module, and every released binary embeds it.
