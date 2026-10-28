# Reference fact rules that read the usage

- Status: Deferred

## What it is

The discovery layer records a usage for every place a skill names a reference:

- a citation
- documentation for the reader
- a step the user does by hand
- data the agent fetches
- instructions the agent follows
- code the agent installs or runs
- an upload the agent sends

The reference brief shows the usage to the judge. The judge weighs a fact finding such as `DOMAIN-NEW` or `GITHUB-OWNER-NEW` against that usage.

The judge instruction of those fact rules asks whether the agent acts on the reference. The usage answers that question. The scanner drops a fact finding on a cited reference to the floor with no judge call.

## What it needs

- A field on the fact rules that names the usages the finding applies to, or a floor per usage.
- A step after discovery that applies the field to each reference finding. The step records the judgement in the same shape the judge uses.
- A rule for a reference with mixed usages, cited in one place and installed from in another. The strongest usage decides.

## Why it waits

The judge does this today with the brief in front of it. The change saves judge tokens and one class of judge mistakes. The change also adds a second place that defines what a fact rule means. The classification first needs a few weeks of bench runs to show its precision.
