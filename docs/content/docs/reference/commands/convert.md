---
title: "convert"
weight: 40
---

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

## Output Formats

| Format       | Extension          | Description                                        |
| ------------ | ------------------ | -------------------------------------------------- |
| `dtcg`       | `.json`            | DTCG-compliant JSON (default)                      |
| `json`       | `.json`            | Flat key-value JSON                                |
| `android`    | `.xml`             | Android-style XML resources                        |
| `swift`      | `.swift`           | iOS Swift constants with native SwiftUI Color      |
| `js`         | `.ts`, `.js`, `.cts`, `.cjs` | JavaScript/TypeScript (see JS options below) |
| `scss`       | `.scss`            | SCSS variables with kebab-case names               |
| `css`        | `.css`             | CSS custom properties                              |
| `snippets`   | `.code-snippets`, `.tmSnippet`, `.json` | Editor snippets (VSCode, TextMate, or Zed) |

## JS Format Options

| Flag           | Values                | Default   | Description                              |
| -------------- | --------------------- | --------- | ---------------------------------------- |
| `--js-module`  | `esm`, `cjs`          | `esm`     | Module system (ESM or CommonJS)          |
| `--js-types`   | `ts`, `jsdoc`         | `ts`      | Type system (TypeScript or JSDoc)        |
| `--js-export`  | `values`, `map`       | `values`  | Export form (simple values or TokenMap)  |

## Examples

```bash
# Flatten tokens to shallow structure
asimonim convert --flatten tokens/*.yaml -o flat.json

# Convert from Editor's Draft to v2025.10 (stable)
asimonim convert --schema v2025.10 tokens.yaml -o stable.json

# In-place schema conversion
asimonim convert --in-place --schema v2025.10 tokens/*.yaml

# Combine multiple files
asimonim convert colors.yaml spacing.yaml -o combined.json

# Generate TypeScript ESM module (default JS output)
asimonim convert --format js -o tokens.ts tokens/*.yaml

# Generate TypeScript CommonJS module
asimonim convert --format js --js-module cjs -o tokens.cts tokens/*.yaml

# Generate JavaScript with JSDoc types
asimonim convert --format js --js-types jsdoc -o tokens.js tokens/*.yaml

# Generate TokenMap class for typed token access
asimonim convert --format js --js-export map -o tokens.ts tokens/*.yaml

# Generate SCSS variables with prefix
asimonim convert --format scss --prefix rh -o _tokens.scss tokens/*.yaml

# Generate Android XML resources
asimonim convert --format android -o values/tokens.xml tokens/*.yaml

# Generate iOS Swift constants
asimonim convert --format swift -o DesignTokens.swift tokens/*.yaml

# Generate CSS custom properties
asimonim convert --format css -o tokens.css tokens/*.yaml

# Generate CSS with :host selector (for shadow DOM)
asimonim convert --format css --css-selector :host -o tokens.css tokens/*.yaml

# Generate Lit CSS module
asimonim convert --format css --css-module lit -o tokens.css.ts tokens/*.yaml

# Generate VSCode snippets
asimonim convert --format snippets -o tokens.code-snippets tokens/*.yaml

# Generate TextMate snippets
asimonim convert --format snippets --snippet-type textmate -o tokens.tmSnippet tokens/*.yaml

# Generate Zed editor snippets
asimonim convert --format snippets --snippet-type zed -o css.json tokens/*.yaml
```

## CSS Output

The `css` format generates CSS custom properties from tokens:

```bash
asimonim convert --format css -o tokens.css tokens/*.yaml
```

**Options:**

| Flag             | Default  | Description                                      |
| ---------------- | -------- | ------------------------------------------------ |
| `--css-selector` | `:root`  | CSS selector wrapping properties (`:root`, `:host`) |
| `--css-module`   | (none)   | JavaScript module wrapper (`lit` for Lit CSS)   |

```bash
# Shadow DOM components
asimonim convert --format css --css-selector :host -o tokens.css tokens/*.yaml

# Lit CSS tagged template literal
asimonim convert --format css --css-module lit -o tokens.css.ts tokens/*.yaml
```

## Editor Snippets

The `snippets` format generates editor snippets for autocompleting CSS custom properties:

```bash
asimonim convert --format snippets -o tokens.code-snippets tokens/*.yaml
```

**Snippet Types:**

| Type       | Extension         | Description                              |
| ---------- | ----------------- | ---------------------------------------- |
| `vscode`   | `.code-snippets`  | VSCode/compatible editors (default)      |
| `textmate` | `.tmSnippet`      | TextMate/Sublime Text plist format       |
| `zed`      | `.json`           | Zed editor snippets                      |

Use `--snippet-type` to select the output format:

```bash
# VSCode snippets (default)
asimonim convert --format snippets -o tokens.code-snippets tokens/*.yaml

# TextMate snippets
asimonim convert --format snippets --snippet-type textmate -o tokens.tmSnippet tokens/*.yaml

# Zed editor snippets
asimonim convert --format snippets --snippet-type zed -o css.json tokens/*.yaml
```
