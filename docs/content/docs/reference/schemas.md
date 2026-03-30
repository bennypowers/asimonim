---
title: "Schema Versions"
weight: 20
---

Asimonim supports multiple DTCG schema versions:

| Version   | References                         | Colors     | Features                    |
| --------- | ---------------------------------- | ---------- | --------------------------- |
| Draft     | `{token.path}`                     | Strings    | Group markers               |
| v2025.10  | `{token.path}` or `$ref: "#/path"` | Structured | `$extends`, `$root`         |

Schema version is automatically detected from file contents, or can be forced with the `--schema` flag.

## Editor's Draft

The original DTCG format:

- String color values (hex, rgb, hsl, named colors)
- Curly brace references: `{color.brand.primary}`
- Group markers for root tokens: `_`, `@`, `DEFAULT`

See [Editor's Draft specification][editorsdraft].

## 2025.10 Stable

The latest stable specification:

- Structured color values with 14 color spaces (sRGB, oklch, display-p3, etc.)
- JSON Pointer references: `$ref: "#/color/brand/primary"`
- Group inheritance: `$extends: "#/baseColors"`
- Standardized `$root` token for root-level tokens
- All draft features (backward compatible)

See [2025.10 specification][202510stable].

## Multi-Schema Workspaces

Asimonim can load multiple token files with different schema versions simultaneously:

```json
{
  "designTokensLanguageServer": {
    "tokensFiles": [
      "legacy/draft-tokens.json",
      "design-system/tokens.json"
    ]
  }
}
```

Schema version detection priority:
1. `$schema` field in the token file (recommended)
2. Per-file `schemaVersion` config in `package.json`
3. Duck-typing based on features (structured colors, `$ref`, `$extends`)
4. Defaults to Editor's Draft for ambiguous files

[editorsdraft]: https://second-editors-draft.tr.designtokens.org/format/
[202510stable]: https://www.designtokens.org/tr/2025.10/
