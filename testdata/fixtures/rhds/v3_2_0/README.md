# RHDS tokens 3.2.0 normalized DTCG fixture

This fixture is a curated, normalized DTCG document derived from the published
`@rhds/tokens@3.2.0` file `json/rhds.tokens.json`. The published file is Style
Dictionary output, not canonical DTCG.

## Provenance

- Package: `@rhds/tokens`
- Version: `3.2.0`
- Source file: `json/rhds.tokens.json`
- Tarball integrity:
  `sha512-YyeRPr2ThFGclyky4PCruj9f7e15yP2xnT3ZTJeU64+k5ICb/YezaKjJs8Dc48WHDhINHCgMt+VecfM86GExuQ==`
- Normalized schema: draft. The document deliberately omits `$schema`, so
  asimonim auto-detects the version when `parser.Options.SchemaVersion` is the
  zero value (`schema.Unknown`). Curly-brace aliases and string color values
  select the draft schema.

The same provenance and counts are stored in the fixture's root
`$extensions["com.rhds.mcp"]`.

## Normalization

- `$value` comes from `original.$value` when present. The Style Dictionary
  resolved value is discarded so aliases survive.
- `$type` is kept when present. The 18 source tokens without a type are
  recovered with an explicit map, applied to `attributes.type` and then
  `attributes.category`:
  - `radius` -> `dimension`
  - `letter-spacing` -> `dimension`
  - `line-height` -> `number`
  - `opacity` -> `number`
- `$description`, `$deprecated`, `$deprecationMessage`, and `$extensions` are
  kept. Version 3.2.0 uses string `$deprecated` values and has no
  `$deprecationMessage`.
- Group and token `$extensions` are kept at their original paths, including
  `com.redhat.ux` and `com.redhat.print`.
- Style Dictionary fields (`name`, `attributes`, `filePath`, `isSource`,
  `original`, `path`) are dropped rather than moved under
  `$extensions["com.style-dictionary"]`.
- Generated `-rgb` and `-hsl` color variants are kept as color tokens. Their
  original values alias the base color, and they are deprecated in 3.2.0.

This is a subset chosen to cover every `$type` and every structure in the
normalization contract: aliases, a three-hop alias chain, composite shadow,
font family array, media query map, cubic bezier array, duration, descriptions,
deprecations, token extensions, and group extensions.

## Generated tokens

The published 3.2.0 document has 547 tokens.

- 160 `-rgb` and `-hsl` tokens are generated and do not exist in the source
  YAML. They are retained in normalized DTCG as aliases, hence
  `generatedColorAliases: 160` in the provenance.
- 57 `light-dark(var(...))` composite theme tokens are generated CSS
  expressions, not DTCG color values. Their source form is an array of two
  per-theme aliases. They are excluded from this DTCG fixture, hence
  `excludedLightDark: 57`. The `@rhds/mcp` data generator keeps them in a
  separate computed theme map.
