# Dangling aliases to unclaimed platform names

- Status: Deferred

## What it is

A skill can name a host on a domain of its own, such as `docs.example.com`, whose DNS record is an alias of a name on a hosting platform. The platform name behind the alias can be unclaimed. Anyone with an account on the platform can then create that name and serve content under the host the skill trusts.

The scanner checks a platform name only when the skill writes that name itself. A host behind an alias gets the low `DOMAIN-HTTP-ERROR` or `DOMAIN-UNREACHABLE` today. ADR 0015 records the gap.

## What it needs

- A CNAME lookup for every domain reference, so the scanner learns the platform name behind an alias.
- A lookup that tells a name without an address from a name that does not exist. With it, four more platforms join the table: `azurewebsites.net`, `fly.dev`, `pages.dev` and `workers.dev`. They answer an unclaimed name with no DNS record.
- The platform table of `PLATFORM-SITE-MISSING`, applied to the name the lookup returns.
- A fact in the reference brief with the alias target, so the judge sees which platform name the finding is about.

## Why it waits

There is no sample of a skill that uses such an alias. The scanner sends no DNS query of its own today. A name lookup happens only inside the connection to the host root, and a CNAME lookup is a new kind of request that needs a decision record first.
