# Install

skill-verdict is a single static binary. It does not need a runtime or shared libraries.

## Download a release

Every [release](https://github.com/bickus/skill-verdict/releases) has a binary for Linux x86-64, Windows x86-64 and macOS Apple silicon, and a `SHA256SUMS` file. Check the hash:

    sha256sum -c --ignore-missing SHA256SUMS

Rename the file to `skill-verdict` and move it to a directory on `PATH`.

## Install with Go

Go 1.26 or later is required:

    go install github.com/bickus/skill-verdict/pkg/cmd/skill-verdict@latest

## Build from a checkout

Run the repository check:

    scripts/check

It checks the source and builds these binaries:

| Platform | File |
|---|---|
| Linux x86-64 | `bin/linux-amd64/skill-verdict` |
| Windows x86-64 | `bin/windows-amd64/skill-verdict.exe` |
| macOS Apple silicon | `bin/darwin-arm64/skill-verdict` |

Move the binary for your platform to a directory on `PATH`.

## Check the installation

    skill-verdict --version

## Choose a scan mode

| Mode | Command | Requirements |
|---|---|---|
| Offline | `skill-verdict scan --config offline.json PATH` | Only the binary and the skill files. |
| No model | `skill-verdict scan --config no-model.json PATH` | Network access for domain, GitHub, npm and PyPI checks. |
| Full | `skill-verdict scan PATH` | Network access, a configured model and its API key. |

A mode is a configuration file, not a flag. Turn the model off with
`"enabled": false` on `honeypot`, `discovery` and `judge` under `layers`, and on
`references` as well to make no network requests at all. See the offline example in the
[README](../../README.md#run-a-scan). The no-model mode still checks external references.

For a full scan, set the model name and API endpoint in the
[configuration](config.md#model-settings). Put the API key in the environment variable named by
`llm.keyEnv`, which is `SKILL_VERDICT_API_KEY` by default. The configuration file never contains
the key itself.

See [External references](references.md) for the services contacted during a network scan.
