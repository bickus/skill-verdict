# Installation & Setup

**skill-verdict** is distributed as a single static binary with no external runtime dependencies. You do not need Python, Node.js, Docker, or shared system libraries installed.

---

## Download Pre-built Binaries

Pre-compiled static binaries are published for every [release](https://github.com/bickus/skill-verdict/releases):

| Platform | Architecture | Binary File |
|---|---|---|
| Linux | x86-64 | `skill-verdict-linux-amd64` |
| macOS | Apple Silicon (M-series) | `skill-verdict-darwin-arm64` |
| Windows | x86-64 | `skill-verdict-windows-amd64.exe` |

### 1. Download and verify

Every release includes a `SHA256SUMS` file. Verify the integrity of your download:

```sh
sha256sum -c --ignore-missing SHA256SUMS
```

### 2. Add to PATH

Make the binary executable (Linux/macOS) and move it into your `PATH`:

```sh
chmod +x skill-verdict-linux-amd64
sudo mv skill-verdict-linux-amd64 /usr/local/bin/skill-verdict
```

---

## Install with Go

If you have Go 1.26 or later installed, you can compile and install directly to `$GOPATH/bin`:

```sh
go install github.com/bickus/skill-verdict/pkg/cmd/skill-verdict@latest
```

*(And as noted on the README: if you instinctively run `@latest` in production environments, train yourself to pin hashes or versions!)*

---

## Build from Source

Clone the repository and run the repository build script:

```sh
git clone https://github.com/bickus/skill-verdict.git
cd skill-verdict
scripts/check
```

`scripts/check` runs the linter, test suite, and cross-compiles static binaries for all supported platforms into `bin/`:

- `bin/linux-amd64/skill-verdict`
- `bin/darwin-arm64/skill-verdict`
- `bin/windows-amd64/skill-verdict.exe`

Copy the binary for your platform to your preferred `bin` directory.

---

## Verify the Installation

Run:

```sh
skill-verdict --version
```

Official releases print the module version tag (e.g. `v0.1.0`). Binaries compiled from source checkouts print `devel`.

---

## Choosing Your Scan Mode

`skill-verdict` supports three operating modes depending on your privacy requirements, network availability, and whether you want AI review:

| Scan Mode | What It Checks | Requirements | Zero-Cost? |
|---|---|---|---|
| **Offline Mode** | Bundle limits, text concealment, static text rules, symlink checks. | Binary only. No network, no API key. | Yes |
| **Network / No-Model Mode** | Everything in Offline + live domain checks, GitHub existence, npm/PyPI registry checks. | Internet access. Optional `GITHUB_TOKEN`. | Yes |
| **Full AI-Assisted Mode** | Everything in Network + simulated agent honeypot + semantic discovery + AI Judge review. | Internet access, LLM model endpoint, and API key. | LLM token costs |

In `skill-verdict`, scan modes are driven by configuration files, not complicated CLI flags.

### 1. Offline Mode (Air-gapped)

Create an `offline.json` config disabling network references and model layers:

```json
{
  "layers": {
    "references": {"enabled": false},
    "honeypot": {"enabled": false},
    "discovery": {"enabled": false},
    "judge": {"enabled": false}
  }
}
```

Run the scan:

```sh
skill-verdict scan --config offline.json ./path/to/skill
```

Zero network packets will be sent.

### 2. Network / No-Model Mode

If you want to verify external domains, repositories, and package supply chains without paying for LLM tokens, disable only the model layers:

```json
{
  "layers": {
    "honeypot": {"enabled": false},
    "discovery": {"enabled": false},
    "judge": {"enabled": false}
  }
}
```

Run:

```sh
skill-verdict scan --config no-model.json ./path/to/skill
```

### 3. Full AI-Assisted Mode

To catch subtle prompt injections with the simulated honeypot, discover semantic threats, and use the AI Judge to filter out false positives:

1. Generate a configuration file:
   ```sh
   skill-verdict config-template config.json
   ```
2. Configure your model endpoint under `llm`:
   - `llm.model`: model name (e.g. `gpt-4o`, `claude-3-5-sonnet`, `deepseek-chat`).
   - `llm.baseUrl`: API base URL (defaults to OpenRouter, compatible with OpenAI, Ollama, vLLM, etc.).
3. Set your environment variable:
   ```sh
   export SKILL_VERDICT_API_KEY=your-api-key
   ```
4. Run:
   ```sh
   skill-verdict scan --config config.json ./path/to/skill
   ```

---

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `SKILL_VERDICT_API_KEY` | *(empty)* | API key or ChatGPT subscription token used for model queries. Configurable via `llm.keyEnv`. |
| `GITHUB_TOKEN` | *(empty)* | Optional GitHub personal access token (no special scopes needed). Raises GitHub API rate limits from 60 to 5,000 requests/hour when checking external repositories. Configurable via `layers.references.githubTokenEnv`. |

See [Configuration](config.md) for advanced options and [Usage](usage.md) for command-line syntax.
