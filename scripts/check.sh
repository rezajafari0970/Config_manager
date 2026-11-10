#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -z "$(gofmt -l cmd internal)"
go mod verify
go vet ./...
go test -race -timeout=20m ./...
go build -trimpath -buildvcs=false -o bin/config-repair ./cmd/config-repair
go build -trimpath -buildvcs=false -o bin/config-panel ./cmd/config-panel
go build -trimpath -buildvcs=false -o bin/config-probe-plan ./cmd/config-probe-plan
go build -trimpath -buildvcs=false -o bin/config-probe-worker ./cmd/config-probe-worker
go build -trimpath -buildvcs=false -o bin/config-residential-worker ./cmd/config-residential-worker
python3 scripts/live-probe-test.py
python3 scripts/residential-probe-test.py
python3 scripts/residential-matrix-test.py
python3 scripts/residential-native-test.py
python3 scripts/residential-intel-test.py

node scripts/source-status-test.mjs

node scripts/client-api-test.mjs

node scripts/client-poll-test.mjs