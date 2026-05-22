package meta

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"
)

//go:embed openapi.tmpl.json
var openapiTmpl string

// BuildOpenAPI expands the embedded OpenAPI template with the given base URL
// and version, then re-marshals through encoding/json to canonicalise (minify
// and drop comments/whitespace). The output matches the byte conventions of
// every other generated JSON file in the project.
func BuildOpenAPI(baseURL, version string) ([]byte, error) {
	t, err := template.New("openapi").Parse(openapiTmpl)
	if err != nil {
		return nil, fmt.Errorf("parse openapi template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]string{
		"BaseURL": baseURL,
		"Version": version,
	}); err != nil {
		return nil, fmt.Errorf("execute openapi template: %w", err)
	}

	// Round-trip through encoding/json to canonicalise.
	var doc any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		return nil, fmt.Errorf("openapi template produced invalid json: %w", err)
	}
	return json.Marshal(doc)
}
