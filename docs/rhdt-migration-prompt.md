# RHDT Migration Prompt

Use this prompt in the `red-hat-design-tokens` repository once asimonim features are merged.

---

## Context

This repository currently uses style-dictionary to build design tokens. We're migrating to asimonim, a DTCG-native token toolchain that provides:

- Native W3C DTCG schema support (Draft and v2025.10)
- CSS `light-dark()` function generation for theme tokens
- Multi-output mode (generate all formats in one pass)
- Split-by support (generate per-category files like `color.ts`, `animation.ts`)
- File headers for license/copyright
- Figma-compatible mode extensions

## Migration Tasks

### 1. Create asimonim config file

Create `.config/design-tokens.yaml`:

```yaml
prefix: rh

files:
  - tokens/**/*.yml
  - tokens/**/*.yaml

schema: v2025.10

header: |
  @license
  Copyright 2024 Red Hat, Inc.
  SPDX-License-Identifier: MIT

formats:
  css:
    lightDark:
      enabled: true
      patterns:
        - ["on-light", "on-dark"]

outputs:
  # Monolithic CSS
  - format: css
    path: css/global.css

  # Monolithic CSS for web components (:host)
  - format: css
    path: css/shared.css
    # TODO: needs host flavor support

  # Lit CSS wrapper
  - format: lit-css
    path: css/reset.css.ts

  # Theme CSS with light-dark()
  - format: css
    path: css/default-theme.css
    # lightDark patterns applied

  # SCSS variables
  - format: scss
    path: scss/_variables.scss

  # Flat JSON
  - format: json
    path: json/rhds.tokens.flat.json
    flatten: true

  # Nested DTCG JSON
  - format: dtcg
    path: json/rhds.tokens.json

  # TypeScript Map (monolithic)
  - format: typescript-map
    path: js/tokens.ts

  # TypeScript per-category (split)
  - format: typescript
    path: js/{group}.ts
    splitBy: topLevel

  # VSCode snippets
  - format: vscode-snippets
    path: editor/vscode/snippets.json

  # Sketch palette
  - format: sketch
    path: editor/sketch/rhds.sketchpalette

  # Figma-compatible DTCG with modes
  - format: dtcg
    path: json/figma-tokens.json
    figmaModes:
      patterns:
        - ["on-light", "on-dark"]
```

### 2. Add root-level Figma modes declaration

In the main tokens file or a dedicated `$metadata.json`, add:

```json
{
  "$extensions": {
    "com.figma": {
      "modes": ["on-light", "on-dark"]
    }
  }
}
```

### 3. Update token structure

#### Remove array-based theme values

**Before (style-dictionary pattern):**
```yaml
color:
  brand:
    red:
      $value: ['{color.brand.red.on-light}', '{color.brand.red.on-dark}']
```

**After (asimonim pattern):**
```yaml
color:
  brand:
    red:
      $value: '{color.brand.red.on-light}'  # Default to light
      # asimonim detects on-light/on-dark siblings and generates light-dark()
    red-on-light:
      $value: '#ee0000'
    red-on-dark:
      $value: '#ff3333'
```

The `light-dark()` CSS function and Figma modes are generated automatically from the sibling pattern.

#### Replace `_` group markers with `$root`

**Before:**
```yaml
color:
  _:
    $value: '{color.default}'
```

**After:**
```yaml
color:
  $root:
    $value: '{color.default}'
```

### 4. Update Makefile

Replace style-dictionary commands:

```makefile
.PHONY: tokens
tokens:
	asimonim convert

# Or with explicit files:
tokens:
	asimonim convert tokens/**/*.yml

# Validate tokens
validate:
	asimonim validate --strict tokens/**/*.yml
```

### 5. Remove style-dictionary dependencies

```bash
npm uninstall style-dictionary @tokens-studio/sd-transforms
```

Remove from `package.json`:
- `style-dictionary`
- `@tokens-studio/sd-transforms`
- Any custom transforms/formats/filters

### 6. Delete style-dictionary config files

- `config/` directory with platform configs
- Custom transforms in `lib/`
- Filter definitions
- Format definitions

### 7. Update package.json exports

The per-category TypeScript exports should still work:

```json
{
  "exports": {
    ".": "./js/tokens.js",
    "./animation.js": "./js/animation.js",
    "./border.js": "./js/border.js",
    "./color.js": "./js/color.js",
    ...
  }
}
```

### 8. Features NOT migrated (handle separately)

These style-dictionary features are intentionally not replicated:

| Feature | Recommendation |
|---------|----------------|
| `-rgb`/`-hsl` variant generation | Deprecate; use `rgb(from var(...))` in CSS |
| Post-build `tsc` execution | Move to Makefile |
| Asset file copying | Move to Makefile |
| Description file injection | Handle in docs pipeline |

### 9. Verify outputs

After migration, compare outputs:

```bash
# Generate with asimonim
asimonim convert

# Diff against previous style-dictionary output
diff -r css/ css.old/
diff -r js/ js.old/
diff -r json/ json.old/
```

Expected differences:
- Header comments (now use configured header)
- `light-dark()` function in CSS (new feature)
- Figma `$extensions` in JSON (new feature)

### 10. CI/CD updates

Update GitHub Actions workflow:

```yaml
- name: Build tokens
  run: |
    npm install -g asimonim  # or use binary release
    asimonim convert
    asimonim validate --strict
```

---

## Verification Checklist

- [ ] All CSS outputs generate correctly
- [ ] All TypeScript outputs generate correctly
- [ ] Per-category split files match expected structure
- [ ] `light-dark()` CSS function works in browsers
- [ ] Figma import works with mode extensions
- [ ] VSCode snippets work in editor
- [ ] Sketch palette imports correctly
- [ ] Package exports resolve correctly
- [ ] CI passes with new build

---

## Rollback Plan

Keep style-dictionary config in a `deprecated/` folder until migration is verified in production.
