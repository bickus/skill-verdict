# Code parsers

- Date: 2026-09-18
- Status: Proposed

## Context

Skills contain code in multiple languages, both in separate files and in markdown code blocks. Several checks require analyzing code structure to:

- distinguish comments and string literals from executable statements
- track values passed into function calls
- track where code defines and uses variables

Regular expressions cannot answer these questions. Simple text patterns cannot distinguish comments from executable code. Malicious skill authors can use this ambiguity to evade detection.

The scanner builds as a single static binary without cgo. Tree-sitter provides grammars for most skill languages, but its runtime uses C. The pure Go parser `mvdan.cc/sh` parses shell scripts more reliably than the tree-sitter Bash grammar.

## Decision

- Checks that inspect code structure parse the code into syntax trees.
- The scanner parses code with tree-sitter. The build compiles the tree-sitter C runtime and grammars into a WebAssembly module, and `wazero` executes the module inside the scanner process. The binary stays static and free of cgo, and produces standard tree-sitter syntax trees.
- The scanner parses shell scripts with `mvdan.cc/sh`, which handles shell syntax more reliably than the tree-sitter Bash grammar.
- The build compiles the WebAssembly module from pinned upstream sources instead of tracking compiled binaries in the repository. The build rebuilds the module only when source inputs change.
- Because skill code is untrusted, all parsing operations enforce file size limits and execution timeouts.
- Future checks can use other parsers. A parser with better language support can replace or supplement the current one.

## Consequences

- `go install` cannot build the scanner directly because the repository does not track the compiled WebAssembly module. Users must download release binaries or build from source using the repository build scripts.
- The build requires a C-to-WebAssembly compiler. The initial build on a machine takes longer because it compiles the WebAssembly module.
- `wazero` adds an external dependency outside the standard library. The runtime and embedded module increase binary size and add startup overhead when scans parse code.
- Parsing inside WebAssembly runs slower than native code.
- Upstream fixes in tree-sitter or language grammars require updated dependency pins and a rebuilt WebAssembly module.
- Language coverage depends on the completeness of each grammar.
