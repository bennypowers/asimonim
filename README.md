# Asimonim

![A vintage Israeli phone token (asimon), captured mid-drop as it falls into a payphone coin slot](./logo.png)
A high-performance design tokens parser and validator, available as a CLI tool and Go library.

> *Asimonim* (אֲסִימוֹנִים) (ahh-see-moh-NEEM) is Hebrew for "[tokens](https://www.wikiwand.com/en/articles/Telephone_token)".

Design systems use [design tokens][dtcg] to store visual primitives like colors, spacing, and typography. Asimonim parses and validates token files defined by the Design Tokens Community Group (DTCG) specification, supporting both the current draft and the stable V2025_10 schema.

## Features

- **Multi-schema support**: Handles both Draft and V2025_10 DTCG schemas
- **Automatic schema detection**: Duck-typing detection of schema version from file contents
- **Alias resolution**: Resolve token references with cycle detection
- **Multi-format export**: Convert tokens to TypeScript, SCSS, Swift, Tailwind, XML, and more
- **CSS output**: Generate CSS custom properties from tokens
- **Search**: Find tokens by name, value, or type with regex support
- **Validation**: Check files for schema compliance and circular references

## Installation

### From Source

```bash
go install bennypowers.dev/asimonim@latest
```

## Quick Start

Validate your design token files:

```bash
# Validate token files
asimonim validate tokens.json

# List all tokens
asimonim list tokens.json

# Output as CSS custom properties
asimonim list tokens.json --format css

# Search for color tokens
asimonim search "primary" tokens.json --type color
```

## CLI Reference

### `asimonim validate`

Validate design token files for correctness and schema compliance.

```
Usage:
  asimonim validate [files...]

Flags:
  -s, --schema string    Force schema version (draft, v2025.10)
      --strict           Fail on warnings
      --quiet            Only output errors
```

**Examples:**

```bash
# Validate multiple files
asimonim validate colors.json spacing.json typography.json

# Force a specific schema version
asimonim validate tokens.json --schema v2025.10

# Quiet mode for CI
asimonim validate tokens.json --quiet
```

### `asimonim list`

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

**Examples:**

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

### `asimonim search`

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

**Examples:**

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

### `asimonim convert`

Convert and combine DTCG token files between formats.

```
Usage:
  asimonim convert [files...]

Flags:
  -o, --output string      Output file (default: stdout)
  -f, --format string      Output format (default "dtcg")
  -p, --prefix string      Prefix for output variable names
      --flatten            Flatten to shallow structure (dtcg/json formats only)
  -d, --delimiter string   Delimiter for flattened keys (default "-")
  -s, --schema string      Force output schema version (draft, v2025.10)
  -i, --in-place           Overwrite input files with converted output
```

**Output Formats:**

| Format       | Extension | Description                                        |
| ------------ | --------- | -------------------------------------------------- |
| `dtcg`       | `.json`   | DTCG-compliant JSON (default)                      |
| `json`       | `.json`   | Flat key-value JSON                                |
| `android`    | `.xml`    | Android-style XML resources                        |
| `swift`      | `.swift`  | iOS Swift constants with native SwiftUI Color      |
| `typescript` | `.ts`     | TypeScript ESM module with `as const` exports      |
| `cts`        | `.cts`    | TypeScript CommonJS module with `as const` exports |
| `scss`       | `.scss`   | SCSS variables with kebab-case names               |
| `tailwind`   | `.js`     | Tailwind theme configuration                       |

**Examples:**

```bash
# Flatten tokens to shallow structure
asimonim convert --flatten tokens/*.yaml -o flat.json

# Convert from Editor's Draft to v2025.10 (stable)
asimonim convert --schema v2025.10 tokens.yaml -o stable.json

# In-place schema conversion
asimonim convert --in-place --schema v2025.10 tokens/*.yaml

# Combine multiple files
asimonim convert colors.yaml spacing.yaml -o combined.json

# Generate TypeScript ESM module
asimonim convert --format typescript -o tokens.ts tokens/*.yaml

# Generate TypeScript CommonJS module
asimonim convert --format cts -o tokens.cts tokens/*.yaml

# Generate SCSS variables with prefix
asimonim convert --format scss --prefix rh -o _tokens.scss tokens/*.yaml

# Generate Tailwind theme config
asimonim convert --format tailwind -o tailwind.tokens.js tokens/*.yaml

# Generate Android XML resources
asimonim convert --format android -o values/tokens.xml tokens/*.yaml

# Generate iOS Swift constants
asimonim convert --format swift -o DesignTokens.swift tokens/*.yaml
```

### `asimonim version`

Display version information.

```
Usage:
  asimonim version

Flags:
      --format string    Output format: text, json (default "text")
```

## Configuration

Asimonim reads configuration from `.config/design-tokens.{yaml,yml,json}`:

```yaml
# .config/design-tokens.yaml
prefix: "rh"
files:
  - ./tokens.json
  - ./tokens/**/*.yaml
  - path: npm:@rhds/tokens/json/rhds.tokens.json
    prefix: rh
groupMarkers: ["_", "@", "DEFAULT"]
schema: draft
```

When running commands without file arguments, files from config are used:

```bash
asimonim list      # Uses files from config
asimonim validate  # Uses files from config
```

### Group Markers (Editor's Draft only)

The Editor's Draft schema has no built-in way for a token to also act as a group. The `groupMarkers` option works around this by treating certain token names as group names. For example, with `groupMarkers: ["DEFAULT"]`:

```json
{
  "color": {
    "DEFAULT": { "$value": "#000" },
    "light": { "$value": "#fff" }
  }
}
```

This produces `--prefix-color` (from DEFAULT) and `--prefix-color-light`. The v2025.10 stable schema uses `$root` instead, so `groupMarkers` is ignored for that schema.

This configuration is also consumed by [dtls](https://github.com/bennypowers/design-tokens-language-server) and [cem](https://github.com/bennypowers/cem).

## Schema Versions

Asimonim supports multiple DTCG schema versions:

| Version   | References          | Colors     | Features                    |
| --------- | ------------------- | ---------- | --------------------------- |
| Draft     | `{token.path}`      | Strings    | Group markers               |
| V2025_10  | `$ref: "#/path"`    | Structured | `$extends`, `$root`         |

Schema version is automatically detected from file contents, or can be forced with the `--schema` flag.

## License

GPLv3

[dtcg]: https://design-tokens.github.io/community-group/format/
