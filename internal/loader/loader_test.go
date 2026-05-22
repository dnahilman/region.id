package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCSV(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setupTinyData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeCSV(t, filepath.Join(dir, "provinces.csv"),
		"11,ACEH\n12,SUMATERA UTARA\n")
	writeCSV(t, filepath.Join(dir, "regencies.csv"),
		"1101,11,KABUPATEN SIMEULUE\n1102,11,KABUPATEN ACEH SINGKIL\n1201,12,KABUPATEN NIAS\n")
	writeCSV(t, filepath.Join(dir, "districts.csv"),
		"1101010,1101,TEUPAH SELATAN\n1101020,1101,SIMEULUE TIMUR\n1201010,1201,GUNUNGSITOLI\n")
	writeCSV(t, filepath.Join(dir, "villages.csv"),
		"1101010001,1101010,LATIUNG\n1101010002,1101010,LABUHAN BAJAU\n1101020001,1101020,DESA SATU\n")
	return dir
}

func TestLoad_Counts(t *testing.T) {
	dir := setupTinyData(t)
	idx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(idx.Provinces) != 2 {
		t.Errorf("provinces: got %d, want 2", len(idx.Provinces))
	}
	if len(idx.Regencies) != 3 {
		t.Errorf("regencies: got %d, want 3", len(idx.Regencies))
	}
	if len(idx.Districts) != 3 {
		t.Errorf("districts: got %d, want 3", len(idx.Districts))
	}
	if len(idx.Villages) != 3 {
		t.Errorf("villages: got %d, want 3", len(idx.Villages))
	}
}

func TestLoad_Lookups(t *testing.T) {
	dir := setupTinyData(t)
	idx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	p := idx.ProvinceByID["11"]
	if p == nil || p.Name != "ACEH" {
		t.Errorf("ProvinceByID[11]: %v, want ACEH", p)
	}

	regs := idx.RegenciesByProvince["11"]
	if len(regs) != 2 {
		t.Errorf("RegenciesByProvince[11]: %d, want 2", len(regs))
	}
	if regs[0].ID != "1101" {
		t.Errorf("first regency id: %q, want 1101", regs[0].ID)
	}

	dists := idx.DistrictsByRegency["1101"]
	if len(dists) != 2 {
		t.Errorf("DistrictsByRegency[1101]: %d, want 2", len(dists))
	}

	vill := idx.VillageByID["1101010001"]
	if vill == nil || vill.Name != "LATIUNG" {
		t.Errorf("VillageByID[1101010001]: %v, want LATIUNG", vill)
	}
}

func TestLoad_PointerStability(t *testing.T) {
	dir := setupTinyData(t)
	idx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Verify that the pointer in the map points into the slice backing array.
	mapPtr := idx.ProvinceByID["11"]
	slicePtr := &idx.Provinces[0]
	if mapPtr != slicePtr {
		t.Errorf("pointer mismatch: map=%p slice=%p", mapPtr, slicePtr)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}
