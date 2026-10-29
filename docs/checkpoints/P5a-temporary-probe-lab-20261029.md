# P5a — temporary isolated core and HTTPS laboratory

## Scope and behavior

The user explicitly accepts testing from any network and asks for a temporary way to test current configs and build a repair knowledge bank. Implemented a Go preparation module/CLI, a temporary Linux sandbox driver and a persistent, secret-free SQLite evidence bank. Existing panel, collector, source settings, raw originals and retention remain unchanged. No production schema migration or panel daemon replacement was necessary. Android import is still `not_run`; no healthy export is published.

The planner prepares separate URI-to-Custom-JSON candidates for supported VLESS/Trojan/SS/VMess/Hy2 representations, reuses structural JSON repair, rejects unknown/duplicate/lossy inputs, preserves large JSON numbers and rejects invalid UTF-8. Rules never modify original bytes. Two additional hypotheses target pinned-core incompatibilities: explicit Trojan `security=none` can be tested with verified TLS, and removed `allowInsecure=true` can be tested with strict verification. Neither hypothesis is automatically approved without a real connection; pins, credentials, endpoints and SNI are never guessed.

The sandbox uses separate network/mount/PID/user/IPC/UTS/cgroup namespaces, namespace UID65534, no capabilities, read-only minimal mounts, pinned core/CA material, 256MiB/128-task limits and deadlines. Egress permits only resolved public proxy endpoints and public DNS. Input to host services is blocked by an interface-scoped rule. All host firewall changes are isolated to uniquely named tables; original host tables are preserved. Temporary local SOCKS inbounds and blocked direct outbounds prevent false proxy successes. Exact candidate validation and modified test-runtime hashes are distinct.

The bank records original/candidate/runtime hashes, target/core/planner/driver/CA versions, stage statuses, diagnosis, elapsed time and original expiry. It stores no raw configs, passwords or logs. A versioned strategy catalog lives in `compatibility/probe-knowledge.json`. This lab is not an always-on worker and must not be callable from the public panel.

## Source evidence

- V2RayNG commit `13138dddcab291ba3618cc54721ce4c7dd926be0`: TrojanFmt, FmtBase, VlessFmt, VmessFmt, ShadowsocksFmt, Hysteria2Fmt, WireguardFmt, CoreOutboundBuilder, CoreConfigManager and RealPingWorkerService.
- Xray commit `5ca6f4b7d4dc20a881d4330e498892697627ec0c`: public Trojan/VLESS transport-security requirements; `infra/conf/transport_security.go:361` rejects `allowInsecure=true`; WireGuard config schema reviewed, extension conversion remains pending.
- Binary SHA256 `64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d`.

## Actual full-run observation

Run `da05f444d11d4ff1967dce0f0f4381df` considered 868 snapshot artifacts with six workers and an eight-second per-destination deadline. It produced 870 attempt records including two alternative candidates. 93 candidates reached both HTTPS destinations; 196 reached one destination. These are dated Linux observations, not Android acceptance or permanent health. 192 originals became unavailable/expired and 10 changed before testing; no result was assigned to a replacement. 53 inputs had no implemented, unambiguous preparation mapping. Originals were not kept alive to complete tests.

Six structurally repaired JSON candidates reached runtime testing; one reached one HTTPS destination. No additional TLS hypothesis achieved the two-destination approval gate in this run. In particular, a structurally/core-correct candidate can still be unreachable because its remote service/DNS/certificate is unusable. No unsupported config was relabeled as repaired.

An early calibration omitted the system CA bundle. Both affected run receipts were explicitly invalidated and excluded from knowledge aggregation; their TLS failures are not evidence against user configs. Subsequent runs mount and fingerprint the CA bundle. Public control endpoints can time out intermittently; a failed sandbox positive control stops the run, and short proxy timeouts do not establish permanent failure.

Private operational receipts are under `.local/p5-*.log` and `.local/probe-bank/evidence.db`; none are published. Final Go/race/build, Python regression, sandbox controls and final-code canary results are recorded in PROJECT_STATE.json and private receipts. No UI behavior was changed, so browser acceptance was not rerun for this checkpoint.

## Limits and continuation

No APK/device/emulator receipt exists. Complex app settings, arbitrary transports/extensions, IPv6-only endpoints, full Custom routing equivalence and UDP payload tests are not fully covered. WireGuard extensions and unknown VMess/query fields remain explicitly untested. The live probe uses a separate runtime envelope and disables direct routes/control listeners. Future integration must bind durable jobs/evidence/candidate downloads to repository generations, deletion/disable controls and expiry, encrypt candidate storage and add crash cleanup before enabling an always-on worker. The current laboratory cleans up bounded jobs and normal signal cancellation; SIGKILL recovery is not yet an autonomous production guarantee.
