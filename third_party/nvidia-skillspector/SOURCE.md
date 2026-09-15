# NVIDIA SkillSpector

- Origin: https://github.com/NVIDIA/skillspector
- Version: 2.11.1
- Copyright 2026 NVIDIA CORPORATION & AFFILIATES
- License: Apache-2.0, see `LICENSE` in this directory

## Files in this repository carrying material from this source

- `pkg/internal/layers/reveal/data/confusables.json`: `ASCII_CONFUSABLE_SKELETON` from
  `src/skillspector/unicode_confusables.py`.
- `pkg/internal/layers/reveal/data/spaced-words.json`: `_SPACED_SECURITY_WORDS` from
  `src/skillspector/security_reconstruction.py`.
- `pkg/rules/data/exfiltration.json`: E1, E2, E3 and E5 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_data_exfiltration.py`.
- `pkg/rules/data/snooping.json`: AS1 and AS2 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_agent_snooping.py`.
- `pkg/rules/data/supply-chain.json`: SC1, SC2, SC3 and SC7 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_supply_chain.py`.
- `pkg/rules/data/deserialization.json`: DS1, DS2 and DS3 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_deserialization.py`.
- `pkg/rules/data/ssrf.json`: SSRF1 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_ssrf.py`.
- `pkg/rules/data/tool-misuse.json`: TM3 and TM4 patterns from
  `src/skillspector/nodes/analyzers/static_patterns_tool_misuse.py`.
- `pkg/rules/data/semantic.json`: SSD-1 to SSD-4 titles and definitions from
  `src/skillspector/nodes/analyzers/semantic_security_discovery.py`.
- `pkg/internal/refcheck/data/popular-packages.json`: `_POPULAR_PYPI` and `_POPULAR_NPM` from
  `src/skillspector/nodes/analyzers/static_patterns_supply_chain.py`.
- `pkg/internal/llmcall/preamble.md`: the adversarial-input opening of `PER_FILE_ANALYSIS_PROMPT`
  from `src/skillspector/nodes/meta_analyzer.py`.
- `pkg/internal/layers/discovery/system.md`: `ANALYZER_PROMPT` from
  `src/skillspector/nodes/analyzers/semantic_security_discovery.py` and the wrapper text of
  `BASE_ANALYSIS_PROMPT` from `src/skillspector/llm_analyzer_base.py`.
- `pkg/internal/layers/judge/system.md`: the body of `PER_FILE_ANALYSIS_PROMPT` from
  `src/skillspector/nodes/meta_analyzer.py` after its adversarial-input preamble.
