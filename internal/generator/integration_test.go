//go:build integration

// Integration test: runs the generator against the real ./data CSVs and
// asserts that the output structure matches the documented contract.
//
// Run with:  go test -tags=integration ./internal/generator/...
package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dnahilman/region.id/internal/loader"
	"github.com/dnahilman/region.id/internal/model"
)

func TestIntegration_FullGenerate(t *testing.T) {
	// Resolve real data dir relative to the repo root.
	dataDir := "../../data"
	if _, err := os.Stat(filepath.Join(dataDir, "provinces.csv")); err != nil {
		t.Skipf("real data not found at %s: %v", dataDir, err)
	}

	idx, err := loader.Load(dataDir)
	if err != nil {
		t.Fatalf("load real data: %v", err)
	}

	if len(idx.Provinces) != 34 {
		t.Errorf("provinces=%d, want 34", len(idx.Provinces))
	}
	if len(idx.Regencies) != 514 {
		t.Errorf("regencies=%d, want 514", len(idx.Regencies))
	}
	if len(idx.Districts) != 7215 {
		t.Errorf("districts=%d, want 7215", len(idx.Districts))
	}
	if len(idx.Villages) != 80534 {
		t.Errorf("villages=%d, want 80534", len(idx.Villages))
	}

	out := t.TempDir()
	res, err := Generate(context.Background(), idx, out, Options{
		BaseURL: "https://dnahilman.github.io/region.id",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Expected file count (excluding meta/openapi added at the CLI layer):
	//   1 provinces.json
	// + 34 province singles
	// + 514 regency singles
	// + 7,215 district singles
	// + 80,534 village singles (with 2 dupes overwriting same path -> still 80,534 writes attempted)
	// + 34 regencies lists
	// + 514 districts lists
	// + 7,215 villages lists
	const wantWrites = 1 + 34 + 514 + 7215 + 80534 + 34 + 514 + 7215
	if res.FilesWritten != wantWrites {
		t.Errorf("FilesWritten=%d, want %d", res.FilesWritten, wantWrites)
	}

	// Spot-check a known entity.
	b, err := os.ReadFile(filepath.Join(out, "api", "province", "11.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p model.Province
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("parse province/11.json: %v", err)
	}
	if p.ID != "11" || p.Name != "ACEH" {
		t.Errorf("province/11: %+v", p)
	}
	if p.Links == nil || p.Links.Self == "" {
		t.Errorf("province/11 missing _links: %+v", p)
	}
}
