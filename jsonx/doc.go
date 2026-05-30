// Package jsonx parses JSONx, a dialect of JSON that is friendlier to write
// and to diff by hand. Every valid JSON document is also valid JSONx, but
// JSONx additionally allows // and /* */ comments, object keys written as bare
// identifiers, Go-style raw strings, and it requires a trailing comma on any
// value that ends a line. It can decode into Go values like encoding/json and
// also supports a stream of type-tagged values (a "series").
package jsonx
