# Decisions

| ADR | Decision |
|---|---|
| [0001](0001-library-surface.md) | `scan`, `rules`, `config`, `finding`, `refs` and the `llm` packages are the supported API; everything else is internal. |
| [0002](0002-borrowed-material.md) | Borrowed data goes in attributed data files. Borrowed algorithms get a new specification and implementation. |
| [0003](0003-rules-as-data.md) | The scanner merges rules from embedded defaults and an optional directory. Negation uses a separate exclusion pattern. A finding names its rule by id, and facts about the rule stay in the rule set. |
| [0004](0004-verdict.md) | The verdict follows finding severity against a configured gate and inspection completeness, not a score. |
| [0005](0005-references.md) | External references resolve once per run in memory. A finding needs a known fact, from a registry, from the site or from the reference value alone. Every fact goes to one brief the judge reads. Reference rules are rules like any other. The scanner never dials a private address. |
| [0006](0006-judge-downgrade.md) | The model may lower a finding down to the floor its rule declares. The model may dismiss a finding only when there is no floor. |
| [0007](0007-configuration-authority.md) | Configuration can change every shipped default. The scanner rejects a configuration only when it cannot apply it. |
| [0008](0008-layers-and-rules-in-configuration.md) | Every layer is one configuration object with its switch, its settings and its rules. Every rule names its layer and kind. The binary writes the complete default configuration. |
| [0009](0009-artifact-model.md) | Every file and every derived item is an artifact with a kind, a registered origin, a content type, a read status and notes. Layers select on those fields. |
| [0010](0010-model-loop.md) | A model layer runs a tool loop with its own tools and ends with the tool it marks final. Adapters replay their native reply form. Configuration never contains the API key. |
| [0011](0011-sample-collections-in-containers.md) | A container image fetches each collection of malicious samples at a pinned commit. Scans of the collection run inside a rootless container. No sample file reaches the host or this repository. |
| [0012](0012-walk-layer-and-interrupting-rules.md) | Reading the files is a layer with no switch that never unpacks a container. Its limits are rules with findings. Every rule has an `interrupt` setting, and an interrupting finding stops the scan through one recording point. |
| [0013](0013-rule-identifiers.md) | A rule id is uppercase hyphenated English, subject first, then condition, with one word per condition across the set. It carries no layer, category, kind or number. |
