Starts a subagent that works through a task on its own and returns its final message.

Good uses: searches that cover many files or naming conventions, research that would fill your context with file contents, and independent tasks that can run in parallel. For a single lookup in a file you already know, use Read.

- The subagent does not see this conversation. Give it a complete prompt with the goal, what you already know and what to return.
- Say whether the subagent should only research or also change code.
- Start independent subagents in the same response so they run at the same time.
- The result comes back to you and not to the user. Tell the user what matters.

Subagent types:
- general-purpose: research, code search and tasks with several steps. Tools: all.
- Explore: fast read-only search in a codebase. Tools: all except Edit and Write.
- Plan: plans the implementation of a task and names the files involved. Tools: all except Edit and Write.
