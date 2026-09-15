Replaces an exact string in a file.

- Read the file in this conversation before you edit it, or the edit fails.
- `old_string` must match the file exactly, indentation included, and occur only once. Include more surrounding lines to make it unique, or set `replace_all` to change every occurrence.
- Do not include the line number prefix from Read output in `old_string` or `new_string`.
- Keep each edit as small as the change allows.
