# JSONx for Zed

Syntax highlighting for [JSONx](../README.md) in the [Zed](https://zed.dev)
editor. It highlights files with the `.jsonx`, `.caco3`, and `.lets`
extensions.

Zed highlights via [Tree-sitter](https://tree-sitter.github.io/tree-sitter/),
so unlike the Vim and VS Code support this extension relies on a separate
grammar repository, [`tree-sitter-jsonx`](https://github.com/shanhuio/tree-sitter-jsonx).

## Layout

- `extension.toml` — extension manifest; points `[grammars.jsonx]` at the
  grammar repo and the commit to build.
- `languages/jsonx/config.toml` — language config (suffixes, comments,
  brackets).
- `languages/jsonx/highlights.scm` — Tree-sitter highlight queries.
- `languages/jsonx/brackets.scm`, `indents.scm` — bracket matching and
  auto-indent.

## Grammar version

`extension.toml`'s `[grammars.jsonx] commit` pins the `tree-sitter-jsonx`
commit that Zed fetches and builds. After changing the grammar, push it and
bump that SHA:

```sh
git -C path/to/tree-sitter-jsonx rev-parse HEAD
```

## Install as a dev extension

In Zed: open the command palette → **zed: install dev extension** → select this
`jsonx/zed` directory. Zed clones the grammar at the pinned commit, compiles it
to WASM, and applies the highlighting.
