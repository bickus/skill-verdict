# Model prompts

Every text the model reads is a markdown or JSON file next to the code that sends it. The
binary embeds the files. Edit a file, rebuild, done.

| File | Sent as |
|---|---|
| `pkg/internal/llmcall/preamble.md` | Start of the system prompt of discovery and judge. |
| `pkg/internal/llmcall/origins.md` | Section naming the scanner steps that made artifacts, when any artifact is not from the bundle. |
| `pkg/internal/llmcall/read.md`, `read.json` | Description and parameter schema of the `read` tool. |
| `pkg/internal/layers/<layer>/system.md` | System prompt of the layer, after the preamble. |
| `pkg/internal/layers/<layer>/user.md` | First user message of the layer. |
| `pkg/internal/layers/<layer>/reminder.md` | User message sent when a reply contains no tool call. |
| `pkg/internal/layers/<layer>/item.json` | Schema of one entry the layer's tools take. |
| `pkg/internal/layers/discovery/finding.md`, `action.md`, `submit.md` | Descriptions of the `finding` tool, of its `action` field and of `submit_result`. |
| `pkg/internal/layers/judge/verdict.md`, `submit.md` | Descriptions of the `verdict` tool and of `submit_result`. |
| `pkg/internal/layers/judge/rules-*.md`, `findings-*.md` | One block of the judge user message. A block whose group has no finding is left out. |
| `pkg/internal/layers/honeypot/system.md` | Whole system prompt of the honeypot, with no preamble. |
| `pkg/internal/layers/honeypot/<tool>.md`, `<tool>.json` | Description and parameter schema of each honeypot tool: `agent`, `bash`, `edit`, `read`, `skill`, `webfetch`, `websearch`, `write` and `submit` for `submit_result`. |
| `pkg/internal/layers/honeypot/loaded.md` | What the honeypot `Skill` tool returns. |
| `pkg/internal/refcheck/brief.md` | Start of the `references.brief.md` artifact. |

A layer builds its tool parameters from `item.json`. Discovery adds the `action` field to it
and takes one entry per call. The judge fills the `verdict` field with the severities the
scanner knows and takes a list of entries.

HTML comments (`<!-- -->`) are stripped before sending. A file with borrowed text starts with
the notice `third_party/README.md` asks for, in such a comment.

## Placeholders

A prompt file names its dynamic parts as `{{name}}`. The code fills every placeholder and nothing
else. A placeholder the code does not fill, or a value the file does not use, stops the program
at startup. A placeholder alone on its line disappears with the line when its value is empty.
The values:

| Placeholder | Layers | Value |
|---|---|---|
| `{{origins}}` | discovery, judge | `origins.md` rendered with one line per origin present, `- name: description` from the origin registry. Empty when every artifact is from the bundle. |
| `{{artifacts}}` | discovery, judge | One line per artifact, paths in case-insensitive A-Z order: `path - type, size bytes[, origin x][, not read: status]`. |
| `{{entry}}` | discovery, judge | Path of the entry file: `SKILL.md.revealed` when the reveal layer made one, else `SKILL.md`. |
| `{{content}}` | discovery, judge | The entry file as the `read` tool returns it, cut at `llm.maxInputChars`. |
| `{{rulesdowngradeable}}`, `{{rulesnondowngradeable}}` | judge | `rules-downgradeable.md` and `rules-nondowngradeable.md` rendered for the findings of that group. Empty when the group has no finding. |
| `{{findingsdowngradeable}}`, `{{findingsnondowngradeable}}` | judge | `findings-downgradeable.md` and `findings-nondowngradeable.md` rendered for the findings of that group. Empty when the group has no finding. |
| `{{rules}}` | discovery system prompt | One block per enabled discovery rule, by ID: `rule_id`, `title` and `description` from the rule data. |
| `{{rules}}` | judge rule blocks | Two lines per rule that raised a finding in the group, first appearance first. The first line reads `[id] - title - severity - floor`. The second is the rule description, indented. The floor reads `none` when the rule declares none and `n/a` when the rule forbids downgrading. |
| `{{findings}}` | judge finding blocks | One line per finding of the group: `id: file@line:rule: evidence`. The id is `D1`, `D2` and so on in the downgradeable group, `ND1`, `ND2` and so on in the other. |
| `{{cwd}}`, `{{date}}` | honeypot system prompt | The working directory of the agent and the date of the scan. |
| `{{skills}}`, `{{name}}` | honeypot user message | The skill list line, `- name: description`, and the skill name. |
| `{{base}}`, `{{body}}` | honeypot `loaded.md` | The directory of the skill as the agent sees it, and `SKILL.md` without its frontmatter. |
