#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
source_tree="${XRAY_CORE_SOURCE:?set XRAY_CORE_SOURCE to the pinned Xray checkout}"
test "$(git -C "$source_tree" rev-parse HEAD)" = 5ca6f4b7d4dc20a881d4330e498892697627ec0c
mkdir -p "$root/.local"
cd "$source_tree"
go mod verify
CGO_ENABLED=0 go build -mod=readonly -trimpath -buildvcs=false -o "$root/.local/libping-fixture" "$root/scripts/testdata/libping-fixture.go"
