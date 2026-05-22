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

// buildTinyIndex mirrors the synthetic data used in loader_test.
func buildTinyIndex() *loader.Index {
	idx := &loader.Index{
		Provinces: []model.Province{{ID: "11", Name: "ACEH"}},
		Regencies: []model.Regency{{ID: "1101", ProvinceID: "11", Name: "K1"}},
		Districts: []model.District{{ID: "1101010", RegencyID: "1101", Name: "D1"}},
		Villages:  []model.Village{{ID: "1101010001", DistrictID: "1101010", Name: "V1"}},

		ProvinceByID:        map[string]*model.Province{},
		RegencyByID:         map[string]*model.Regency{},
		DistrictByID:        map[string]*model.District{},
		VillageByID:         map[string]*model.Village{},
		RegenciesByProvince: map[string][]*model.Regency{},
		DistrictsByRegency:  map[string][]*model.District{},
		VillagesByDistrict:  map[string][]*model.Village{},
	}
	idx.ProvinceByID["11"] = &idx.Provinces[0]
	idx.RegencyByID["1101"] = &idx.Regencies[0]
	idx.DistrictByID["1101010"] = &idx.Districts[0]
	idx.VillageByID["1101010001"] = &idx.Villages[0]
	idx.RegenciesByProvince["11"] = []*model.Regency{&idx.Regencies[0]}
	idx.DistrictsByRegency["1101"] = []*model.District{&idx.Districts[0]}
	idx.VillagesByDistrict["1101010"] = []*model.Village{&idx.Villages[0]}
	return idx
}

func TestGenerate_TinyIndex(t *testing.T) {
	dir := t.TempDir()
	idx := buildTinyIndex()

	res, err := Generate(context.Background(), idx, dir, Options{Workers: 2})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 1 provinces.json + 1 province + 1 regencies list + 1 regency
	// + 1 districts list + 1 district + 1 villages list + 1 village = 8
	if res.FilesWritten != 8 {
		t.Errorf("FilesWritten=%d, want 8", res.FilesWritten)
	}

	// Verify a couple of files exist with expected content.
	prov, err := os.ReadFile(filepath.Join(dir, "api", "province", "11.json"))
	if err != nil {
		t.Fatalf("read province/11.json: %v", err)
	}
	var got model.Province
	if err := json.Unmarshal(prov, &got); err != nil {
		t.Fatalf("parse province json: %v\nraw: %s", err, prov)
	}
	if got.ID != "11" || got.Name != "ACEH" {
		t.Errorf("province content: %+v", got)
	}
	if got.Links == nil {
		t.Error("province should have _links by default")
	}

	// List endpoint: rows must omit _links (byte-compat with emsifa).
	regs, err := os.ReadFile(filepath.Join(dir, "api", "regencies", "11.json"))
	if err != nil {
		t.Fatalf("read regencies/11.json: %v", err)
	}
	want := `[{"id":"1101","province_id":"11","name":"K1"}]`
	if string(regs) != want {
		t.Errorf("regencies list byte-compat mismatch\n got: %s\nwant: %s", regs, want)
	}
}

func TestGenerate_NoLinks_ByteCompat(t *testing.T) {
	dir := t.TempDir()
	idx := buildTinyIndex()

	_, err := Generate(context.Background(), idx, dir, Options{NoLinks: true, Workers: 1})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prov, err := os.ReadFile(filepath.Join(dir, "api", "province", "11.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Must match the emsifa output exactly.
	want := `{"id":"11","name":"ACEH"}`
	if string(prov) != want {
		t.Errorf("no-links province byte-compat mismatch\n got: %s\nwant: %s", prov, want)
	}
}

func TestGenerate_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	idx := buildTinyIndex()
	_, _ = Generate(context.Background(), idx, dir, Options{NoLinks: true, Workers: 1})

	prov, _ := os.ReadFile(filepath.Join(dir, "api", "provinces.json"))
	if len(prov) == 0 {
		t.Fatal("empty provinces.json")
	}
	if prov[len(prov)-1] == '\n' {
		t.Errorf("trailing newline present (would break emsifa byte-compat): %q", prov)
	}
}
