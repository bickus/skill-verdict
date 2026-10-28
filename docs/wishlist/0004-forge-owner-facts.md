# Owner and repository facts for forges other than GitHub

- Status: Deferred

## What it is

The references layer resolves a GitHub URL to its owner and repository and records facts about
them: whether the owner exists, was renamed or is new, whether the repository exists. A URL on
GitLab, SourceHut, Codeberg or Bitbucket is collected as a domain only. The report then carries
facts about `sr.ht` or `gitlab.com`, the host, and nothing about the account and repository the
skill actually trusts.

The static rule `UNPINNED-DOWNLOAD` now matches branch URLs on those hosts. When the judge reads
such a finding, the rule text tells it to lower the severity for a widely used project based on
the reference facts. The only facts present are the host's registration date and certificate,
which say nothing about the project, so the judge keeps the severity every time.

## What it needs

- A collector per forge that extracts owner and repository from the URL shapes of that host:
  `gitlab.com/<group>/<project>`, `git.sr.ht/~<user>/<repo>`, `codeberg.org/<owner>/<repo>`,
  `bitbucket.org/<workspace>/<repo>`.
- A checker per forge against its public API for existence and account age, with the same
  rules as the GitHub checker under forge-neutral identifiers, or the GitHub rule identifiers
  widened to cover any forge.
- The reference kind in the report set to owner and repository for those URLs, so the judge
  does not read the host domain as the subject of interest.

## Why it waits

A branch URL on a forge other than GitHub is caught now and reaches the judge. The cost is a
severity that stays at critical where a known project would have been lowered. Each forge needs
its own API client and a rate-limit policy, and the sample so far is one skill on SourceHut.
