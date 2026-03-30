---
title: "Configuration"
weight: 15
---

Asimonim reads configuration from `.config/design-tokens.{yaml,yml,json}`:

```yaml
# .config/design-tokens.yaml
prefix: "rh"
resolvers:
  - ./tokens.resolver.json
  - npm:@acme/tokens/tokens.resolver.json
files:
  - ./tokens.json
  - ./tokens/**/*.yaml
  - path: npm:@rhds/tokens/json/rhds.tokens.json
    prefix: rh
groupMarkers: ["_", "@", "DEFAULT"]
schema: draft
cdn: unpkg  # CDN for network fallback (unpkg, esm.sh, esm.run, jspm, jsdelivr)
```

When running commands without file arguments, files from config are used:

```bash
asimonim list      # Uses files from config
asimonim validate  # Uses files from config
```

The language server also reads from `package.json`:

```json
{
  "designTokensLanguageServer": {
    "prefix": "my-ds",
    "tokensFiles": [
      "npm:@my-design-system/tokens/tokens.json",
      {
        "path": "npm:@his-design-system/tokens/tokens.json",
        "prefix": "his-ds",
        "groupMarkers": ["GROUP"]
      },
      {
        "path": "./docs/docs-site-tokens.json",
        "prefix": "docs-site"
      }
    ]
  }
}
```

## Resolvers

The `resolvers` field accepts [DTCG resolver documents](https://www.designtokens.org/tr/2025.10/resolver/) -- JSON files that declare how to compose multiple token files via sets, modifiers, and resolution order. Each entry can be a local path (relative or absolute) or an `npm:`/`jsr:` package specifier.

Resolver documents are distinct from token files -- they reference and orchestrate token files rather than containing tokens directly.

### Auto-Discovery

`DiscoverResolvers` scans the project's root `package.json` and inspects each direct dependency (the `dependencies` map) for resolver files. Only direct dependencies are checked -- `devDependencies`, `peerDependencies`, and transitive dependencies are not scanned.

Dependencies are processed in sorted order for deterministic results. Each dependency's `package.json` is checked for a resolver declaration using either:

**`designTokens` field** (recommended, checked first):
```json
{
  "name": "@acme/tokens",
  "designTokens": {
    "resolver": "tokens.resolver.json"
  }
}
```

**`designTokens` export condition** (fallback):
```json
{
  "name": "@acme/tokens",
  "exports": {
    ".": {
      "designTokens": "./tokens.resolver.json",
      "import": "./dist/index.js"
    }
  }
}
```

When both are present, the `designTokens` field takes priority over the export condition.

## Network Fallback

When using `npm:` specifiers for token packages, Asimonim normally resolves them
from `node_modules`. If the package isn't installed locally, you can enable
**network fallback** to fetch tokens from a CDN (default:
[unpkg.com](https://unpkg.com), configurable via the `cdn` option).

This is opt-in and disabled by default.

### Enable in package.json

```json
{
  "designTokensLanguageServer": {
    "networkFallback": true,
    "networkTimeout": 30,
    "cdn": "unpkg",
    "tokensFiles": [
      "npm:@my-design-system/tokens/tokens.json"
    ]
  }
}
```

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `networkFallback` | `boolean` | `false` | Enable CDN fallback for package specifiers |
| `networkTimeout` | `number` | `30` | Max seconds to wait for CDN requests |
| `cdn` | `string` | `"unpkg"` | CDN provider: `unpkg`, `esm.sh`, `esm.run`, `jspm`, `jsdelivr` |

### Security

- Network fallback is **opt-in** -- it never fetches from the network unless
  explicitly enabled
- Responses are limited to 10 MB to prevent resource exhaustion
- Requests have a configurable timeout (default 30 seconds)
- Only `npm:` specifiers with a file component trigger CDN lookups

## Token Prefixes

The DTCG format does not require a prefix for tokens, but it is recommended to
use a prefix to avoid conflicts with other design systems. If your token files
do not nest all of their tokens under a common prefix, you can pass one yourself
in the `prefix` property of the token file object.

## Group Markers

{{< tip "warning" >}}
Group markers are **only used with Editor's Draft schema**. The 2025.10 stable
specification uses the standardized `$root` reserved token name instead.
{{< /tip >}}

The `groupMarkers` option works around the DTCG draft schema's lack of a
built-in way for a token to also act as a group. For example, with
`groupMarkers: ["_"]`:

```json
{
  "color": {
    "red": {
      "_": {
        "$value": "#FF0000",
        "$description": "Red color"
      },
      "darker": {
        "$value": "#AA0000",
        "$description": "Darker red color"
      }
    }
  }
}
```

This creates tokens: `--color-red` and `--color-red-darker`.

The v2025.10 stable schema uses `$root` instead, so `groupMarkers` is ignored
for that schema version.
