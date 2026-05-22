// Package loader parses the four CSV files in the data directory and builds
// an in-memory Index with O(1) parent→child and by-ID lookups.
//
// Slices are the source of truth — maps store pointers into the slice
// backing arrays, so callers can iterate ordered slices for deterministic
// output and dereference map pointers for random access. Slices are never
// re-grown after initial parse, so the pointers stay valid.
package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dnahilman/region.id/internal/model"
)

// Index is the loaded, indexed dataset.
type Index struct {
	Provinces []model.Province
	Regencies []model.Regency
	Districts []model.District
	Villages  []model.Village

	ProvinceByID map[string]*model.Province
	RegencyByID  map[string]*model.Regency
	DistrictByID map[string]*model.District
	VillageByID  map[string]*model.Village

	RegenciesByProvince map[string][]*model.Regency
	DistrictsByRegency  map[string][]*model.District
	VillagesByDistrict  map[string][]*model.Village
}

// LineNumber returns the 1-indexed CSV row at which an entity was parsed.
// Returns 0 if not tracked (kept on the side to avoid bloating model structs).
// Currently unused — placeholder for future error reporting refinement.
func (i *Index) LineNumber(entity string, id string) int { return 0 }

// Load reads provinces.csv, regencies.csv, districts.csv, villages.csv from
// dataDir and returns a populated Index. Parsing is strict: each file must
// have the expected fixed number of fields per record and at least one row.
func Load(dataDir string) (*Index, error) {
	idx := &Index{}

	if err := loadProvinces(filepath.Join(dataDir, "provinces.csv"), idx); err != nil {
		return nil, err
	}
	if err := loadRegencies(filepath.Join(dataDir, "regencies.csv"), idx); err != nil {
		return nil, err
	}
	if err := loadDistricts(filepath.Join(dataDir, "districts.csv"), idx); err != nil {
		return nil, err
	}
	if err := loadVillages(filepath.Join(dataDir, "villages.csv"), idx); err != nil {
		return nil, err
	}

	buildMaps(idx)
	return idx, nil
}

func openCSV(path string, fields int) (*csv.Reader, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	r := csv.NewReader(f)
	r.FieldsPerRecord = fields
	r.ReuseRecord = true
	return r, f, nil
}

func loadProvinces(path string, idx *Index) error {
	r, f, err := openCSV(path, 2)
	if err != nil {
		return err
	}
	defer f.Close()

	// Pre-size for the well-known dataset; cheap to grow if user data differs.
	idx.Provinces = make([]model.Province, 0, 64)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("provinces.csv: %w", err)
		}
		idx.Provinces = append(idx.Provinces, model.Province{
			ID:   row[0],
			Name: row[1],
		})
	}
	return nil
}

func loadRegencies(path string, idx *Index) error {
	r, f, err := openCSV(path, 3)
	if err != nil {
		return err
	}
	defer f.Close()

	idx.Regencies = make([]model.Regency, 0, 1024)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("regencies.csv: %w", err)
		}
		idx.Regencies = append(idx.Regencies, model.Regency{
			ID:         row[0],
			ProvinceID: row[1],
			Name:       row[2],
		})
	}
	return nil
}

func loadDistricts(path string, idx *Index) error {
	r, f, err := openCSV(path, 3)
	if err != nil {
		return err
	}
	defer f.Close()

	idx.Districts = make([]model.District, 0, 8192)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("districts.csv: %w", err)
		}
		idx.Districts = append(idx.Districts, model.District{
			ID:        row[0],
			RegencyID: row[1],
			Name:      row[2],
		})
	}
	return nil
}

func loadVillages(path string, idx *Index) error {
	r, f, err := openCSV(path, 3)
	if err != nil {
		return err
	}
	defer f.Close()

	idx.Villages = make([]model.Village, 0, 90_000)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("villages.csv: %w", err)
		}
		idx.Villages = append(idx.Villages, model.Village{
			ID:         row[0],
			DistrictID: row[1],
			Name:       row[2],
		})
	}
	return nil
}

// buildMaps populates all lookup maps. Must be called after the slices are
// fully appended (no further growth) so pointers stay stable.
func buildMaps(idx *Index) {
	idx.ProvinceByID = make(map[string]*model.Province, len(idx.Provinces))
	for i := range idx.Provinces {
		idx.ProvinceByID[idx.Provinces[i].ID] = &idx.Provinces[i]
	}

	idx.RegencyByID = make(map[string]*model.Regency, len(idx.Regencies))
	idx.RegenciesByProvince = make(map[string][]*model.Regency, len(idx.Provinces))
	for i := range idx.Regencies {
		r := &idx.Regencies[i]
		idx.RegencyByID[r.ID] = r
		idx.RegenciesByProvince[r.ProvinceID] = append(idx.RegenciesByProvince[r.ProvinceID], r)
	}

	idx.DistrictByID = make(map[string]*model.District, len(idx.Districts))
	idx.DistrictsByRegency = make(map[string][]*model.District, len(idx.Regencies))
	for i := range idx.Districts {
		d := &idx.Districts[i]
		idx.DistrictByID[d.ID] = d
		idx.DistrictsByRegency[d.RegencyID] = append(idx.DistrictsByRegency[d.RegencyID], d)
	}

	idx.VillageByID = make(map[string]*model.Village, len(idx.Villages))
	idx.VillagesByDistrict = make(map[string][]*model.Village, len(idx.Districts))
	for i := range idx.Villages {
		v := &idx.Villages[i]
		idx.VillageByID[v.ID] = v
		idx.VillagesByDistrict[v.DistrictID] = append(idx.VillagesByDistrict[v.DistrictID], v)
	}
}
