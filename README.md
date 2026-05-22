# region.id

> Static API for Indonesian administrative regions — provinces, regencies, districts, villages.
> Go rewrite of [emsifa/api-wilayah-indonesia](https://github.com/emsifa/api-wilayah-indonesia) with HATEOAS, OpenAPI, validation, a modern playground, and a single CLI binary.

🌐 **Live demo & playground**: https://dnahilman.github.io/region.id

📊 **34 provinces · 514 regencies · 7,215 districts · 80,534 villages · 96,057 endpoints**

---

## Why a rewrite

The original PHP project pioneered the idea of a static-file API for Indonesian region data, which is brilliant for hosting and bandwidth costs. region.id keeps that core idea but adds:

- ⚡ **Fast build** — ~10 seconds for the full 96k-file regeneration (vs. several minutes in the original)
- 🔍 **Built-in validation** — duplicate IDs, orphan references, format conformance checked before generation
- 🧩 **HATEOAS links** — every single-entity response carries `_links.self`, `_links.parent`, `_links.children` for easy navigation
- 📜 **OpenAPI 3.1 spec** — published at `/api/openapi.json`, importable into Postman / SDK generators
- 🧾 **Metadata endpoint** — `/api/meta.json` exposes build time, commit, counts, attribution
- 🎨 **Modern playground UI** — compact, dark-mode-aware, no JS framework
- 📦 **Single CLI binary** — `region generate`, `region validate`, `region serve`
- 🔁 **Backward-compatible** — every emsifa endpoint URL still works, byte-identical with `--no-links`

## Endpoints

List endpoints (return an array):

```
GET /api/provinces.json
GET /api/regencies/{provinceId}.json
GET /api/districts/{regencyId}.json
GET /api/villages/{districtId}.json
```

Single endpoints (return an object):

```
GET /api/province/{id}.json
GET /api/regency/{id}.json
GET /api/district/{id}.json
GET /api/village/{id}.json
```

New endpoints:

```
GET /api/meta.json
GET /api/openapi.json
```

### Sample responses

```bash
$ curl https://dnahilman.github.io/region.id/api/province/11.json
{"id":"11","name":"ACEH","_links":{"self":"/api/province/11.json","children":"/api/regencies/11.json"}}

$ curl https://dnahilman.github.io/region.id/api/regency/1101.json
{"id":"1101","province_id":"11","name":"KABUPATEN SIMEULUE","_links":{"self":"/api/regency/1101.json","parent":"/api/province/11.json","children":"/api/districts/1101.json"}}
```

## ID format

| Level    | Length | Example      | Notes                                   |
| -------- | ------ | ------------ | --------------------------------------- |
| Province | 2      | `11`         |                                         |
| Regency  | 4      | `1101`       | First 2 digits == province ID           |
| District | 7      | `1101010`    | First 4 digits == regency ID            |
| Village  | 10     | `1101010001` | First 7 digits == district ID           |

## CLI

```bash
# Generate the static API into ./static
region generate --data ./data --out ./static

# With absolute HATEOAS links
region generate --base-url https://dnahilman.github.io/region.id

# Byte-identical to emsifa (no _links)
region generate --no-links

# Validate CSVs without generating
region validate --data ./data --strict

# Preview locally
region serve --dir ./static --addr :8080
```

### Install

Pre-built binaries are published on each tagged release. Or build from source:

```bash
git clone https://github.com/dnahilman/region.id
cd region.id
go build -o region ./cmd/region
./region version
```

## Architecture

```
data/*.csv  ──►  region generate  ──►  static/api/**/*.json  ──►  GitHub Pages
                       │
                       ├── validation (fail-fast)
                       ├── in-memory index (O(1) parent→child)
                       ├── parallel writes (NumCPU workers)
                       └── manifest (skip if CSVs unchanged)
```

See [`docs/architecture.md`](docs/architecture.md) for deeper notes.

## Attribution

Data originally compiled by [emsifa](https://github.com/emsifa) from BPS and Kemendagri public sources. This project is a separate code base under MIT, distributing the same data verbatim.

## License

MIT — see [LICENSE](LICENSE).
