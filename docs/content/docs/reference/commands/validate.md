---
title: "validate"
weight: 10
---

Validate design token files for correctness and schema compliance.

```
Usage:
  asimonim validate [files...]

Flags:
  -s, --schema string    Force schema version (draft, v2025.10)
      --strict           Fail on warnings
      --quiet            Only output errors
```

## Examples

```bash
# Validate multiple files
asimonim validate colors.json spacing.json typography.json

# Force a specific schema version
asimonim validate tokens.json --schema v2025.10

# Quiet mode for CI
asimonim validate tokens.json --quiet
```
