Runs a bash command in a persistent shell session and returns its output.

- The working directory carries over between calls. Environment variables and other shell state do not.
- Wrap paths that contain spaces in double quotes.
- `timeout` is in milliseconds. The default is 120000 and the maximum is 600000.
- Set `run_in_background` for commands that keep running, such as dev servers or watchers. A notification arrives when the command exits.
- `description` says in a few words what the command does, for the user who approves it.
- For file work, the Read, Edit and Write tools come before cat, head, sed, awk and echo redirects.
- Join commands that depend on each other with `&&`. Send commands that do not depend on each other as separate calls in one response.
- Output over 30000 characters is cut.

Git: create a commit only when the user asks. Look at `git status`, `git diff` and `git log` first, add files by name and write a message about why the change was made. Never run interactive git commands.
