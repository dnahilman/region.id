// Package hateoas builds the _links envelope attached to single-entity
// responses. Lists never receive _links per row (would double the payload).
//
// All link URLs are prefixed with the BaseURL configured at generation time.
// When BaseURL is empty, URLs are root-relative (e.g. "/api/province/11.json").
package hateoas

import (
	"github.com/dnahilman/region.id/internal/model"
)

// ForProvince returns the _links block for a single province response.
// Provinces have no parent; their children are the regency list.
func ForProvince(baseURL string, p *model.Province) *model.Links {
	return &model.Links{
		Self:     baseURL + "/api/province/" + p.ID + ".json",
		Children: baseURL + "/api/regencies/" + p.ID + ".json",
	}
}

// ForRegency returns the _links block for a single regency response.
func ForRegency(baseURL string, r *model.Regency) *model.Links {
	return &model.Links{
		Self:     baseURL + "/api/regency/" + r.ID + ".json",
		Parent:   baseURL + "/api/province/" + r.ProvinceID + ".json",
		Children: baseURL + "/api/districts/" + r.ID + ".json",
	}
}

// ForDistrict returns the _links block for a single district response.
func ForDistrict(baseURL string, d *model.District) *model.Links {
	return &model.Links{
		Self:     baseURL + "/api/district/" + d.ID + ".json",
		Parent:   baseURL + "/api/regency/" + d.RegencyID + ".json",
		Children: baseURL + "/api/villages/" + d.ID + ".json",
	}
}

// ForVillage returns the _links block for a single village response.
// Villages are leaf entities; Children is omitted.
func ForVillage(baseURL string, v *model.Village) *model.Links {
	return &model.Links{
		Self:   baseURL + "/api/village/" + v.ID + ".json",
		Parent: baseURL + "/api/district/" + v.DistrictID + ".json",
	}
}
