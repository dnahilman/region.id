// Command region is the single CLI binary for region.id. It dispatches to
// three subcommands:
//
//	region generate   build the static API from CSVs
//	region validate   run CSV integrity checks only
//	region serve      preview the generated static API locally
//	region version    print version + commit
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dnahilman/region.id/internal/generator"
	"github.com/dnahilman/region.id/internal/loader"
	"github.com/dnahilman/region.id/internal/manifest"
	"github.com/dnahilman/region.id/internal/meta"
	"github.com/dnahilman/region.id/internal/server"
	"github.com/dnahilman/region.id/internal/validator"
	"github.com/dnahilman/region.id/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "version", "--version", "-V":
		fmt.Println("region " + version.String())
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `region — static API generator for Indonesian regions

Usage:
  region <command> [flags]

Commands:
  generate   Build the static API into <out>/api/**.json
  validate   Run CSV integrity checks
  serve      Preview a generated <out> directory over HTTP
  version    Print version and commit
  help       Show this help

Run "region <command> --help" for command-specific flags.`)
}

// ----- generate -----

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "directory containing the four CSV files")
	outDir := fs.String("out", "./static", "output directory (api/ tree is written under here)")
	baseURL := fs.String("base-url", "", "absolute base URL for HATEOAS links (default: relative URLs)")
	noLinks := fs.Bool("no-links", false, "omit _links (byte-identical to emsifa)")
	force := fs.Bool("force", false, "ignore manifest, full rebuild")
	workers := fs.Int("workers", 0, "worker goroutines (default: NumCPU)")
	verbose := fs.Bool("v", false, "verbose progress")
	_ = fs.Parse(args)

	// Normalise base-url to no trailing slash.
	*baseURL = strings.TrimRight(*baseURL, "/")

	start := time.Now()

	// Load.
	idx, err := loader.Load(*dataDir)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	if *verbose {
		log.Printf("loaded: %d provinces, %d regencies, %d districts, %d villages",
			len(idx.Provinces), len(idx.Regencies), len(idx.Districts), len(idx.Villages))
	}

	// Validate (always — fail-fast).
	report := validator.Validate(idx)
	if report.HasErrors() {
		for _, e := range report.Errors {
			fmt.Fprintln(os.Stderr, e.String())
		}
		errs, warns := report.Counts()
		fmt.Fprintf(os.Stderr, "validation failed: %d errors, %d warnings\n", errs, warns)
		os.Exit(1)
	}
	if *verbose {
		for _, w := range report.Warnings {
			fmt.Fprintln(os.Stderr, w.String())
		}
	}

	// Incremental build check.
	csvNames := []string{"provinces.csv", "regencies.csv", "districts.csv", "villages.csv"}
	csvHashes, err := manifest.HashDataDir(*dataDir, csvNames)
	if err != nil {
		log.Fatalf("hash data dir: %v", err)
	}
	opts := manifest.Options{
		BaseURL:          *baseURL,
		NoLinks:          *noLinks,
		ToolMajorVersion: version.Major(),
	}
	optHash := manifest.HashOptions(opts)

	if !*force {
		existing, err := manifest.Load(*outDir)
		if err != nil {
			log.Printf("warning: could not load manifest: %v", err)
		}
		plan := manifest.Decide(existing, csvHashes, optHash)
		if !plan.Rebuild {
			fmt.Printf("region: up to date (%s)\n", plan.Reason)
			return
		}
		if *verbose {
			log.Printf("rebuilding: %s", plan.Reason)
		}
	}

	// Clean previous api/ subtree (keeps top-level files like the manifest until we Save below).
	apiDir := filepath.Join(*outDir, "api")
	if err := os.RemoveAll(apiDir); err != nil {
		log.Fatalf("clean %s: %v", apiDir, err)
	}

	// Generate.
	res, err := generator.Generate(context.Background(), idx, *outDir, generator.Options{
		BaseURL: *baseURL,
		NoLinks: *noLinks,
		Workers: *workers,
		Verbose: *verbose,
	})
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	// Meta + OpenAPI + manifest + copy web/.
	totalFiles := res.FilesWritten
	now := time.Now()

	metaPayload := meta.BuildMeta(idx, version.Version, version.Commit, totalFiles+2 /*meta + openapi*/, now)
	if err := writeJSONFile(filepath.Join(apiDir, "meta.json"), metaPayload); err != nil {
		log.Fatalf("write meta.json: %v", err)
	}
	openapiBytes, err := meta.BuildOpenAPI(*baseURL, version.Version)
	if err != nil {
		log.Fatalf("build openapi: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "openapi.json"), openapiBytes, 0644); err != nil {
		log.Fatalf("write openapi.json: %v", err)
	}
	totalFiles += 2

	if err := copyWebAssets("./web", *outDir); err != nil {
		// Web assets are optional during development; warn rather than fail.
		log.Printf("warning: copy web assets: %v", err)
	}

	if err := manifest.Save(*outDir, &manifest.Manifest{
		SchemaVersion: manifest.SchemaVersion,
		BuildTime:     now.UTC().Format(time.RFC3339),
		ToolVersion:   "region " + version.String(),
		CSVHashes:     csvHashes,
		OptionsHash:   optHash,
		FileCount:     totalFiles,
	}); err != nil {
		log.Fatalf("save manifest: %v", err)
	}

	fmt.Printf("region: wrote %d files to %s in %s\n", totalFiles, apiDir, time.Since(start).Round(time.Millisecond))
}

// writeJSONFile encodes v with the same byte conventions used by the generator:
// minified, no trailing newline, no HTML escaping.
func writeJSONFile(path string, v any) error {
	data, err := encodeMinified(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func encodeMinified(v any) ([]byte, error) {
	// Use the generator's exported convention. We can't import a private helper,
	// so we inline a one-shot encode here (rare path: just 2 calls per build).
	type encoder interface {
		Encode(any) error
	}
	var buf strings.Builder
	enc := newJSONEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.String()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return []byte(out), nil
}

// newJSONEncoder is a tiny indirection to keep encodeMinified easy to read.
func newJSONEncoder(w io.Writer) interface{ Encode(any) error } {
	type writerEncoder = io.Writer
	_ = writerEncoder(nil)
	return jsonEncoder{w: w}
}

type jsonEncoder struct{ w io.Writer }

func (e jsonEncoder) Encode(v any) error {
	// Defer to encoding/json via a thin import-shim so the package list stays clean.
	return encodeJSON(e.w, v)
}

func copyWebAssets(srcDir, outDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("web source not found: %w", err)
	}
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		dst := filepath.Join(outDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0644)
	})
}

// ----- validate -----

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "directory containing the four CSV files")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	_ = fs.Parse(args)

	idx, err := loader.Load(*dataDir)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	report := validator.Validate(idx)

	for _, e := range report.Errors {
		fmt.Fprintln(os.Stderr, e.String())
	}
	for _, w := range report.Warnings {
		fmt.Fprintln(os.Stderr, w.String())
	}
	errs, warns := report.Counts()
	fmt.Fprintf(os.Stderr, "%d errors, %d warnings\n", errs, warns)

	if errs > 0 || (*strict && warns > 0) {
		os.Exit(1)
	}
}

// ----- serve -----

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dir := fs.String("dir", "./static", "directory to serve")
	addr := fs.String("addr", ":8080", "listen address")
	_ = fs.Parse(args)

	if err := server.Run(*addr, *dir); err != nil {
		log.Fatal(err)
	}
}
