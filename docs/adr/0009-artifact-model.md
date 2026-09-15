# Artifact model

- Date: 2026-09-11
- Status: Accepted

## Context

The walk reads files. Layers derive more items from them. The report has to say for each file whether the scanner read it, and the reason when it did not.

Three things were wrong:

- A layer that wanted text had to know which other layers produce items. To pick the right ones it named their origins.
- The list of read outcomes existed for the report. Layers read the same list to decide what to process.
- A file the walk could not read was not an item at all. Only the report list knew about it.

## Decision

Every file the walk meets is an artifact, and every item a layer derives is an artifact. An artifact has:

- a kind, the purpose of the artifact, such as skill text or a brief for a reader
- an origin, the part of the scanner that made the artifact. A layer that creates artifacts registers its origin with a name and a short description. Only that layer uses the origin.
- a content type, what the walk found, such as text, a media file or a link
- a read status, whether the walk got the item. The walk sets the status once, and a derived artifact has none.
- notes, free text for the reader. Every layer may add a note. The report prints the notes and nothing decides on them.

A layer selects artifacts by these fields and nothing else.

## Consequences

- A new layer that produces artifacts changes no other layer.
- A reader of the registry sees every origin the scanner can produce with a description.
- The report shows every file the walk met. A file the walk did not read has the reason in its notes.
- A layer cannot mark a file as read or unread. Only the walk knows that.
