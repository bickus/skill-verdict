# Declared-marker reconstruction

- Status: Deferred

## What it does

A skill can conceal an instruction by declaring a marker and asking the reader to undo it:
"remove every `#` from the following", "replace `x-` with nothing", "the tags are noise", or by
HTML-entity encoding the text. NVIDIA SkillSpector's `security_reconstruction.py` parses such
declarations, applies them and matches the reconstructed text. skill-verdict's derived views undo
Unicode confusables, default-ignorable code points, compatibility forms and letter spacing, not
declared markers.

## What it needs

- Verb tables: the verbs and phrasings that declare a marker (remove, strip, delete, ignore, and
  their forms) and their negations.
- Quote, tag and HTML-entity decoding for the marker and the text around it.
- A bounded negation check before the declaration ("do not remove"), which RE2 cannot express as a
  lookbehind and needs its own code.
- A dynamic-programming word matcher that decides whether a candidate reconstruction spells a
  security word within a bounded edit budget.
- Roughly fifteen bounded constants: maximum marker length, maximum declaration distance, maximum
  candidates, maximum reconstructed length and the like.
- Source position mapping, as the existing views have, so a finding points at the original line.

## Why it waits

The parser is intricate and hard to fit the 400-line file and 60-line function caps without
fragmenting it. No incident in the corpus the design was checked against turns on it: the
concealment seen in the wild is Unicode and spacing, both handled. When a sample needs it, this
entry says what to build.
