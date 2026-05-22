# region.id

> Static API for Indonesian administrative regions — provinces, regencies, districts, villages.

🌐 **Live demo & playground**: https://dnahilman.github.io/region.id

📊 **34 provinces · 514 regencies · 7,215 districts · 80,534 villages · 96,057 endpoints**

---

## Endpoints

Base URL: `https://dnahilman.github.io/region.id`

List endpoints (return an array):

```
GET /api/provinces.json
GET /api/regencies/{provinceId}.json
GET /api/districts/{regencyId}.json
GET /api/villages/{districtId}.json
```

Single endpoints (return an object with `_links`):

```
GET /api/province/{id}.json
GET /api/regency/{id}.json
GET /api/district/{id}.json
GET /api/village/{id}.json
```

Meta endpoints:

```
GET /api/meta.json
GET /api/openapi.json
```

### Examples

```bash
curl https://dnahilman.github.io/region.id/api/provinces.json

curl https://dnahilman.github.io/region.id/api/province/11.json
# {"id":"11","name":"ACEH","_links":{"self":"/api/province/11.json","children":"/api/regencies/11.json"}}

curl https://dnahilman.github.io/region.id/api/regency/1101.json
# {"id":"1101","province_id":"11","name":"KABUPATEN SIMEULUE","_links":{"self":"/api/regency/1101.json","parent":"/api/province/11.json","children":"/api/districts/1101.json"}}

curl https://dnahilman.github.io/region.id/api/regencies/11.json
# [{"id":"1101","province_id":"11","name":"KABUPATEN SIMEULUE"},...]
```

## ID format

| Level    | Length | Example      | Notes                         |
| -------- | ------ | ------------ | ----------------------------- |
| Province | 2      | `11`         |                               |
| Regency  | 4      | `1101`       | First 2 digits = province ID  |
| District | 7      | `1101010`    | First 4 digits = regency ID   |
| Village  | 10     | `1101010001` | First 7 digits = district ID  |

## CLI

```bash
# Generate static API into ./static
region generate --data ./data --out ./static --base-url https://dnahilman.github.io/region.id

# Validate CSVs only
region validate --data ./data --strict

# Preview locally
region serve --dir ./static --addr :8080
```

### Install

Pre-built binaries are available on each tagged release. Or build from source:

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

## License

MIT licensed. Data attributed to [emsifa](https://github.com/emsifa/api-wilayah-indonesia) (BPS / Kemendagri public sources). Source on [GitHub](https://github.com/dnahilman/region.id).
