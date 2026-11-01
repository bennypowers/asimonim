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

## Workspace resolution

The server resolves its workspace in this order: an explicit config, a host
workspace supplied by an embedder, the session's cached resolution, the
client's MCP roots (directly on protocol revisions before `2026-07-28`, and
through multi-round-trip requests on `2026-07-28` and later), then the working
directory.

Every completed tool and resource result carries provenance in `_meta`:

```json
{
  "_meta": {
    "asimonim/workspace": {
      "root": "/path/to/workspace",
      "source": "roots"
    }
  }
}
```

`source` is `explicit-config`, `host`, `roots`, or `cwd`. A `warning` field is
present when the server fell back to the working directory because roots could
not be used.

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
