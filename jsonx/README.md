# jsonx

Package `jsonx` parses JSONx, a dialect of JSON that is friendlier to write and
to diff by hand. Every valid JSON document is also valid JSONx. JSONx differs
from JSON in three main ways:

1. **Trailing commas are enforced at end of line.** A value that ends a line
   must be followed by a comma, so in a multi-line object or array every entry
   — including the last — ends with a comma. (On a single line the closing `}`
   or `]` terminates the final entry, so no trailing comma is needed there.)
   This keeps line-based diffs clean: adding or removing a field touches only
   its own line.

2. **Comments are supported** — both line comments (`// ...`) and block
   comments (`/* ... */`).

3. **Object keys may be unquoted identifiers.** A key that is a Go-style
   identifier can be written without quotes: `{value: 42}` instead of
   `{"value": 42}`.

For example:

```jsonx
{
    // a friendly config
    name: "service",
    port: 8080,
    tags: [
        "a",
        "b",
    ],
}
```

decodes to the same value as the equivalent strict JSON.

## Usage

Unmarshal into a Go value, like `encoding/json`:

```go
var cfg Config
err := jsonx.Unmarshal(data, &cfg)
```

Read from a file (`ReadFileMaybeJSON` falls back to plain JSON if the input is
not valid JSONx):

```go
err := jsonx.ReadFile("config.jsonx", &cfg)
```

Convert JSONx to canonical JSON bytes:

```go
out, errs := jsonx.ToJSON(input)
```

## Typed series

A JSONx stream can also hold a sequence of type-tagged values (a "series").
Each entry is a type name followed by a value:

```jsonx
user {name: "alice", admin: true}
host {addr: "10.0.0.1", port: 22}
```

Decode a series with a `TypeMaker` that returns a fresh value for each type
name:

```go
typed, errs := jsonx.ReadSeriesFile("data.jsonx", func(t string) any {
    switch t {
    case "user":
        return new(User)
    case "host":
        return new(Host)
    }
    return nil
})
```
