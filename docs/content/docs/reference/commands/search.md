---
title: "search"
weight: 30
---

Search design tokens by name, value, or type.

```
Usage:
  asimonim search <query> [files...]

Flags:
  -s, --schema string    Force schema version (draft, v2025.10)
      --name             Search names only
      --value            Search values only
      --type string      Filter by token type
      --regex            Treat query as a regular expression
      --format string    Output format: table, json, names (default "table")
```

## Examples

```bash
# Search by name or value
asimonim search "blue" tokens.json

# Search names only with regex
asimonim search "^color\." tokens.json --name --regex

# Find all dimension tokens containing "spacing"
asimonim search "spacing" tokens.json --type dimension

# Output matching token names only
asimonim search "primary" tokens.json --format names
```
