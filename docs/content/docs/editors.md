---
title: "Editor Integration"
weight: 30
---

Asimonim includes a built-in language server (`asimonim lsp`) with editor
extensions for VS Code, Zed, and Claude Code.

Any editor with LSP support can use Asimonim. Run `asimonim lsp` as the
language server command, with document selectors for CSS, HTML, Twig, PHP,
JavaScript, TypeScript, JSON, and YAML.

## VS Code

Install [Design Tokens Language Server][vscode-ext] from the VS Code Marketplace.

## Zed

Install [design-tokens][zed-ext] from the Zed extension registry.

## Claude Code

Asimonim is available as a [Claude Code plugin][claude-plugin].

## Neovim

Using native Neovim LSP (see [`:help lsp`][neovimlspdocs] for more info):

Create a file like `~/.config/nvim/lsp/asimonim.lua`:

```lua
---@type vim.lsp.ClientConfig
return {
  cmd = { 'asimonim', 'lsp' },
  root_markers = { '.git', 'package.json' },
  filetypes = { 'css', 'html', 'twig', 'php', 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'json', 'yaml' },
  settings = {
    dtls = {
      tokensFiles = {
        {
          path = "~/path/to/tokens.json",
          prefix = "my-ds",
        },
      },
      groupMarkers = { '_', '@', 'DEFAULT' },
    }
  },
  on_attach = function(client, bufnr)
    if vim.lsp.document_color then
      vim.lsp.document_color.enable(true, bufnr, {
        style = 'virtual'
      })
    end
  end,
}
```

{{< tip >}}
If your tokens are in `node_modules` (e.g., `npm:@my-ds/tokens/tokens.json`),
the default `root_markers` may find the wrong `package.json`. The example
above uses `{ '.git', 'package.json' }` which prefers `.git` over nested
`package.json` files.

For non-git projects or monorepos, use a custom `root_dir` that explicitly
skips `node_modules`:

```lua
root_dir = function(bufnr, on_dir)
  local root = vim.fs.root(bufnr, function(name, path)
    if name == 'package.json' and not path:match('node_modules') then
      return true
    end
    return name == '.git'
  end)
  if root then on_dir(root) end
end,
```
{{< /tip >}}

[vscode-ext]: https://marketplace.visualstudio.com/items?itemName=pwrs.design-tokens-language-server-vscode
[zed-ext]: https://zed.dev/extensions/design-tokens
[claude-plugin]: https://github.com/bennypowers/asimonim
[neovimlspdocs]: https://neovim.io/doc/user/lsp.html
