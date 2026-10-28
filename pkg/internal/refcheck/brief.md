# External references

Facts the scanner collected about the external references of this skill. Domain facts come from RDAP and one HTTPS request to the host root. GitHub facts come from the GitHub API. Package facts come from packages.ecosyste.ms, npm and PyPI. An address has no lookup, its only fact is whether it is public.

A reference with status `failed` or `not-checked` has no facts either way. `derived: true` marks a reference the skill does not name, the scanner reached it by following a redirect. A fact is about the reference itself, `url` gives the full addresses the skill wrote it at, and the lookup never requested those paths.

Each reference has an id such as `R3`, and each place the skill uses it has one such as `R3.1`. A `usage` line under a place says what the skill does with the reference there, as the decision model answered from the lines around the place: `citation`, `documentation`, `user-action`, `tool-server`, `agent-fetch`, `agent-instructions`, `agent-install` or `agent-upload`. A place with no usage line got no clear answer. A package and a derived reference have no usage line.

`tls_validation` is the validation level of the certificate the host presented, and `tls_organization` is the organization a certificate authority verified for an OV or EV certificate. A missing OV or EV certificate shows nothing, a present one can in some cases help confirm that a domain is what it claims to be.

`http_status` is an error the host returned at its root. `site: missing` means the platform itself answered that no site exists under the name.
