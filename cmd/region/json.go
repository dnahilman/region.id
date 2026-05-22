package main

import (
	"encoding/json"
	"io"
)

// encodeJSON marshals v to w using the project's canonical conventions:
// no HTML escaping, no indentation, no trailing newline (the caller in
// main.go strips that).
func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
