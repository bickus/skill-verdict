# Unpinned installs by tool and by popularity

- Date: 2026-10-30
- Status: Accepted

## Context

A skill can tell the agent to fetch code by name without a fixed version:

- install a package or a container image
- run a package with a tool such as npx
- clone a repository

The rest of this record calls all three an install. The tool fetches the newest release. The owner of the package decides what that release contains. Two kinds of bad release are possible:

- a stranger takes over the account or the release process of a popular package
- the owner of a package with few users publishes one, or a stranger takes over the owner's account

A user can protect a tool against new releases. A minimum release age makes the package manager skip releases younger than a set number of days. A local firewall blocks releases that a malware scanner flagged. Many people install a popular package. They usually find a bad release within hours or days, and the registry removes it. A minimum release age waits longer than that, so it stops the first kind. Nobody watches the releases of a package with few users. A bad release of such a package can stay unnoticed for longer than any waiting period.

Protection differs from tool to tool:

- some tools have a minimum release age, others have none
- npm and pip ship with it turned off
- the scanner never sees the machine that runs the skill

Only the user knows what protects their machine. With one rule for every tool, a user who protects npm has to drop the check for container images too.

## Decision

- An install without a fixed version gets a finding from the rule of its tool. Each tool has its own rule. With a protected tool, the user can lower the rule of that tool or turn it off.
- Each of these fixes the version:
  - an exact version
  - a digest
  - a commit hash
  - a lockfile that the install reads
- The scanner checks each place in the skill for a fixed version. A skill can fix a version in one file and leave it open in another, and the finding points at the open places.
- A second rule finds the unpopular references among installs without a fixed version. Its default severity is at or above the gate. A minimum release age does not protect against an unpopular package, so the rule stays separate from the rule of the tool. The second rule checks every match of the rule of the tool. A rule of a tool that is off still finds its matches for this and records no finding.
- Popularity is a count that the registry or the code host publishes, compared with a line in the configuration. A package counts its downloads over a recent period. A repository counts stars and forks, and both must reach their lines.
- When two sources give a count, the count from the registry or the code host itself wins.
- A reference counts as unpopular unless the scanner knows a count at or above the line. A reference has no count in two cases:
  - the source publishes no count for the reference
  - the scanner has no source for the host
- A lookup that fails leaves the reference unchecked, as ADR 0005 says.
- A tool gets no rule for unpopular references when the scanner has no source for any of its references. Such a rule would find exactly what the rule of the tool finds.

## Consequences

- The scanner drops the rule that covered every tool. A configuration that overrides the old id has to name the new rules. ADR 0013 gives a rule a new id when its meaning changes.
- Without protection, an install of an unpopular reference without a fixed version gets two findings at the gate. The verdict is the same as with one.
- An owner can buy downloads, stars and forks. A line on them stops only an owner who did not inflate them.
- A new package from a known vendor has few downloads at first, and it counts as unpopular. The scanner cannot tell a vendor's package from anyone else's.
- Each package costs one more request, for the count from the registry itself.
- Some tools have no source for counts. For such a tool, its own rule is the only check. With that rule low or off, the scanner accepts popular and unpopular installs alike.
