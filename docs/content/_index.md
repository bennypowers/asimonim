---
title: "Asimonim"
layout: home
---

# Asimonim

![Asimonim logo](/asimonim/images/logo.svg)

Parse, validate, search, and convert [DTCG design tokens][dtcg] from the command line or your editor.
{.subheading}

Multi-schema support, alias resolution with cycle detection, and export to
CSS, SCSS, TypeScript, Swift, Android XML, and more. A built-in [language
server][lsp] brings hover docs, completions, diagnostics, and code actions
to VS Code, Zed, Neovim, and any LSP-capable editor. An [MCP server][mcp]
lets AI agents query your tokens directly.

```bash
npm install -g @pwrs/asimonim
```

<div class="mt-3 grid-2">
  {{< cta link="/docs/installation" text="Get Started" >}}
  {{< cta link="/docs" type="secondary" text="Read the Docs" >}}
</div>

[dtcg]: https://design-tokens.github.io/community-group/format/
[lsp]: /asimonim/docs/lsp/
[mcp]: /asimonim/docs/reference/commands/mcp/
