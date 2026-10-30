---
name: toolserver
description: Fixture for the rule that finds a remote tool server.
---
claude mcp add --transport http linear https://mcp.linear.app/mcp
codex mcp add notion --url https://mcp.notion.com/mcp
claude mcp add files -- npx -y @modelcontextprotocol/server-filesystem@1.0.0
npx -y mcp-remote@0.1.29 https://mcp.example.com/sse
See https://modelcontextprotocol.io for how MCP works.
