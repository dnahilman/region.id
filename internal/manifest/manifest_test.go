package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashOptions_Stable(t *testing.T) {
	a := HashOptions(Options{BaseURL: "https://x.test", NoLinks: false, ToolMajorVersion: "0"})
	b := HashOptions(Options{BaseURL: "https://x.test", NoLinks: false, ToolMajorVersion: "0"})
	if a != b {
		t.Errorf("hash not stable: %q vs %q", a, b)
	}
}

func TestHashOptions_ChangesWithFlags(t *testing.T) {
	a := HashOptions(Options{BaseURL: "https://x.test", NoLinks: false, ToolMajorVersion: "0"})
	b := HashOptions(Options{BaseURL: "https://x.test", NoLinks: true, ToolMajorVersion: "0"})
	if a == b {
		t.Error("expected different hash when NoLinks changes")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	os.WriteFile(p, []byte("hello"), 0644)
	h1, err := HashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q vs %q", h1, h2)
	}
	const wantSHA = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h1 != wantSHA {
		t.Errorf("hash(hello)=%q, want %q", h1, wantSHA)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		SchemaVersion: SchemaVersion,
		BuildTime:     "2026-05-22T00:00:00Z",
		ToolVersion:   "region 0.1.0+abc",
		CSVHashes:     map[string]string{"provinces.csv": "abc", "regencies.csv": "def"},
		OptionsHash:   "options-hash",
		FileCount:     96057,
	}
	if err := Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.FileCount != m.FileCount || got.CSVHashes["provinces.csv"] != "abc" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestLoad_Missing(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing manifest, got %+v", got)
	}
}

func TestDecide(t *testing.T) {
	cur := map[string]string{"provinces.csv": "h1", "regencies.csv": "h2"}
	const optHash = "opt"

	if p := Decide(nil, cur, optHash); !p.Rebuild {
		t.Error("nil manifest should rebuild")
	}

	same := &Manifest{
		SchemaVersion: SchemaVersion,
		CSVHashes:     cur,
		OptionsHash:   optHash,
	}
	if p := Decide(same, cur, optHash); p.Rebuild {
		t.Errorf("identical state should NOT rebuild: %+v", p)
	}

	diffCSV := &Manifest{
		SchemaVersion: SchemaVersion,
		CSVHashes:     map[string]string{"provinces.csv": "DIFF", "regencies.csv": "h2"},
		OptionsHash:   optHash,
	}
	if p := Decide(diffCSV, cur, optHash); !p.Rebuild {
		t.Error("csv change should rebuild")
	}

	diffOpt := &Manifest{
		SchemaVersion: SchemaVersion,
		CSVHashes:     cur,
		OptionsHash:   "DIFF",
	}
	if p := Decide(diffOpt, cur, optHash); !p.Rebuild {
		t.Error("options change should rebuild")
	}
}
