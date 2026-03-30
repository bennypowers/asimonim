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

## Contributing

See [CONTRIBUTING.md][contributingmd]

## License

GPLv3

[dtcg]: https://design-tokens.github.io/community-group/format/
[contributingmd]: ./CONTRIBUTING.md
