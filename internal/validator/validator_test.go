package validator

import (
	"testing"

	"github.com/dnahilman/region.id/internal/loader"
	"github.com/dnahilman/region.id/internal/model"
)

// makeIdx builds an index in-memory and runs buildMaps via Load. We can't call
// the unexported buildMaps directly, so we hand-stitch what the loader would.
func makeIdx(provs []model.Province, regs []model.Regency, dists []model.District, vills []model.Village) *loader.Index {
	idx := &loader.Index{
		Provinces:           provs,
		Regencies:           regs,
		Districts:           dists,
		Villages:            vills,
		ProvinceByID:        map[string]*model.Province{},
		RegencyByID:         map[string]*model.Regency{},
		DistrictByID:        map[string]*model.District{},
		VillageByID:         map[string]*model.Village{},
		RegenciesByProvince: map[string][]*model.Regency{},
		DistrictsByRegency:  map[string][]*model.District{},
		VillagesByDistrict:  map[string][]*model.Village{},
	}
	for i := range idx.Provinces {
		idx.ProvinceByID[idx.Provinces[i].ID] = &idx.Provinces[i]
	}
	for i := range idx.Regencies {
		r := &idx.Regencies[i]
		idx.RegencyByID[r.ID] = r
		idx.RegenciesByProvince[r.ProvinceID] = append(idx.RegenciesByProvince[r.ProvinceID], r)
	}
	for i := range idx.Districts {
		d := &idx.Districts[i]
		idx.DistrictByID[d.ID] = d
		idx.DistrictsByRegency[d.RegencyID] = append(idx.DistrictsByRegency[d.RegencyID], d)
	}
	for i := range idx.Villages {
		v := &idx.Villages[i]
		idx.VillageByID[v.ID] = v
		idx.VillagesByDistrict[v.DistrictID] = append(idx.VillagesByDistrict[v.DistrictID], v)
	}
	return idx
}

func hasCode(r *Report, c Code) bool {
	for _, e := range r.Errors {
		if e.Code == c {
			return true
		}
	}
	for _, e := range r.Warnings {
		if e.Code == c {
			return true
		}
	}
	return false
}

func TestValidate_Clean(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "ACEH"}},
		[]model.Regency{{ID: "1101", ProvinceID: "11", Name: "K1"}},
		[]model.District{{ID: "1101010", RegencyID: "1101", Name: "D1"}},
		[]model.Village{{ID: "1101010001", DistrictID: "1101010", Name: "V1"}},
	)
	r := Validate(idx)
	if r.HasErrors() {
		t.Errorf("expected no errors, got: %+v", r.Errors)
	}
}

func TestValidate_DupID_IsWarning(t *testing.T) {
	// DUP_ID is a warning by policy (matches emsifa last-wins behaviour).
	// --strict at the CLI level promotes warnings to errors.
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "A"}, {ID: "11", Name: "A2"}},
		nil, nil, nil,
	)
	r := Validate(idx)
	if r.HasErrors() {
		t.Errorf("DUP_ID should be warning, but Errors=%+v", r.Errors)
	}
	if !hasCode(r, CodeDupID) {
		t.Error("expected DUP_ID warning")
	}
}

func TestValidate_Orphan(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "A"}},
		[]model.Regency{{ID: "9999", ProvinceID: "99", Name: "X"}}, // bad province
		nil, nil,
	)
	r := Validate(idx)
	if !hasCode(r, CodeOrphan) {
		t.Errorf("expected ORPHAN error, got: %+v", r.Errors)
	}
}

func TestValidate_EmptyName(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "   "}},
		nil, nil, nil,
	)
	r := Validate(idx)
	if !hasCode(r, CodeEmptyName) {
		t.Error("expected EMPTY_NAME error")
	}
}

func TestValidate_BadFormat(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "1", Name: "A"}}, // too short
		nil, nil, nil,
	)
	r := Validate(idx)
	if !hasCode(r, CodeBadFormat) {
		t.Error("expected BAD_FORMAT error")
	}
}

func TestValidate_PrefixMismatch(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "A"}, {ID: "12", Name: "B"}},
		// regency id 1101 but claims province_id=12 → prefix mismatch
		[]model.Regency{{ID: "1101", ProvinceID: "12", Name: "X"}},
		nil, nil,
	)
	r := Validate(idx)
	if !hasCode(r, CodePrefixMismatch) {
		t.Errorf("expected PREFIX_MISMATCH, got: %+v", r.Errors)
	}
}

func TestValidate_DuplicateNameWarning(t *testing.T) {
	idx := makeIdx(
		[]model.Province{{ID: "11", Name: "A"}},
		[]model.Regency{
			{ID: "1101", ProvinceID: "11", Name: "SAME"},
			{ID: "1102", ProvinceID: "11", Name: "SAME"},
		},
		nil, nil,
	)
	r := Validate(idx)
	if r.HasErrors() {
		t.Errorf("expected no fatal errors, got: %+v", r.Errors)
	}
	if !hasCode(r, CodeDuplicateName) {
		t.Errorf("expected WARN_DUPLICATE_NAME, got warnings: %+v", r.Warnings)
	}
}
