# Reference severity by who can claim the name

- Date: 2026-10-28
- Status: Accepted

## Context

The scanner collects facts about the external references of a skill:

- the registration record of a domain
- the answer at a host root
- the existence of an account or a repository on a code host
- the publication state of a package

Each fact starts a finding at the default severity of its rule.

The facts differ in who can act on them. Three of them are open to anyone:

- a free domain, which anyone can register
- a deleted account name, which anyone can take
- an unused package name, which anyone can publish under

Only the party the skill already names can change the other facts:

- an error at a host root
- a name without a DNS record
- a deleted repository under a living account

With one severity for both kinds, two skills block the same way. One cites an API host whose root answers 404. The other installs from a deleted account.

A name on a hosting platform needs its own fact. The platform answers at the root whether a site exists under the name. An unclaimed name there is as free to take as an unregistered domain. A status code alone does not separate that answer from an API host that serves no page at its root.

The scanner requests the root of a host and the public API of a registry, and nothing else. A path the skill wrote can be a webhook or an attacker's endpoint. Every new fact has to come from the requests the scanner already makes.

## Decision

A reference rule gets a default severity at or above the gate for one kind of fact: an outsider can take the name or the content behind it. An outsider is anyone other than the party the skill names. Four facts are of that kind:

- a domain with no registration, or one that expires soon
- an account on a code host that does not exist or has moved to another name
- a package name nobody has published under
- a name on a hosting platform that the platform reports as unclaimed

A fact that only the named party can change gets a default severity below the gate:

- an error status or a refused connection at a host root
- a name without a DNS record under a registered domain
- a repository that does not exist under an account that does

A rule that raises a finding from the facts about one host counts only the facts with a default severity above low.

The platform's own answer at the host root tells the scanner that a name is unclaimed. The scanner ships a table of platforms and their answers. A status code alone never counts as that answer. The requests stay the host root and the registry APIs, never a path the skill wrote.

## Consequences

- A root error, a missing DNS record and a deleted repository are low findings. The verdict stays below the gate for them. The judge and the reader still see the fact.
- The platform table needs upkeep. A platform can change its answer page. The scanner then reports the low root error finding for its names. The table has to follow.
- The scanner does not detect a subdomain that points at an unclaimed platform name through a DNS alias. Detection needs a DNS lookup the scanner does not make.
- A platform that answers an unclaimed name with no DNS record is not in the table. The scanner cannot tell a name with no address from a name that does not exist. Telling them apart needs a DNS lookup of the scanner's own.
- A deleted repository under a young account gets the same medium finding as any deleted repository. The account's age stays a separate finding.
- An operator can raise the severity of any of these rules in the configuration. ADR 0007 allows that.
