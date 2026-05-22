// Package manifest implements the incremental-build short-circuit for the
// generator. It stores SHA256 hashes of every source CSV plus a hash of the
// build options (base URL, no-links flag, tool major version) in
// .build-manifest.json living inside the output directory.
//
// On the next run, the generator recomputes the same hashes; when everything
// matches, it logs "up to date" and exits without touching any files. Any
// mismatch triggers a full rebuild. Per-CSV partial rebuilds are intentionally
// out of scope for v0.1 — the hierarchical ID encoding makes them fragile.
package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// FileName is the on-disk manifest, relative to the output directory.
const FileName = ".build-manifest.json"

// SchemaVersion bumps whenever Manifest's JSON shape changes incompatibly.
const SchemaVersion = 1

// Manifest is what we write to disk between builds.
type Manifest struct {
	SchemaVersion int               `json:"schema_version"`
	BuildTime     string            `json:"build_time"`
	ToolVersion   string            `json:"tool_version"`
	CSVHashes     map[string]string `json:"csv_hashes"`
	OptionsHash   string            `json:"options_hash"`
	FileCount     int               `json:"file_count"`
}

// Options is the subset of generator options that affects output bytes.
// Anything that changes file contents must be mixed into the OptionsHash so
// the manifest invalidates correctly when toggled.
type Options struct {
	BaseURL          string
	NoLinks          bool
	ToolMajorVersion string // e.g. "0"
}

// HashOptions returns a stable hex SHA256 over the options.
func HashOptions(o Options) string {
	h := sha256.New()
	fmt.Fprintf(h, "base_url=%s\n", o.BaseURL)
	fmt.Fprintf(h, "no_links=%t\n", o.NoLinks)
	fmt.Fprintf(h, "tool_major=%s\n", o.ToolMajorVersion)
	return hex.EncodeToString(h.Sum(nil))
}

// HashFile streams the file at path and returns hex(sha256(contents)).
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashDataDir computes a hash for each csv file in dataDir. Hashes are keyed
// by base filename so the manifest stays portable across machines/paths.
func HashDataDir(dataDir string, names []string) (map[string]string, error) {
	out := make(map[string]string, len(names))
	sort.Strings(names)
	for _, name := range names {
		h, err := HashFile(filepath.Join(dataDir, name))
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", name, err)
		}
		out[name] = h
	}
	return out, nil
}

// Load reads the manifest at outDir. Returns (nil, nil) when the file does
// not exist — a missing manifest is not an error, it means full rebuild.
func Load(outDir string) (*Manifest, error) {
	path := filepath.Join(outDir, FileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	return &m, nil
}

// Save writes m to outDir/FileName. Pretty-printed for human diffability.
func Save(outDir string, m *Manifest) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, FileName), b, 0644)
}

// Plan reports whether a rebuild is needed given the current hashes/options.
// Returns ok==true when a rebuild is required and a short reason; ok==false
// means the existing artifacts are up to date.
type Plan struct {
	Rebuild bool
	Reason  string
}

// Decide compares the existing manifest (may be nil) against the current
// CSV hashes and options hash. Schema-version mismatches force rebuild.
func Decide(existing *Manifest, currentCSVHashes map[string]string, currentOptionsHash string) Plan {
	if existing == nil {
		return Plan{Rebuild: true, Reason: "no existing manifest"}
	}
	if existing.SchemaVersion != SchemaVersion {
		return Plan{Rebuild: true, Reason: "manifest schema version changed"}
	}
	if existing.OptionsHash != currentOptionsHash {
		return Plan{Rebuild: true, Reason: "build options changed"}
	}
	if len(existing.CSVHashes) != len(currentCSVHashes) {
		return Plan{Rebuild: true, Reason: "csv file set changed"}
	}
	for name, want := range currentCSVHashes {
		if existing.CSVHashes[name] != want {
			return Plan{Rebuild: true, Reason: fmt.Sprintf("%s changed", name)}
		}
	}
	return Plan{Rebuild: false, Reason: "up to date"}
}
