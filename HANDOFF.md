# Handoff — P0 foundation

## Verified product state

Go development lives on branch `go/v2rayng-2.3.6` of `rezajafari0970/Config_manager`. The legacy main branch is retained. Verify the published branch and actual working tree before continuing; source publication is not application deployment.

The server working directory is `/root/projects/v2ray-config-repair`. Its `.local/` directory contains pinned upstream clones, the Xray oracle, an isolated Go toolchain, development logs and generated receipts. None of that private working material is committed.

Implemented: target identity, driver contracts, random interval helper, generation/fallback decision policy, initial Custom JSON diagnosis, a byte-local stats empty-array repair, CLI, Go CI and controlled core tests. No source URL collection, database, scheduler service, panel or exporter runs yet.

## Checks

Formatting, vet, race tests and build passed with official checksum-verified Go 1.27.1. Thirteen synthetic tests against official checksum-verified Xray 26.7.28 passed, including rejected input before stats repair and accepted output after repair. These are core-schema checks only; actual proxy connectivity and V2RayNG Android import/runtime are untested.

Server Android preflight: no adb, emulator, KVM or authorized device. A hosted emulator or attached test device must be provisioned before claiming P6 acceptance. Pin actual APK and native library hashes; the user screenshot alone supplies the target version identity.

## Environment issue and working build command

The existing system Go installation's `src/runtime/race/internal/amd64v1/race_linux.syso` differs from the official archive and causes `ELF section name out of range`, also reproduced with an empty independent test module. A fresh official toolchain and cache under this project's `.local/` pass the complete gate. No global Go, GCC, service or unrelated project was modified. Underlying cause of the differing file was not determined.

From the server project directory:

```sh
env GOROOT="$PWD/.local/toolchain/go" PATH="$PWD/.local/toolchain/go/bin:$PATH" GOCACHE="$PWD/.local/go-clean-cache" GOMAXPROCS=2 GOMEMLIMIT=512MiB GOFLAGS=-p=2 bash scripts/check.sh
python3 scripts/core-smoke.py --core .local/xray-26.7.28/xray
```

GitHub connector publication is available. The current server's default SSH identity did not authorize this repository; the origin uses public HTTPS for fetch/readback. Do not assume automated server pushes are configured. Do not read or copy unrelated project secrets to work around this.

## Next work

1. Expand P1 source/fixture matrix from exact upstream commits; include Android settings, URI query edge cases, Custom mixed subscriptions, legacy vs flat settings and WireGuard IPv6/multi-peer cases.
2. Implement P2 persistent source model/migrations and authenticated Persian mobile panel with add/edit/delete, enable/disable, interval bounds/unit and source revision. Only then wire P3 jobs.
3. Prepare Android acceptance infrastructure. Mark tests not_run until actual APK/device evidence exists.
4. Keep the final automatic installer in P8. No curl installer or production-ready release is advertised at this checkpoint.
