# Command names that runners resolve to another package

- Status: Deferred

## What it is

`npx tsc` runs the local TypeScript compiler when the project has TypeScript installed. In a project without TypeScript, npx fetches the npm package `tsc`. The `tsc` package is an old, deprecated release of the TypeScript compiler. The current compiler is the package `typescript`. A skill that means TypeScript then runs code its author did not choose.

The scanner looks up the name after the runner. For `npx tsc` it checks `tsc` and reads more than 2 million downloads a month. The count makes `tsc` popular. The count does not show which package the skill meant.

Four more runners resolve a command name the same way:

- `bunx`
- `pnpm dlx`
- `yarn dlx`
- `uvx`

## What it needs

- A way to tell that the name after a runner is a command of another package. A manifest in the skill that lists the package with the command is one source.
- A decision on what the finding is when a skill runs a command name that resolves to an unrelated or deprecated package.

## Why it waits

The case needs its own design. The bench has no measured sample of the problem yet.
