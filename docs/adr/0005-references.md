# References

- Date: 2026-09-09
- Status: Accepted

## Context

A skill can refer to things outside its files: a domain, a repository owner, a package. A file scan says nothing about them:

- the domain can be a month old and redirect elsewhere
- the owner's account can no longer exist
- the package can be missing from the registry

A check means a request to a server on the network, and the answer can change tomorrow. The skill's author chose which hosts the scanner will ask. Every lookup is therefore a possible attack.

## Decision

- The scanner resolves external references once per run and keeps the results in memory. The results never reach the disk.
- The scanner reports a finding only when it knows the fact:
  - for a package, the registry must say that the package does not exist
  - for a domain, the site must refuse the connection or return an error status
  - a fact that follows from the reference value alone needs no request, such as whether an IP address is public
- A timeout means no answer. The reference stays unchecked and the scan is incomplete.
- A fact that needs judgement, for example a young domain, is a rule like any other. Its finding starts at the rule's severity and the judge lowers it within the rule's policy.
- The scanner writes every collected fact to one brief artifact per skill. The judge reads the brief like any other artifact. No model layer creates reference findings.
- The scanner never connects to a private, loopback, link-local, multicast or unspecified address. The ban covers every hop of every lookup, including redirects.

## Consequences

- Each run stands alone. The same skill can get a different verdict tomorrow. Run the scan again to find out.
- A registry that refuses or throttles the scanner makes the verdict incomplete, never clean.
- A site cannot get a clean verdict by blocking the scanner.
- A reference finding without a judge keeps its rule severity. The operator sets the severity of each reference rule for a run without a model.
- A reference to an internal address stays unchecked. A skill cannot use the scanner to reach the internal network.
