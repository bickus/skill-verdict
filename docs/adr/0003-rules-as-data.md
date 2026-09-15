# Rules as data

- Date: 2026-09-09
- Status: Accepted

## Context

Detection rules change more often than any other part of the scanner:

- a pattern matches too much
- a new attack family appears
- a deployment needs a rule the defaults do not include

With rules as Go code, each of these needs a code change and a release. A second problem: Go's regular expression engine has no lookaround, so "match X unless Y is on the same line" cannot be one pattern.

## Decision

Every rule has a record in the rule data.

- A rule record has a unique id.
- A pattern rule needs nothing but its record. A rule that code checks, such as a size limit, still takes every setting from its record.
- The binary embeds the default rules. An operator can add a directory of rule files. The scanner merges both by id, later file wins.
- An "unless" case uses separate exclusion patterns. The scanner tests them against the matched line. The scanner never fakes lookaround inside a pattern.
- Every emitted id must exist in the rule set.

## Consequences

- A change to a pattern rule or to any rule setting is a data change only.
- When exclusion patterns cannot express a rule's lookaround, the rule stays out of the set.
- Every layer that emits a finding by id takes severity and policy from the rule set. All layers therefore agree.
- A finding has its rule id. Facts about the rule, such as its title and category, stay in the rule set. A reader of the finding looks them up there.
