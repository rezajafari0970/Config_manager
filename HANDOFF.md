# Handoff — P2 source management

## Verified product state

Go development lives on branch `go/v2rayng-2.3.6` of `rezajafari0970/Config_manager`. The legacy main branch is retained. Verify the published branch and actual working tree before continuing; source publication is not application deployment.

The server working directory is `/root/projects/v2ray-config-repair`. Its `.local/` directory contains pinned upstream clones, the Xray oracle, an isolated Go toolchain, development logs and generated receipts. None of that private working material is committed.

Implemented: target identity, driver contracts, random interval helper, generation/fallback decision policy, initial Custom JSON diagnosis, a byte-local stats empty-array repair, CLI, Go CI and controlled core tests. An authenticated Persian source-management panel and encrypted SQLite store now run as an independent systemd service. Fetch worker and exporter are not implemented. Sources and sessions survive a process restart.

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

## Runtime and next work

The independent panel service is `config-repair.service`; loopback is `127.0.0.1:18191`, the TLS reverse proxy listens on port `18443` of the server IP. Runtime config is `/etc/config-repair/panel.json`, initial password `/etc/config-repair/initial-admin-password` (root-only), database `/var/lib/config-repair/panel.db`, and binary `/opt/config-repair/current/config-panel`. Do not print/commit runtime secrets. Public browser login/logout and TLS 1.3 were tested. Read `docs/PANEL.md` and `docs/checkpoints/P2-panel-20261029.md`. `.local/p2-published-readback.json` records the exact published/deployed revision after publication.


1. Expand P1 source/fixture matrix from exact upstream commits; include Android settings, URI query edge cases, Custom mixed subscriptions, legacy vs flat settings and WireGuard IPv6/multi-peer cases.
2. Implement P3 bounded, durable fetch jobs, SSRF-safe actual dialing and redirects, HTTP validators, source revisions and atomic generations. Persisted P2 next_run deadlines are not running jobs. Define behavior for overdue deadlines on first worker startup. Run-now becomes available with the worker.
3. Prepare Android acceptance infrastructure. Mark tests not_run until actual APK/device evidence exists.
4. Keep the final automatic installer in P8. No curl installer or production-ready release is advertised at this checkpoint.
