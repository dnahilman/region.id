// Package validator checks the integrity of a loaded Index before generation.
//
// All rules are fail-fast: any Error stops the build. Warnings are non-fatal
// unless the caller uses Strict mode.
package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dnahilman/region.id/internal/loader"
)

// Code identifies a class of validation issue. Stable strings — CI logs and
// scripts may grep for them.
type Code string

const (
	CodeDupID           Code = "DUP_ID"
	CodeOrphan          Code = "ORPHAN"
	CodeEmptyName       Code = "EMPTY_NAME"
	CodeBadFormat       Code = "BAD_FORMAT"
	CodePrefixMismatch  Code = "PREFIX_MISMATCH"
	CodeDuplicateName   Code = "WARN_DUPLICATE_NAME"
)

// Issue is a single validation finding.
type Issue struct {
	Code   Code
	File   string // "regencies.csv"
	Line   int    // 1-indexed; 0 if unknown
	Entity string // "province" | "regency" | "district" | "village"
	ID     string
	Detail string
}

// String formats an Issue for pretty terminal output. Mirrors compiler-style
// "file:line CODE message" so editors can jump to the offending line.
func (i Issue) String() string {
	loc := i.File
	if i.Line > 0 {
		loc = fmt.Sprintf("%s:%d", i.File, i.Line)
	}
	return fmt.Sprintf("[%s] %s %s id=%s %s", loc, i.Code, i.Entity, i.ID, i.Detail)
}

// Report bundles all issues from a single validation pass.
type Report struct {
	Errors   []Issue
	Warnings []Issue
}

// HasErrors reports whether the validator found any fatal issues.
func (r *Report) HasErrors() bool { return len(r.Errors) > 0 }

// Issue counts.
func (r *Report) Counts() (errs int, warns int) { return len(r.Errors), len(r.Warnings) }

// Print writes all issues to w (typically os.Stderr) and returns the final
// summary line. Errors first, warnings second, both in input order.
func (r *Report) Print(w fmt.State) {
	for _, e := range r.Errors {
		fmt.Fprintln(w, e.String())
	}
	for _, e := range r.Warnings {
		fmt.Fprintln(w, e.String())
	}
}

var (
	reProvince = regexp.MustCompile(`^\d{2}$`)
	reRegency  = regexp.MustCompile(`^\d{4}$`)
	reDistrict = regexp.MustCompile(`^\d{7}$`)
	reVillage  = regexp.MustCompile(`^\d{10}$`)
)

// Validate runs all rules against idx and returns a Report.
// Rules: DUP_ID, ORPHAN, EMPTY_NAME, BAD_FORMAT, PREFIX_MISMATCH, WARN_DUPLICATE_NAME.
func Validate(idx *loader.Index) *Report {
	r := &Report{}
	validateProvinces(idx, r)
	validateRegencies(idx, r)
	validateDistricts(idx, r)
	validateVillages(idx, r)
	return r
}

// dupIDIssue creates a DUP_ID issue. By policy DUP_ID is a warning, not an
// error, so callers append to Warnings. The original PHP generator silently
// overwrote dupes (last wins); we match that behavior by default but surface
// the issue so it's visible. `region validate --strict` promotes warnings.
func dupIDIssue(file string, line int, entity, id string, prev int) Issue {
	return Issue{
		Code: CodeDupID, File: file, Line: line, Entity: entity, ID: id,
		Detail: fmt.Sprintf("duplicate of line %d", prev),
	}
}

func validateProvinces(idx *loader.Index, r *Report) {
	seen := make(map[string]int, len(idx.Provinces))
	for i, p := range idx.Provinces {
		line := i + 1
		if !reProvince.MatchString(p.ID) {
			r.Errors = append(r.Errors, Issue{
				Code: CodeBadFormat, File: "provinces.csv", Line: line, Entity: "province", ID: p.ID,
				Detail: "id must match ^\\d{2}$",
			})
		}
		if strings.TrimSpace(p.Name) == "" {
			r.Errors = append(r.Errors, Issue{
				Code: CodeEmptyName, File: "provinces.csv", Line: line, Entity: "province", ID: p.ID,
				Detail: "name is empty",
			})
		}
		if prev, ok := seen[p.ID]; ok {
			r.Warnings = append(r.Warnings, dupIDIssue("provinces.csv", line, "province", p.ID, prev))
		} else {
			seen[p.ID] = line
		}
	}
}

func validateRegencies(idx *loader.Index, r *Report) {
	seen := make(map[string]int, len(idx.Regencies))
	nameByParent := make(map[string]map[string]int) // provinceID -> name -> lineFirstSeen

	for i, reg := range idx.Regencies {
		line := i + 1
		if !reRegency.MatchString(reg.ID) {
			r.Errors = append(r.Errors, Issue{
				Code: CodeBadFormat, File: "regencies.csv", Line: line, Entity: "regency", ID: reg.ID,
				Detail: "id must match ^\\d{4}$",
			})
		} else if !strings.HasPrefix(reg.ID, reg.ProvinceID) || len(reg.ProvinceID) != 2 {
			r.Errors = append(r.Errors, Issue{
				Code: CodePrefixMismatch, File: "regencies.csv", Line: line, Entity: "regency", ID: reg.ID,
				Detail: fmt.Sprintf("expected id to start with province_id=%q", reg.ProvinceID),
			})
		}
		if strings.TrimSpace(reg.Name) == "" {
			r.Errors = append(r.Errors, Issue{
				Code: CodeEmptyName, File: "regencies.csv", Line: line, Entity: "regency", ID: reg.ID,
				Detail: "name is empty",
			})
		}
		if _, ok := idx.ProvinceByID[reg.ProvinceID]; !ok {
			r.Errors = append(r.Errors, Issue{
				Code: CodeOrphan, File: "regencies.csv", Line: line, Entity: "regency", ID: reg.ID,
				Detail: fmt.Sprintf("references missing province_id=%q", reg.ProvinceID),
			})
		}
		if prev, ok := seen[reg.ID]; ok {
			r.Warnings = append(r.Warnings, dupIDIssue("regencies.csv", line, "regency", reg.ID, prev))
		} else {
			seen[reg.ID] = line
		}
		// Sibling-name duplicate warning.
		if names, ok := nameByParent[reg.ProvinceID]; ok {
			if prev, dup := names[reg.Name]; dup {
				r.Warnings = append(r.Warnings, Issue{
					Code: CodeDuplicateName, File: "regencies.csv", Line: line, Entity: "regency", ID: reg.ID,
					Detail: fmt.Sprintf("sibling has identical name (line %d)", prev),
				})
			} else {
				names[reg.Name] = line
			}
		} else {
			nameByParent[reg.ProvinceID] = map[string]int{reg.Name: line}
		}
	}
}

func validateDistricts(idx *loader.Index, r *Report) {
	seen := make(map[string]int, len(idx.Districts))
	for i, d := range idx.Districts {
		line := i + 1
		if !reDistrict.MatchString(d.ID) {
			r.Errors = append(r.Errors, Issue{
				Code: CodeBadFormat, File: "districts.csv", Line: line, Entity: "district", ID: d.ID,
				Detail: "id must match ^\\d{7}$",
			})
		} else if !strings.HasPrefix(d.ID, d.RegencyID) || len(d.RegencyID) != 4 {
			r.Errors = append(r.Errors, Issue{
				Code: CodePrefixMismatch, File: "districts.csv", Line: line, Entity: "district", ID: d.ID,
				Detail: fmt.Sprintf("expected id to start with regency_id=%q", d.RegencyID),
			})
		}
		if strings.TrimSpace(d.Name) == "" {
			r.Errors = append(r.Errors, Issue{
				Code: CodeEmptyName, File: "districts.csv", Line: line, Entity: "district", ID: d.ID,
				Detail: "name is empty",
			})
		}
		if _, ok := idx.RegencyByID[d.RegencyID]; !ok {
			r.Errors = append(r.Errors, Issue{
				Code: CodeOrphan, File: "districts.csv", Line: line, Entity: "district", ID: d.ID,
				Detail: fmt.Sprintf("references missing regency_id=%q", d.RegencyID),
			})
		}
		if prev, ok := seen[d.ID]; ok {
			r.Warnings = append(r.Warnings, dupIDIssue("districts.csv", line, "district", d.ID, prev))
		} else {
			seen[d.ID] = line
		}
	}
}

func validateVillages(idx *loader.Index, r *Report) {
	seen := make(map[string]int, len(idx.Villages))
	for i, v := range idx.Villages {
		line := i + 1
		if !reVillage.MatchString(v.ID) {
			r.Errors = append(r.Errors, Issue{
				Code: CodeBadFormat, File: "villages.csv", Line: line, Entity: "village", ID: v.ID,
				Detail: "id must match ^\\d{10}$",
			})
		} else if !strings.HasPrefix(v.ID, v.DistrictID) || len(v.DistrictID) != 7 {
			r.Errors = append(r.Errors, Issue{
				Code: CodePrefixMismatch, File: "villages.csv", Line: line, Entity: "village", ID: v.ID,
				Detail: fmt.Sprintf("expected id to start with district_id=%q", v.DistrictID),
			})
		}
		if strings.TrimSpace(v.Name) == "" {
			r.Errors = append(r.Errors, Issue{
				Code: CodeEmptyName, File: "villages.csv", Line: line, Entity: "village", ID: v.ID,
				Detail: "name is empty",
			})
		}
		if _, ok := idx.DistrictByID[v.DistrictID]; !ok {
			r.Errors = append(r.Errors, Issue{
				Code: CodeOrphan, File: "villages.csv", Line: line, Entity: "village", ID: v.ID,
				Detail: fmt.Sprintf("references missing district_id=%q", v.DistrictID),
			})
		}
		if prev, ok := seen[v.ID]; ok {
			r.Warnings = append(r.Warnings, dupIDIssue("villages.csv", line, "village", v.ID, prev))
		} else {
			seen[v.ID] = line
		}
	}
}
