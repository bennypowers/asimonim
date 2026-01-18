## asimonim CLI usage

When running asimonim commands against test fixtures, use the `-p` flag:

```shell
$ make
$ dist/bin/asimonim validate testdata/fixtures/draft/simple/tokens.json
```

## Go

Getter methods should be named `Foo()`, not `GetFoo()`.

Use go 1.25+ features. Run `go vet` to surface gopls suggestions:
- replace `interface{}` with `any`
- replace `if/else` with `min`
- replace `m[k]=v` loop with `maps.Copy`
- use `slices.Contains` instead of manual loops

## Debugging

When debugging Go code, use the internal logger. Don't use `fmt.Printf` which pollutes stdio.

## Testing

Run `make lint` and `make test` to verify changes.

Practice TDD. When writing tests, always use the fixture/golden patterns:

- **Fixtures**: Input test data in `testdata/` directories
- **Goldens**: Expected output files to compare against (e.g., `expected.json`)
- Tests should support `--update` flag to regenerate golden files when intentional changes occur

### Fixture Structure

Each test scenario is a subdirectory containing:
- Input files (`.json`, `.yaml`, `.tokens.json`)
- `expected.json` (required for assertions)

### Using NewFixtureFS

**Always use `testutil.NewFixtureFS` for tests**, never use `NewOSFileSystem()` in tests:

```go
func TestSomething(t *testing.T) {
    // Load fixtures into MapFileSystem
    mfs := testutil.NewFixtureFS(t, "draft/simple", "/test")

    // Read fixture files from the virtual filesystem
    input, err := mfs.ReadFile("/test/tokens.json")
    expected, err := mfs.ReadFile("/test/expected.json")

    // Pass the MapFileSystem to functions under test
    result, err := parser.ParseFile(mfs, "/test/tokens.json", opts)
}
```

### Why MapFileSystem?

1. **Isolation**: Tests don't depend on working directory or real filesystem state
2. **Speed**: In-memory filesystem is faster than disk I/O
3. **Reproducibility**: Same test runs identically on any machine
4. **Parallelism**: Tests can run concurrently without filesystem conflicts
5. **Integration**: Compatible with cem's testing infrastructure

### Avoiding Inline Test Data

**Don't inline source code in tests:**

```go
// Bad - inline source
json := []byte(`{"color": {"$value": "#fff"}}`)
tokens, _ := parser.ParseBytes(json, opts)

// Good - use fixture file
mfs := testutil.NewFixtureFS(t, "draft/simple", "/test")
tokens, _ := parser.ParseFile(mfs, "/test/tokens.json", opts)
```

## FileSystem Interface

This package defines a `FileSystem` interface congruent with `bennypowers.dev/cem/internal/platform.FileSystem` and `bennypowers.dev/mappa/fs.FileSystem`. This enables duck typing compatibility.

**Always use the pluggable FileSystem:**
- Never use `os.ReadFile`, `os.Stat`, `os.ReadDir`, etc. directly
- All functions that read from disk must accept a `FileSystem` parameter
- This enables testability with mock filesystems and integration with cem/dtls
- Use `NewOSFileSystem()` only at the top level (CLI entry point)
- Use `NewMapFileSystem()` and `NewFixtureFS()` in tests

Example:
```go
// Good - accepts FileSystem
func ParseFile(fs FileSystem, path string, opts Options) ([]*Token, error)

// Bad - uses os directly
func ParseFile(path string, opts Options) ([]*Token, error) {
    data, _ := os.ReadFile(path)  // Don't do this
}
```

## Schema Versions

asimonim supports multiple DTCG schema versions:

- **Draft** (Editor's Draft): String colors, curly brace refs `{token.path}`, group markers
- **V2025_10** (Stable): Structured colors, JSON Pointer refs `$ref: "#/path"`, `$extends`, `$root`

When extending for new schema versions:
1. Define new `SchemaVersion` constant in `schema/version.go`
2. Implement `SchemaHandler` interface
3. Register handler in `schema/registry.go`
4. Update detection logic in `schema/detector.go`
5. Add test fixtures in `testdata/fixtures/`

## Git

When commit messages mention AI agents, always use `Assisted-By`, never `Co-Authored-By`.
