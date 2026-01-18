# asimonim Research Notes

This document preserves research findings across context windows.

## Design Decisions

- **npm/jsr specifiers**: Supported from initial release (port from cem)
- **Terminal colors**: 24-bit truecolor only (modern terminals)
- **LSP positions**: Included in Token type from start (for dtls integration)

## Source Projects

### cem (bennypowers.dev/cem)

**Design tokens code location:** `/home/bennyp/Developer/cem/designtokens/designtokens.go` (~309 lines)

**Key functions to port:**
- `flattenTokens()` - DTCG flattening with $type inheritance
- `readJSONFileOrSpecifier()` - Load from local/npm/jsr
- `parseNpmSpecifier()` - Parse `npm:@scope/pkg/path/file.json`
- `dtcgTypeToCSS()` - Map DTCG types to CSS syntax
- `toTokenResult()` - Convert raw data to TokenResult

**Types:** `/home/bennyp/Developer/cem/types/designtokens.go` (~49 lines)
```go
type TokenResult interface {
    Value() any
    Description() string
    Syntax() string
}

type DesignTokens interface {
    Get(name string) (TokenResult, bool)
}
```

**Integration points:**
- `cmd/generate.go` - --design-tokens, --design-tokens-prefix flags
- `cmd/config/config.go` - DesignTokensConfig struct
- `workspace/workspace.go` - designTokensCacheImpl
- `generate/generate.go` - preprocess loads tokens

### design-tokens-language-server (bennypowers.dev/dtls)

**Key packages:**

1. **schema/** - Multi-version support
   - `version.go` - SchemaVersion enum (Draft, V2025_10)
   - `handler.go` - SchemaHandler interface
   - `registry.go` - Handler registry pattern
   - `detector.go` - DetectVersion with duck-typing

2. **tokens/** - Token data structures
   - `types.go` - Token struct with full metadata (~50 fields)
   - `manager.go` - Composite key storage `"filePath:tokenName"`

3. **resolver/** - Token resolution
   - `graph.go` - DependencyGraph, cycle detection, topological sort
   - `aliases.go` - Curly brace + JSON pointer resolution
   - `extends.go` - $extends inheritance (2025.10 only)

4. **parser/** - Parsing pipeline
   - `json/parser.go` (~668 lines) - JSON parsing with yaml.v3 for positions
   - `common/patterns.go` - Regex patterns
   - `common/references.go` - Reference extraction utilities
   - `common/color.go` - 14 DTCG color spaces

**Schema detection priority:**
1. Explicit `$schema` field
2. Config-provided default
3. Duck-typing ($ref, $extends, structured colors -> 2025.10)
4. Default to Draft

**Reference types:**
- Curly brace: `{color.primary}` (both schemas)
- JSON Pointer: `"$ref": "#/color/primary"` (2025.10 only)

### mappa (bennypowers.dev/mappa)

**Reference architecture for:**
- FileSystem interface (`fs/fs.go`)
- Makefile structure for go-release-workflows
- Cobra/Viper CLI patterns
- testutil.NewFixtureFS pattern
- Builder pattern for config

**Key patterns:**
- FileSystem congruent with cem for duck typing
- Never use os.ReadFile directly - always accept FileSystem param
- Fixture/golden testing with testdata/ directories
- MapFileSystem for in-memory testing

## CLAUDE.md Guidelines (Combined)

```markdown
## Go

Getter methods should be named `Foo()`, not `GetFoo()`.

Use go 1.25+ features. Run `go vet` to surface gopls suggestions:
- replace `interface{}` with `any`
- replace `if/else` with `min`
- replace `m[k]=v` loop with `maps.Copy`
- use `slices.Contains` instead of manual loops

## Testing

Practice TDD. Use fixture/golden patterns:
- **Fixtures**: Input data in `testdata/` directories
- **Goldens**: Expected output in `testdata/golden/`
- Tests should support `--update` flag for golden regeneration

**Always use `testutil.NewFixtureFS`** - never `NewOSFileSystem()` in tests.

### Fixture Structure
Each test scenario is a subdirectory containing:
- Input files (`.json`, `.yaml`)
- `expected.json` for assertions

## FileSystem Interface

Define `FileSystem` congruent with cem/mappa for duck typing:
- Never use `os.ReadFile`, `os.Stat`, etc. directly
- All disk-reading functions must accept FileSystem param
- Use `NewOSFileSystem()` only at CLI entry point
- Use `NewMapFileSystem()` in tests

## Logging

Never pollute stdout. Use internal logging package for debug output.

## Git

Always use `Assisted-By`, never `Co-Authored-By` for AI contributions.
```

## Schema Versions

### Editor's Draft
- String color values: `"#FF6B35"`, `"rgb(...)"`
- Curly brace references: `"{color.primary}"`
- Group markers: `_`, `@`, `DEFAULT`
- No $ref, no $extends

### 2025.10 Stable
- Structured colors: `{ colorSpace, components, alpha, hex }`
- 14 color spaces: sRGB, oklch, display-p3, etc.
- JSON Pointer refs: `"$ref": "#/path/to/token"`
- Group inheritance: `"$extends": "#/baseGroup"`
- `$root` reserved token (replaces group markers)
- Backward compatible with curly brace refs

## CLI Commands Specification

### validate
```
asimonim validate [files...]

Flags:
  --schema     Force schema version (draft, v2025_10)
  --strict     Fail on warnings
  --quiet      Only output errors
```

### list
```
asimonim list [files...]

Flags:
  --type       Filter by token type
  --resolved   Show resolved values
  --css        Output as CSS custom properties
  --swatch     Show terminal color swatches
  --format     json|table|css
```

### search
```
asimonim search <query> [files...]

Flags:
  --name       Search names only
  --value      Search values only
  --type       Filter by type
  --regex      Query is regex
  --format     json|table|names
```

## Dependencies

Required:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Config management
- `gopkg.in/yaml.v3` - JSON/YAML parsing with positions
- `github.com/tidwall/jsonc` - JSON with comments
- `github.com/bmatcuk/doublestar/v4` - Glob matching
- `github.com/mazznoer/csscolorparser` - Color parsing

No CGO required (unlike mappa which needs tree-sitter).

## Build Targets

go-release-workflows compatible (CGO_ENABLED=0):
- linux-x64, linux-arm64
- darwin-x64, darwin-arm64
- win32-x64, win32-arm64
