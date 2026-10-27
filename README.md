# Asimonim

![A vintage Israeli phone token (asimon), captured mid-drop as it falls into a payphone coin slot](./logo.png)

[![codecov](https://codecov.io/gh/bennypowers/asimonim/graph/badge.svg)](https://codecov.io/gh/bennypowers/asimonim)

A high-performance design tokens parser, validator, and language server, available as a CLI tool and Go library.

> *Asimonim* (אֲסִימוֹנִים) (ahh-see-moh-NEEM) is Hebrew for "[tokens](https://www.wikiwand.com/en/articles/Telephone_token)".

Design systems use [design tokens][dtcg] to store visual primitives like colors, spacing, and typography. Asimonim parses and validates token files defined by the Design Tokens Community Group (DTCG) specification, supporting both the current draft and the stable V2025_10 schema.

## Features

- **Multi-schema support**: Handles both Draft and V2025_10 DTCG schemas
- **Automatic schema detection**: Duck-typing detection of schema version from file contents
- **Alias resolution**: Resolve token references with cycle detection
- **Multi-format export**: Convert tokens to TypeScript, SCSS, Swift, XML, and more
- **CSS output**: Generate CSS custom properties from tokens
- **Search**: Find tokens by name, value, or type with regex support
- **Validation**: Check files for schema compliance and circular references
- **Language Server**: Full LSP support for design tokens in your editor
- **MCP Server**: Model Context Protocol server for AI-assisted development

## Installation

### npm

```bash
npm install -g @pwrs/asimonim
```

### Gentoo Linux

Enable the `bennypowers` overlay, then install:

```bash
eselect repository enable bennypowers
emaint sync -r bennypowers
emerge dev-util/asimonim
```

### From Source

```bash
go install bennypowers.dev/asimonim@latest
```

## Documentation

Full documentation is available at **[bennypowers.dev/asimonim](https://bennypowers.dev/asimonim/)**.

- [Quick Start](https://bennypowers.dev/asimonim/docs/quick-start/)
- [Editor Integration](https://bennypowers.dev/asimonim/docs/editors/)
- [LSP Features](https://bennypowers.dev/asimonim/docs/lsp/)
- [Configuration](https://bennypowers.dev/asimonim/docs/configuration/)
- [CLI Reference](https://bennypowers.dev/asimonim/docs/reference/commands/)
- [Schema Versions](https://bennypowers.dev/asimonim/docs/reference/schemas/)

## Go library

The module is `bennypowers.dev/asimonim`. The public packages are:

| Package | Purpose |
| --- | --- |
| `parser` | Parse DTCG JSON and YAML into tokens |
| `resolver` | Resolve aliases and `$extends` relationships |
| `token` | Core token types and prefix-aware lookup |
| `schema` | Schema version detection and conversion |
| `validator` | Schema consistency checks |
| `convert` | Serialize tokens to CSS, SCSS, JS, Swift, Android XML, and other formats |
| `load` | High-level loading from paths and package specifiers |
| `fs` | Filesystem abstraction |
| `config` | Configuration loading |
| `version` | Build and version information |

Run `go doc bennypowers.dev/asimonim/parser` (or the name of any package
above) for the full API and examples.

### Version information

```go
import "bennypowers.dev/asimonim/version"

fmt.Println(version.Get())
fmt.Println(version.Info()["gitCommit"])
```

`Get` returns the injected or detected version, `Info` returns a structured
map for diagnostics, and `Full` includes the Git commit when available.

### Compatibility policy

The module is pre-1.0 (`0.x`). While it remains at `0.x`:

- Minor releases can include breaking changes to public package APIs. We call
  out such changes in the release notes.
- Patch releases are backwards compatible within the same minor version.
- Pin exact versions, for example `go get bennypowers.dev/asimonim@v0.3.3`.
  Do not rely on floating ranges or on compatibility across minor versions.
- Once the module reaches `1.0`, we will follow semantic versioning and make
  breaking changes only in major releases.

Everything under `internal/` is not public API.

### Input formats

Asimonim parses [DTCG][dtcg] documents. Style Dictionary JSON is not DTCG.
Its `value` and `type` keys, category nesting, and reference syntax differ
from DTCG's `$value`, `$type`, and `{group.token}` forms. Normalize Style
Dictionary output into DTCG before passing it to asimonim. The parser does
not translate Style Dictionary files.

## Contributing

See [CONTRIBUTING.md][contributingmd]

## License

GPL-3.0-or-later

[dtcg]: https://design-tokens.github.io/community-group/format/
[contributingmd]: ./CONTRIBUTING.md
