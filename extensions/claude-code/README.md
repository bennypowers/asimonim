# Asimonim for Claude Code

Design tokens tooling for Claude Code, providing both Language Server Protocol (LSP) and Model Context Protocol (MCP) support for DTCG design tokens.

## Features

### LSP Features (Editor Intelligence)

- **Token Completion**: Auto-complete CSS custom property names from design tokens
- **Hover Documentation**: View token descriptions, values, and types on hover
- **Diagnostics**: Warn when `var()` fallback values don't match token definitions
- **Code Actions**: Toggle fallback values, fix wrong token definitions
- **Document Color**: Display token color values as swatches
- **Semantic Tokens**: Highlight token references in token definition files
- **Go-to-Definition**: Jump to the token's definition in JSON/YAML files

### MCP Features (AI-Native Token Understanding)

- **Token Discovery**: Browse available token sources and tokens via resources
- **Token Search**: Find tokens by name, value, description, or type
- **Validation**: Validate token files for circular references and deprecations
- **Format Conversion**: Convert tokens to CSS, SCSS, JavaScript, Swift, Android XML, and more
- **npm/jsr Discovery**: Automatically discovers tokens from installed packages

#### MCP Tools

| Tool | Description |
| ---- | ----------- |
| `validate_tokens` | Validate token files, detect circular references, report deprecated tokens |
| `search_tokens` | Search tokens by name, value, description, or type with regex support |
| `convert_tokens` | Convert tokens to CSS, SCSS, JS, Swift, Android XML, or other formats |

#### MCP Resources

| URI | Description |
| --- | ----------- |
| `asimonim://tokens` | List available token sources with counts |
| `asimonim://tokens/{source}` | All tokens from a specific source |
| `asimonim://token/{source}/{path}` | Individual token detail |
| `asimonim://config` | Workspace configuration |

## Installation

### 1. Install Asimonim Binary

Choose your preferred method:

**Via npm:**
```bash
npm install -g @pwrs/asimonim
```

**Via Go:**
```bash
go install bennypowers.dev/asimonim@latest
```

Verify installation:
```bash
asimonim version
```

### 2. Install Claude Code Plugin

Add this marketplace to Claude Code:
```
/plugin marketplace add bennypowers/asimonim
```

Then install the plugin:
```
/plugin install asimonim
```

## Usage

### LSP (Editor Intelligence)

The LSP activates automatically for:
- CSS files (`.css`)
- HTML files (`.html`, `.htm`)
- JavaScript files (`.js`, `.mjs`, `.cjs`, `.jsx`)
- TypeScript files (`.ts`, `.tsx`, `.mts`, `.cts`)
- JSON files (`.json`)
- YAML files (`.yaml`, `.yml`)

Simply open a file with design token references and start typing!

### MCP (AI Understanding)

The MCP server activates automatically when the plugin is installed. Use it by asking questions like:

- "What design tokens are available in this project?"
- "Find all color tokens"
- "Convert my tokens to CSS custom properties"
- "Are there any deprecated tokens?"

The AI will have direct access to your design token definitions and can help you use them correctly.

## Configuration

The plugin works out-of-the-box with zero configuration. Asimonim automatically:
- Discovers token files from `.config/design-tokens.yaml`
- Scans npm/jsr dependencies for `designTokens` field or export condition
- Parses resolver documents for token sources

### Custom Configuration

Create a `.config/design-tokens.yaml` in your project root:

```yaml
prefix: "my-ds"
files:
  - ./tokens.json
  - npm:@my-design-system/tokens/tokens.json
resolvers:
  - npm:@my-design-system/tokens/tokens.resolver.json
schema: v2025.10
```

## Troubleshooting

### LSP Not Starting

1. Verify asimonim is installed and in PATH:
   ```bash
   which asimonim
   asimonim lsp --help
   ```

2. Check Claude Code logs for errors:
   ```
   /logs
   ```

3. Restart the LSP server:
   ```
   /lsp restart
   ```

### No Completions Appearing

1. Ensure your project has design token files configured
2. Check that token files are valid:
   ```bash
   asimonim validate tokens.json
   ```
3. Verify token packages in `node_modules` have `designTokens` field

### MCP Not Working

1. Verify the MCP server is loaded:
   ```
   /mcp
   ```

2. Check that asimonim is installed:
   ```bash
   which asimonim
   asimonim mcp --help
   ```

3. Restart Claude Code to reload the MCP server

## Support

- [Issues](https://github.com/bennypowers/asimonim/issues)
- [Source](https://github.com/bennypowers/asimonim)

## License

GPL-3.0 - See [LICENSE](../../LICENSE) for details
