package docker

import (
	"shanhu.io/std/errcode"
)

// shellSplit splits a simple shell command line into tokens. Tokens are
// separated by whitespace. A token may be wrapped in double quotes to
// include whitespace; inside a double-quoted token, a backslash escapes
// a following double quote.
func shellSplit(line string) ([]string, error) {
	var tokens []string
	var cur []byte
	inToken := false
	inQuote := false

	flush := func() {
		tokens = append(tokens, string(cur))
		cur = cur[:0]
		inToken = false
	}

	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			if c == '\\' && i+1 < len(line) && line[i+1] == '"' {
				cur = append(cur, '"')
				i++
				continue
			}
			if c == '"' {
				inQuote = false
				continue
			}
			cur = append(cur, c)
			continue
		}
		switch c {
		case ' ', '\t', '\n', '\r':
			if inToken {
				flush()
			}
		case '"':
			inToken = true
			inQuote = true
		default:
			inToken = true
			cur = append(cur, c)
		}
	}
	if inQuote {
		return nil, errcode.InvalidArgf("unterminated double quote")
	}
	if inToken {
		flush()
	}
	return tokens, nil
}
