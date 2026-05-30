" Vim syntax file
" Language:    JSONx (shanhu.io/std/jsonx)
" Description: A JSON dialect with // and /* */ comments, unquoted identifier
"              keys, Go-style raw strings, and end-of-line trailing commas.
"
" Install:
"   Copy this file to ~/.vim/syntax/jsonx.vim (or
"   ~/.config/nvim/syntax/jsonx.vim for Neovim), then tell Vim which files are
"   JSONx, e.g. add to your vimrc:
"
"       autocmd BufRead,BufNewFile *.jsonx setfiletype jsonx

if exists("b:current_syntax")
  finish
endif

syn case match

" Comments.
syn keyword jsonxTodo         contained TODO FIXME XXX NOTE
syn match   jsonxLineComment  "//.*$" contains=jsonxTodo,@Spell
syn region  jsonxBlockComment start="/\*" end="\*/" contains=jsonxTodo,@Spell

" Literals: true, false, null.
syn keyword jsonxKeyword true false null

" Numbers: hexadecimal, decimal integers and floats (Go number format). A
" leading +/- sign is matched separately as jsonxSign.
syn match jsonxNumber "\<0x\x\+\>"
syn match jsonxNumber "\<\d\+\%(\.\d*\)\=\%([eE][-+]\=\d\+\)\=\>"
syn match jsonxSign   "[-+]\ze\s*\%(\d\|\.\d\)"

" Strings: "..." with escapes, and `...` raw strings (may span lines).
syn match  jsonxEscape    contained "\\[\"\\/bfnrt]"
syn match  jsonxEscape    contained "\\u\x\{4}"
syn region jsonxString    start=+"+ skip=+\\.+ end=+"+ contains=jsonxEscape,@Spell
syn region jsonxRawString start=+`+ end=+`+

" Identifiers, then object keys on top so a key wins over a plain identifier.
" An object key is an identifier or a string immediately before a ':'.
syn match jsonxIdentifier "\<\h\w*\>"
syn match jsonxKey        "\<\h\w*\>\ze\s*:"

" Punctuation.
syn match jsonxDelimiter "[][{}.,:;]"

hi def link jsonxLineComment  Comment
hi def link jsonxBlockComment Comment
hi def link jsonxTodo         Todo
hi def link jsonxKeyword      Keyword
hi def link jsonxNumber       Number
hi def link jsonxSign         Number
hi def link jsonxString       String
hi def link jsonxRawString    String
hi def link jsonxEscape       SpecialChar
hi def link jsonxIdentifier   Identifier
hi def link jsonxKey          Label
hi def link jsonxDelimiter    Delimiter

let b:current_syntax = "jsonx"
