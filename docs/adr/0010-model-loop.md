# Model loop

- Date: 2026-09-11
- Status: Accepted

## Context

A model layer sent one file chunk with one prompt and parsed a JSON object out of the reply. The model saw one chunk of one file and nothing else of the skill. A reply that was not a bare JSON object was a failed call. Reasoning models carry state between turns, and a subscription backend speaks another wire protocol than a hosted router. The one request, one reply design fits none of that.

## Decision

- A model layer runs a loop. The loop starts with a system prompt and one user message. The model answers with tool calls, the loop answers with tool results, until the model calls the tool that ends the task. That tool's argument is the layer's result. A reply without tool calls gets a reminder and another turn.
- A layer defines the tools it gives the model, its prompts and its reminder, and marks the tool that ends the task. The loop takes them as they come and never singles out a tool by name.
- The model reads skill content through a tool that serves artifacts from the inventory. The tool never touches the file system.
- A provider adapter returns the assistant reply with its native wire form attached. The adapter sends that form back unchanged on the next request. Earlier messages are never rewritten.
- Each wire protocol is a provider adapter, and configuration picks the protocol. The API key comes from the environment or from a file. The configuration never contains the key.

## Consequences

- A layer's result is structured by construction. Free text from the model is never parsed.
- A new layer brings its own tools and prompt files and changes nothing shared.
- A loop costs as many requests as the model needs to read.
- The adapter, not the loop, knows how a provider stores reasoning between turns. A new provider is one adapter.
- A reply the adapter cannot read is a failed request, never an empty success.
