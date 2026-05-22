package generator

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
)

// bufPool reuses bytes.Buffer instances across worker goroutines for JSON
// encoding. Stripping the trailing newline that json.Encoder.Encode writes
// keeps output byte-identical to the original PHP json_encode behaviour.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// encodeMinified marshals v with encoding/json and strips the trailing newline.
// The returned slice is owned by the caller until it is finished writing it —
// the caller MUST NOT retain a reference past the file write.
//
// This re-uses a pooled buffer to avoid per-call allocation churn at 96k calls.
func encodeMinified(v any) ([]byte, *bytes.Buffer, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // emsifa doesn't escape HTML; preserve byte-compat
	if err := enc.Encode(v); err != nil {
		bufPool.Put(buf)
		return nil, nil, err
	}
	out := buf.Bytes()
	// Encode always appends a single '\n'; strip it to match emsifa output.
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, buf, nil
}

// writeJSON encodes v to path. Returns nil on success.
func writeJSON(path string, v any) error {
	data, buf, err := encodeMinified(v)
	if err != nil {
		return err
	}
	defer bufPool.Put(buf)
	return os.WriteFile(path, data, 0644)
}
