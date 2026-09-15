# Layers and rules in configuration

- Date: 2026-09-11
- Status: Accepted

## Context

A scan is a sequence of layers, and each layer runs rules. The operator needs to see and change three things without reading code: whether a layer runs, how the layer is tuned, and what each rule does. A rule has two parts. Its definition, the patterns and scope that find text, belongs to the author. Its settings, the category, the severity, the switch and the review policy, belong to the operator. A name the operator has to guess is a defect.

## Decision

- Every layer is one object in configuration with one shape: its switch, the settings only it uses, and the settings of the rules it runs. A layer the scan cannot run without has no switch.
- Every rule names its layer and its kind. The kind states how certain a finding is: a heuristic pattern match, a model judgement or a verified fact. Neither is a setting.
- Every rule setting is mirrored in configuration under the rule's layer and exact id. Configuration never carries a definition. A category is a name with no fixed list.
- The binary writes the complete default configuration on request. That file is the reference for names and defaults, and it loads unchanged.

## Consequences

- A bulk change lists each id.
- A rule without a layer, a rule setting under the wrong layer, and a setting for an unknown rule are rejected.
- A written configuration names the rules of the version that wrote it. A later version that drops a rule rejects that file until the operator removes the entry.
- Adding a layer or a rule setting means adding it to the configuration shape and the written file in the same change.
