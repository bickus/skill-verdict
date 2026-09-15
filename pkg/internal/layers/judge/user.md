{{origins}}
## Artifacts
{{artifacts}}

## Rules
Every rule below raised at least one of the findings. A line reads `[id] - title - severity - floor`, and the indented line under it says what the rule reports and how to weigh it. The severity is what the rule gives its findings. The floor is the lowest severity a finding of that rule can end at after your verdict.
{{rulesdowngradeable}}
{{rulesnondowngradeable}}
## Analysis Findings
A line reads `id: file@line:rule: evidence`. The id names the finding, and the rule points at the rule's line above.
{{findingsdowngradeable}}
{{findingsnondowngradeable}}
{{briefs}}
## Entry file
The content of `{{entry}}` follows, as the `read` tool returns it. Do not read it again. Read the other files that carry findings.

{{content}}
