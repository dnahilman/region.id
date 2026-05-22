// Package model defines the canonical shapes for Indonesian administrative
// region entities. Field declaration order is preserved by encoding/json and
// must match emsifa/api-wilayah-indonesia so list responses are byte-compatible.
//
// HATEOAS links are an optional, omittable trailing field. List endpoints
// leave Links nil so the row JSON stays identical to the original API; single
// endpoints attach Links just before encoding.
package model

// Links is the HATEOAS envelope attached to single-entity responses.
// Self is always populated; Parent is set for non-root entities; Children is
// set for non-leaf entities.
type Links struct {
	Self     string `json:"self"`
	Parent   string `json:"parent,omitempty"`
	Children string `json:"children,omitempty"`
}

// Province is the top-level administrative entity (2-digit ID).
type Province struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Links *Links `json:"_links,omitempty"`
}

// Regency is a kabupaten or kota (4-digit ID; first 2 digits == ProvinceID).
type Regency struct {
	ID         string `json:"id"`
	ProvinceID string `json:"province_id"`
	Name       string `json:"name"`
	Links      *Links `json:"_links,omitempty"`
}

// District is a kecamatan (7-digit ID; first 4 digits == RegencyID).
type District struct {
	ID        string `json:"id"`
	RegencyID string `json:"regency_id"`
	Name      string `json:"name"`
	Links     *Links `json:"_links,omitempty"`
}

// Village is a kelurahan or desa (10-digit ID; first 7 digits == DistrictID).
type Village struct {
	ID         string `json:"id"`
	DistrictID string `json:"district_id"`
	Name       string `json:"name"`
	Links      *Links `json:"_links,omitempty"`
}
