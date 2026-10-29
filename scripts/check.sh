#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -z "$(gofmt -l cmd internal)"
go mod verify
go vet ./...
go test -race ./...
go build -trimpath -buildvcs=false -o bin/config-repair ./cmd/config-repair
go build -trimpath -buildvcs=false -o bin/config-panel ./cmd/config-panel
go build -trimpath -buildvcs=false -o bin/config-probe-plan ./cmd/config-probe-plan
python3 scripts/live-probe-test.py
