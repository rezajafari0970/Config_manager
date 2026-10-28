#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -z "$(gofmt -l cmd internal)"
go vet ./...
go test -race ./...
go build -trimpath -o bin/config-repair ./cmd/config-repair
