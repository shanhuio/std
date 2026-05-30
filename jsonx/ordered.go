package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// member is one key/value pair of an orderedObject.
type member struct {
	key   string
	value any
}

// orderedObject is a JSON object that preserves the order in which its members
// appeared in the input, unlike map[string]any.
type orderedObject []member

// decodeOrdered decodes JSON bytes into a tree of scalars, []any and
// orderedObject. Object key order is preserved, and numbers are kept as
// json.Number so their original formatting (and integer precision) is retained.
func decodeOrdered(bs []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(bs))
	dec.UseNumber()
	return decodeValue(dec)
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	delim, ok := tok.(json.Delim)
	if !ok {
		// A scalar: bool, json.Number, string, or nil.
		return tok, nil
	}

	switch delim {
	case '{':
		return decodeObject(dec)
	case '[':
		return decodeArray(dec)
	}
	return nil, fmt.Errorf("unexpected delimiter %q", delim)
}

func decodeObject(dec *json.Decoder) (orderedObject, error) {
	obj := orderedObject{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("object key is not a string: %v", keyTok)
		}
		value, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		obj = append(obj, member{key: key, value: value})
	}
	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return obj, nil
}

func decodeArray(dec *json.Decoder) ([]any, error) {
	arr := []any{}
	for dec.More() {
		value, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, value)
	}
	// Consume the closing ']'.
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return arr, nil
}
