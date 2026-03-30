---
title: "mcp"
weight: 50
---

Launch a Model Context Protocol (MCP) server for AI-assisted development with
design tokens. The server communicates over stdin/stdout using JSON-RPC.

```text
Usage:
  asimonim mcp
```

The MCP server discovers tokens from:
- Local token files specified in `.config/design-tokens.yaml`
- npm/jsr dependencies with `designTokens` field or export condition
- Resolver documents referenced in config

## Tools

| Tool | Description |
| ---- | ----------- |
| `validate_tokens` | Validate token files for correctness, detect circular references, report deprecated tokens |
| `search_tokens` | Search tokens by name, value, description, or type with regex support |
| `convert_tokens` | Convert tokens to CSS, SCSS, JavaScript, Swift, Android XML, or other formats |

## Resources

| URI | Description |
| --- | ----------- |
| `asimonim://tokens` | List available token sources with counts |
| `asimonim://tokens/{source}` | All tokens from a specific source |
| `asimonim://token/{source}/{path}` | Individual token detail |
| `asimonim://config` | Workspace configuration |

## Example (Claude Code `settings.json`)

```json
{
  "mcpServers": {
    "asimonim": {
      "command": "asimonim",
      "args": ["mcp"]
    }
  }
}
```
