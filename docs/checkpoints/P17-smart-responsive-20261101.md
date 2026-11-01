# P17 — responsive Smart and ready-only single output
Date: 2026-11-01. Smart v5, normal template v4, schema9. Previous published source:62beec0b5525886d180887fbae15d2c87dd72e08.

## Problem and behavior
The single Smart queue included unfinished heads and could return empty204 while other ready cache existed. It now removes unavailable heads, snapshots only smart_ready and wraps after the finite available round. Newly ready profiles join the next round. Zero-ready returns503 with explanatory text/Retry-After; HEAD/prefetch are nonconsuming. Publication URLs and source/global ordinary rotations are unchanged.

The renewed performance request supersedes exact P15 globals. Smart selects the alive lowest-latency member using leastPing, burst10s/sampling1/timeout3s and explicit block fallback. DNS53 interception includes socks and the pinned app's injected tun. Exact endpoint-domain bootstrap avoids depending on an unconnected tunnel. A/AAAA use proxied DoH; non-IP DNS is explicitly forwarded over TCP through loopback/balancer. All member outbounds remain copied with namespace-only changes, all logs off. See SMART_PERFORMANCE.md for settings and limitations.

A real two-profile canary found Instagram TLS failure on one profile although both general HTTPS tests passed. The existing sandbox validator now adds a bounded TLS-valid Instagram homepage check. Fresh source-owned https_multi_site_verified is required for Smart; general connection status and ordinary export remain independent. Old general-only results were scheduled for revalidation; raw bytes, existing results, source settings and normal output were retained. No new schema/table/bank.

## Evidence
- Full gofmt/module/vet/race/build checks and12Python regressions passed.
- Exact core matrix1/2/10/50; all copied outbound values/ordering and logs-off tests passed.
- Controlled two-peer comparison:24requests/phase; before11slow and p95=243.63ms; after0slow and p95=2.04ms. Artificial120ms per direction, not mobile speed.
- Runtime preferred-peer failure recovered in5.48s with4failed requests during detection; DNS A/TXT and endpoint bootstrap passed. Existing TCP/UDP/UDP443 packet fixture passed; real QUIC not_run.
- Synthetic browser/restart passed. Smart worker:59profiles of50members,4sources,2slots, SIGKILL recovery; four real input canaries were core-validated, synthetic admission receipts are not connectivity evidence.
- Normal worker:9inputs,8core-valid,4general+Instagram passes,3slots, SIGKILL recovery.
- Live public desktop/mobile-emulation browser:60single GETs plus2navigations returned200 with one inline JSON each, no downloads/JS errors. During initial post-admission warmup a503 was correctly visible until valid cache existed; no stale proofs were exported to hide that.
- Final live site canary:2profiles, Instagram/Google/Cloudflare/Wikipedia all returned expected200/204 over sandboxed proxies. This is server/homepage evidence only.
- Live ordinary and3source-scoped single links passed;12parallel Smart GETs were unique within their round.
- Audit: 4 ready profiles reconstructed byte-for-byte, 40 input selections, 160 copied outbound definitions; logs off.
- Current proof audit: 122 ready, 1220 member selections,0missing/expired/site-unverified evidence; SQLite quick_check=ok, foreign_key_check empty.

## Release
Release: /opt/config-repair/releases/p17-6ace5eb1d551b0bc63f7
Panel SHA256: 6ace5eb1d551b0bc63f78246c8d64c1a0049b4cccbee52faf3d5dd439bc89a24
Worker SHA256: 274b5b5eb3b4a4edcd71aeb191b79ea4d6196568e08659aedf77487e91ed4b29
Sandbox driver SHA256: 7c72e55456eb7bac75f1284560f61ef3d220ef16dc7c3fd17d678f547d73a136
Previous release: /opt/config-repair/releases/p17-227a3e8c20df3867706b
Matching backup: /var/backups/config-repair/pre-smart-responsive-20261101-073706
All3systemd services active/enabled. Source/settings/rotation fingerprints matched across installation. User changed Smart count from5to10 during the work; current count10/revision7 is preserved. Do not restore the older50-member value.

Rollback is same-schema: stop all3services, select the previous matching binaries/driver and restart, retaining current data. Restore backup DB/blobs/key only for actual data recovery with deliberate reconciliation of newer user changes. Never print the private key/config file.

Final private receipts use p17-final-check/browser/worker, p17-conversion-worker, p17-responsive, p17-deployed/public/audit/quality/site-canary/http-check and p17-published. Earlier intermediate v4 and pre-site canary are historical only. Publish by expected-HEAD lease, read back every changed file and verify runtime binary/driver hashes separately.

## Unmeasured
No actual Android import/VPN, MCI/Irancell/Rightel/Wi-Fi, app media/CDN coverage, download throughput, real UDP loss or HTTP3 measurement. No guarantee of seamless TCP migration, zero packet loss or a universal best carrier configuration. Existing Custom JSON must be reimported.
