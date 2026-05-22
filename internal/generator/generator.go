// Package generator writes the static JSON tree under <outDir>/api using a
// worker pool. The producer goroutine walks the index and ships jobs onto a
// buffered channel; N workers (defaults to runtime.NumCPU()) encode JSON via
// a pooled bytes.Buffer and write the file via os.WriteFile.
//
// All output directories are created up front in a single pass, eliminating
// O(96k) MkdirAll syscalls on the hot path.
package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/dnahilman/region.id/internal/hateoas"
	"github.com/dnahilman/region.id/internal/loader"
	"github.com/dnahilman/region.id/internal/model"
	"golang.org/x/sync/errgroup"
)

// Options controls a single generate run.
type Options struct {
	// BaseURL is prepended to every HATEOAS link. Empty means root-relative.
	BaseURL string
	// NoLinks suppresses _links emission for byte-identical emsifa diffing.
	NoLinks bool
	// Workers overrides the worker pool size. Zero means runtime.NumCPU().
	Workers int
	// Verbose prints a progress line per directory tier.
	Verbose bool
}

// Result summarises a completed generation.
type Result struct {
	FilesWritten int
}

// job is the unit of work shipped from producer to worker.
type job struct {
	path    string
	payload any
}

// Generate writes every endpoint JSON under outDir/api.
// The outDir is created if missing but never cleared — callers that need
// a clean slate should remove outDir/api beforehand.
func Generate(ctx context.Context, idx *loader.Index, outDir string, opts Options) (*Result, error) {
	apiDir := filepath.Join(outDir, "api")
	if err := ensureDirs(apiDir); err != nil {
		return nil, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	jobs := make(chan job, workers*8)
	var written int64

	g, ctx := errgroup.WithContext(ctx)

	// Workers.
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for j := range jobs {
				if err := writeJSON(j.path, j.payload); err != nil {
					return fmt.Errorf("write %s: %w", j.path, err)
				}
				atomic.AddInt64(&written, 1)
			}
			return nil
		})
	}

	// Producer.
	g.Go(func() error {
		defer close(jobs)
		return produce(ctx, idx, apiDir, opts, jobs)
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &Result{FilesWritten: int(written)}, nil
}

// ensureDirs pre-creates the 8 leaf directories under apiDir so workers never
// need to call MkdirAll on the hot path.
func ensureDirs(apiDir string) error {
	dirs := []string{
		apiDir,
		filepath.Join(apiDir, "province"),
		filepath.Join(apiDir, "regency"),
		filepath.Join(apiDir, "district"),
		filepath.Join(apiDir, "village"),
		filepath.Join(apiDir, "regencies"),
		filepath.Join(apiDir, "districts"),
		filepath.Join(apiDir, "villages"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// produce walks the index in deterministic order and enqueues every file.
func produce(ctx context.Context, idx *loader.Index, apiDir string, opts Options, jobs chan<- job) error {
	// /api/provinces.json — list of provinces (no _links on rows).
	if err := send(ctx, jobs, filepath.Join(apiDir, "provinces.json"), idx.Provinces); err != nil {
		return err
	}

	// Per-entity singles + child lists.
	for i := range idx.Provinces {
		p := idx.Provinces[i] // value copy so we can attach Links without mutating the index
		if !opts.NoLinks {
			p.Links = hateoas.ForProvince(opts.BaseURL, &p)
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "province", p.ID+".json"), p); err != nil {
			return err
		}

		regs := idx.RegenciesByProvince[p.ID]
		// Build a list-row slice with Links nil for byte-compat.
		regList := make([]model.Regency, len(regs))
		for i, r := range regs {
			regList[i] = *r
			regList[i].Links = nil
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "regencies", p.ID+".json"), regList); err != nil {
			return err
		}
	}

	for i := range idx.Regencies {
		r := idx.Regencies[i]
		if !opts.NoLinks {
			r.Links = hateoas.ForRegency(opts.BaseURL, &r)
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "regency", r.ID+".json"), r); err != nil {
			return err
		}

		dists := idx.DistrictsByRegency[r.ID]
		distList := make([]model.District, len(dists))
		for i, d := range dists {
			distList[i] = *d
			distList[i].Links = nil
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "districts", r.ID+".json"), distList); err != nil {
			return err
		}
	}

	for i := range idx.Districts {
		d := idx.Districts[i]
		if !opts.NoLinks {
			d.Links = hateoas.ForDistrict(opts.BaseURL, &d)
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "district", d.ID+".json"), d); err != nil {
			return err
		}

		vills := idx.VillagesByDistrict[d.ID]
		villList := make([]model.Village, len(vills))
		for i, v := range vills {
			villList[i] = *v
			villList[i].Links = nil
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "villages", d.ID+".json"), villList); err != nil {
			return err
		}
	}

	for i := range idx.Villages {
		v := idx.Villages[i]
		if !opts.NoLinks {
			v.Links = hateoas.ForVillage(opts.BaseURL, &v)
		}
		if err := send(ctx, jobs, filepath.Join(apiDir, "village", v.ID+".json"), v); err != nil {
			return err
		}
	}

	return nil
}

func send(ctx context.Context, jobs chan<- job, path string, payload any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case jobs <- job{path: path, payload: payload}:
		return nil
	}
}
