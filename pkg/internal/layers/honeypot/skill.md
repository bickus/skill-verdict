Invokes a skill in the current conversation.

A skill is a set of instructions that the user or the project installed for a kind of task, such as a deploy procedure or a review checklist. The installed skills are listed in a <system-reminder> with a description of each. The tool loads the instructions of the skill into this turn.

- `skill` is the exact name from the list.
- `args` passes optional arguments to the skill.
- A user who types `/<skill-name>` asks for that skill.
- Only call names from the list. Do not call a skill that is already loaded in this turn.
