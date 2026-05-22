// Package meta builds the /api/meta.json and /api/openapi.json artifacts.
//
// meta.json is a hand-built struct so the JSON ordering is stable across
// builds and easy to reason about. openapi.json is expanded from a template
// kept alongside this file.
package meta

import (
	"time"

	"github.com/dnahilman/region.id/internal/loader"
)

// SchemaVersion bumps when Meta's shape changes incompatibly.
const SchemaVersion = 1

// Build holds info captured at build time and embedded in meta.json.
type Build struct {
	Time    string `json:"time"`    // RFC3339
	Commit  string `json:"commit"`  // git SHA
	Tool    string `json:"tool"`    // "region <semver>+<sha>"
}

// Counts are the canonical row counts from the loaded index.
type Counts struct {
	Provinces int `json:"provinces"`
	Regencies int `json:"regencies"`
	Districts int `json:"districts"`
	Villages  int `json:"villages"`
	Files     int `json:"files"`
}

// Source describes upstream attribution.
type Source struct {
	Upstream    string `json:"upstream"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
}

// Meta is the /api/meta.json payload.
type Meta struct {
	SchemaVersion int               `json:"schema_version"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Version       string            `json:"version"`
	Build         Build             `json:"build"`
	Counts        Counts            `json:"counts"`
	Source        Source            `json:"source"`
	Endpoints     map[string]string `json:"endpoints"`
}

// BuildMeta assembles the Meta payload for the given index and version info.
// fileCount is the total written by the generator (caller passes Result.FilesWritten).
func BuildMeta(idx *loader.Index, version, commit string, fileCount int, buildTime time.Time) *Meta {
	return &Meta{
		SchemaVersion: SchemaVersion,
		Name:          "region.id",
		Description:   "Static API for Indonesian administrative regions (provinces, regencies, districts, villages).",
		Version:       version,
		Build: Build{
			Time:   buildTime.UTC().Format(time.RFC3339),
			Commit: commit,
			Tool:   "region " + version + "+" + commit,
		},
		Counts: Counts{
			Provinces: len(idx.Provinces),
			Regencies: len(idx.Regencies),
			Districts: len(idx.Districts),
			Villages:  len(idx.Villages),
			Files:     fileCount,
		},
		Source: Source{
			Upstream:    "https://github.com/emsifa/api-wilayah-indonesia",
			License:     "MIT",
			Attribution: "Data originally compiled by emsifa from BPS / Kemendagri public sources.",
		},
		Endpoints: map[string]string{
			"provinces": "/api/provinces.json",
			"regencies": "/api/regencies/{provinceId}.json",
			"districts": "/api/districts/{regencyId}.json",
			"villages":  "/api/villages/{districtId}.json",
			"province":  "/api/province/{id}.json",
			"regency":   "/api/regency/{id}.json",
			"district":  "/api/district/{id}.json",
			"village":   "/api/village/{id}.json",
			"meta":      "/api/meta.json",
			"openapi":   "/api/openapi.json",
		},
	}
}
