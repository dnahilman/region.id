package hateoas

import (
	"encoding/json"
	"testing"

	"github.com/dnahilman/region.id/internal/model"
)

func TestForProvince_RelativeAndAbsolute(t *testing.T) {
	p := &model.Province{ID: "11", Name: "ACEH"}

	l := ForProvince("", p)
	if l.Self != "/api/province/11.json" {
		t.Errorf("self relative: %q", l.Self)
	}
	if l.Children != "/api/regencies/11.json" {
		t.Errorf("children relative: %q", l.Children)
	}
	if l.Parent != "" {
		t.Errorf("province should have no parent, got %q", l.Parent)
	}

	l = ForProvince("https://x.test", p)
	if l.Self != "https://x.test/api/province/11.json" {
		t.Errorf("self abs: %q", l.Self)
	}
}

func TestForRegency_JSONShape(t *testing.T) {
	r := &model.Regency{ID: "1101", ProvinceID: "11", Name: "K"}
	r.Links = ForRegency("", r)

	got, _ := json.Marshal(r)
	want := `{"id":"1101","province_id":"11","name":"K","_links":{"self":"/api/regency/1101.json","parent":"/api/province/11.json","children":"/api/districts/1101.json"}}`
	if string(got) != want {
		t.Errorf("regency json mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestForVillage_NoChildren(t *testing.T) {
	v := &model.Village{ID: "1101010001", DistrictID: "1101010", Name: "V"}
	v.Links = ForVillage("", v)

	got, _ := json.Marshal(v)
	want := `{"id":"1101010001","district_id":"1101010","name":"V","_links":{"self":"/api/village/1101010001.json","parent":"/api/district/1101010.json"}}`
	if string(got) != want {
		t.Errorf("village json mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestListRow_NoLinks(t *testing.T) {
	// List rows: Links remains nil → omitted entirely → byte-compat with emsifa.
	r := &model.Regency{ID: "1101", ProvinceID: "11", Name: "K"}
	got, _ := json.Marshal(r)
	want := `{"id":"1101","province_id":"11","name":"K"}`
	if string(got) != want {
		t.Errorf("list row should omit _links\n got: %s\nwant: %s", got, want)
	}
}
