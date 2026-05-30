(comment) @comment

(string) @string
(raw_string) @string
(escape_sequence) @string.escape
(number) @number

[
  (true)
  (false)
] @boolean

(null) @constant.builtin

; Object keys (identifier or string before a colon).
(pair key: (identifier) @property)
(pair key: (string) @property)

; Other bare identifiers: dotted paths and series type names. Scoped to
; dotted_name so this never overlaps an object key (kept as @property above).
(dotted_name (identifier) @variable)

[
  "{"
  "}"
  "["
  "]"
] @punctuation.bracket

[
  ":"
  ","
  ";"
  "."
] @punctuation.delimiter
