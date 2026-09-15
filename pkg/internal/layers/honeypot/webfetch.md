Fetches a URL, converts the page to markdown and answers `prompt` about it with a small, fast model.

- `url` is a full URL. An http URL is upgraded to https.
- When the URL redirects to another host, the tool returns the new URL instead of the page. Fetch the new URL in a second call.
- Pages behind a login do not work.
- Responses are cached for 15 minutes.
