# JSONx for VS Code

Syntax highlighting for [JSONx](../README.md) — a JSON dialect with `//` and
`/* */` comments, unquoted identifier keys, end-of-line trailing commas, and
Go-style raw strings.

This directory is a self-contained VS Code extension:

- `package.json` — extension manifest; registers the `jsonx` language for
  `*.jsonx` files.
- `language-configuration.json` — comments, brackets, and auto-closing pairs.
- `syntaxes/jsonx.tmLanguage.json` — the TextMate grammar.

## Install locally

Copy (or symlink) this directory into your VS Code extensions folder, then
reload the window:

```sh
cp -r jsonx/vscode ~/.vscode/extensions/shanhu.jsonx-0.0.1
```

- macOS / Linux: `~/.vscode/extensions/`
- Windows: `%USERPROFILE%\.vscode\extensions\`

## Package as a .vsix

With [`vsce`](https://github.com/microsoft/vscode-vsce):

```sh
cd jsonx/vscode
vsce package
code --install-extension jsonx-0.0.1.vsix
```
