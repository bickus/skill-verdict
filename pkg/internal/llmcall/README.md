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
| `pkg/internal/trace/brief.md` | Start of the `code.brief.md` artifact. |
| `pkg/rules/data/questions.json` | Question templates of the decision model, with their `focus` instruction and options: `who`, `usage`, `goes`, and the guard question `addressed` that the layer adds to every request. |
| `pkg/internal/trace/note.md` | First line of the `trace` text state. |
| `pkg/internal/layers/decision/frame.md` | The block of text around a marked line: the file, its role, the fence, the headings and the numbered lines. |
| `pkg/internal/layers/decision/block.md` | Text state for a question whose subject is `frame`: one `frame.md` block. |
| `pkg/internal/layers/decision/places.md` | Text state for a question whose subject is `places` on a finding of a text rule or of a discovery rule: the name the findings share, then one `place.md` block per finding. |
| `pkg/internal/layers/decision/reference.md`, `place.md`, `place-trace.md` | Text state for the questions about the places of one reference: the reference, its URLs, then one `place.md` block per place with its id, its frame, and the trace when the place is in traced code. |
| `pkg/internal/layers/decision/entry.md` | Text state for a question whose subject is `entry`: the finding, then `SKILL.md` with its frontmatter. |
| `pkg/internal/layers/decision/facts.md` | Text state for a question whose subject is `facts`: the brief lines of the reference, then the frames of up to five places. |

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
| `{{entry}}` | discovery, judge | Path of the entry file, `SKILL.md`. |
| `{{folder}}` | discovery, judge | The name of the skill folder. A skill registry installs the skill under this name. |
| `{{content}}` | discovery, judge | The entry file as the `read` tool returns it, cut at `llm.maxInputChars`. |
| `{{rulesdowngradeable}}`, `{{rulesnondowngradeable}}` | judge | `rules-downgradeable.md` and `rules-nondowngradeable.md` rendered for the findings of that group. Empty when the group has no finding. |
| `{{findingsdowngradeable}}`, `{{findingsnondowngradeable}}` | judge | `findings-downgradeable.md` and `findings-nondowngradeable.md` rendered for the findings of that group. Empty when the group has no finding. |
| `{{rules}}` | discovery system prompt | One block per enabled discovery rule, by ID: `rule_id`, `title` and `description` from the rule data. |
| `{{rules}}` | judge rule blocks | Two lines per rule that raised a finding in the group, first appearance first. The first line reads `[id] - title - severity - floor`. The second is the rule description, indented. The floor reads `none` when the rule declares none and `n/a` when the rule forbids downgrading. |
| `{{findings}}` | judge finding blocks | One line per finding of the group: `id: file@line:rule: evidence`. The id is `D1`, `D2` and so on in the downgradeable group, `ND1`, `ND2` and so on in the other. |
| `{{file}}`, `{{role}}`, `{{fence}}`, `{{section}}`, `{{lines}}` | decision `frame.md` | The path of the file, its role (`entry`, `frontmatter`, `script`, `referenced-by-entry`, `readme` or `other`), the language of the fenced code block that holds the line or `no`, the headings above the line, and the numbered lines of the block with `>>` before the marked line. With no heading above the line, the section line disappears. |
| `{{frame}}` | decision `block.md`, `place.md` | `frame.md` rendered for the line. |
| `{{subject}}`, `{{places}}` | decision `places.md` | The name the grouped findings share, such as a package, and the `place.md` blocks with ids `P1`, `P2` and so on. |
| `{{reference}}`, `{{urls}}`, `{{places}}` | decision `reference.md` | The reference value, one `url:` line per URL, and the `place.md` blocks. Places with the same file, line, text and trace share one block. |
| `{{id}}`, `{{trace}}` | decision `place.md` | The place id such as `R3.1` or `P2`, and `place-trace.md` rendered with the trace. The trace is empty when the place is not in traced code. |
| `{{about}}`, `{{content}}` | decision `entry.md` | The file, line and evidence of the finding, and the numbered lines of `SKILL.md`, cut with a `[cut: N more lines]` line. |
| `{{brief}}`, `{{places}}` | decision `facts.md` | The lines of the reference brief about the reference, and the frames of up to five of its places. |
| `{{cwd}}`, `{{date}}` | honeypot system prompt | The working directory of the agent and the date of the scan. |
| `{{skills}}`, `{{name}}` | honeypot user message | The skill list line, `- name: description`, and the skill name. |
| `{{base}}`, `{{body}}` | honeypot `loaded.md` | The directory of the skill as the agent sees it, and `SKILL.md` without its frontmatter. |
