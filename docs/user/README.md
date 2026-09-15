# skill-verdict Documentation

**skill-verdict** is a security scanner designed specifically for AI agent skills, plugins, and MCP configurations.

Skills aren't just local markdown files. They are instructions that direct AI coding agents on what tools to run, what packages to install, and what remote services to query. A skill doesn't need an overt exploit payload to compromise your workstation: an unpinned dependency, an abandoned GitHub repository, an expired domain, or a prompt injection trick can turn a trusted agent against your system.

This documentation covers everything from running your first scan to fine-tuning the detection engine for your personal workflow or CI/CD pipelines.

---

## Documentation Index

### Getting Started

* **[Installation Guide](install.md)**  
  How to download pre-built binaries, verify checksums, build from source using Go, and choose an operating mode.

* **[Command-Line Usage](usage.md)**  
  How to scan individual skills or skill directories, understand terminal reports, configure output formats, and integrate exit codes into CI/pre-commit pipelines.

### How the Scanner Thinks

* **[Verdicts and Decision Logic](verdicts.md)**  
  Understanding the four verdicts (`clean`, `review`, `incomplete`, `block`), how the Gate works, why unfinished checks cannot be assumed clean, and how the AI Judge eliminates false positives while keeping guardrails intact.

* **[Detection Pipeline](detection.md)**  
  How the scanner inspects skills across all 7 layers: bundle hygiene, text concealment revealing, static heuristics, external references, simulated agent honeypots, semantic discovery, and AI review.

### Threats & Rule Catalog

* **[Rules Reference (Organized by Threat)](rules.md)**  
  The full catalog of built-in security checks grouped by real-world threat: supply chain moving targets, domain hijackings, credential theft, prompt injection, evasion, and opaque binaries.

* **[External References & Supply Chain](references.md)**  
  How external checks work: live lookups for domains (RDAP), GitHub repositories, and package registries (npm, PyPI), network privacy considerations, and OPSEC cautions.

### Customization & Tuning

* **[Configuration Guide](config.md)**  
  How to customize every aspect of the scanner: setting up local or cloud LLMs (OpenRouter, OpenAI, Ollama), fine-tuning scan limits, overriding rule severities, setting AI downgrade floors, and adding custom rules.
