# region.id Makefile
# Cross-platform helpers around the `region` CLI.

VERSION ?= 0.1.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

BASE_URL ?=

.PHONY: build generate validate serve test bench clean fmt vet tidy

build:
	go build -ldflags="$(LDFLAGS)" -o region ./cmd/region

generate: build
	./region generate --data ./data --out ./static $(if $(BASE_URL),--base-url $(BASE_URL))

validate:
	go run ./cmd/region validate --data ./data --strict

serve:
	go run ./cmd/region serve --dir ./static --addr :8080

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

bench:
	go test -bench=. -benchtime=1x ./internal/generator/...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf static region region.exe
