---
title: "list"
weight: 20
---

List all tokens from design token files with optional filtering and formatting.

```
Usage:
  asimonim list [files...]

Flags:
  -s, --schema string    Force schema version (draft, v2025.10)
      --type string      Filter by token type
      --resolved         Show resolved values (follow aliases)
      --format string    Output format: table, json, css (default "table")
      --css              Shorthand for --format css
```

## Examples

```bash
# List all tokens as a table
asimonim list tokens.json

# Output as JSON
asimonim list tokens.json --format json

# Generate CSS custom properties
asimonim list tokens.json --format css

# Show only color tokens with resolved values
asimonim list tokens.json --type color --resolved
```
