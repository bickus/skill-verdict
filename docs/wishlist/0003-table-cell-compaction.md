# Table cell compaction

- Status: Deferred

## What it is

The compact view joins letter-spaced words: six or more letters each separated by non-letter
characters become one word, so `i g n o r e` matches the same rules as `ignore`. A Markdown table
row of six or more single-letter cells, `| a | b | c | d | e | f |`, has the same shape and is
joined too. A rule whose pattern matches the joined letters then fires on a table that never
spelled the word.

## What it needs

- A separator class that stops at the cell boundary `|`, or a prefilter that skips a match whose
  separators contain the pipe character.
- Nothing for the spaced-security-word prefilter, which needs the letters to spell a listed word.

## Why it waits

Rows of single letters are rare, the joined letters have to spell a rule pattern to produce a
finding, and such a finding is a static match the judge can dismiss. The near-miss test in the
text view package covers a row of five cells; the sixth cell is this entry.
