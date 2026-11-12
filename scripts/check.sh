#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -z "$(gofmt -l cmd internal)"
go mod verify
go vet ./...
go test -race -timeout=30m ./...
go build -trimpath -buildvcs=false -o bin/config-compat ./cmd/config-compat
go build -trimpath -buildvcs=false -o bin/config-repair ./cmd/config-repair
go build -trimpath -buildvcs=false -o bin/config-panel ./cmd/config-panel
go build -trimpath -buildvcs=false -o bin/config-probe-plan ./cmd/config-probe-plan
go build -trimpath -buildvcs=false -o bin/config-probe-worker ./cmd/config-probe-worker
python3 scripts/client-compat-test.py
python3 scripts/live-probe-test.py
python3 scripts/payload-quality-test.py

node scripts/source-status-test.mjs

node scripts/client-api-test.mjs

node scripts/client-poll-test.mjs
go build -trimpath -buildvcs=false -o bin/config-egress-worker ./cmd/config-egress-worker
python3 scripts/egress-test.py

python3 scripts/benchmark-test.py

python3 scripts/egress-country-test.py
