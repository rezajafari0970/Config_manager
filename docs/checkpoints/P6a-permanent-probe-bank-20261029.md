# P6a — persistent samples, autonomous testing and server-verified subscriptions

The user's latest instruction requests permanent samples for ongoing testing, automatic new-model detection, separate repaired variants and subscription outputs. This supersedes transient-only test sample retention. Source response/membership TTL remains unchanged; the new archive is independent.

## Implemented

- Schema5 transactional sample archive, exact-byte original storage, encrypted candidate history, model/rule evidence, durable job leases and persistent bank controls/tombstones.
- Every accepted source generation and manual import queues new exact samples. Source expiry/deletion/replacement cannot remove archived samples. Explicit bank disable/delete/retest revokes stale completions and current evidence; global pause revokes running leases.
- Independent root supervisor over a protected Unix socket. Untrusted configs execute only in isolated network/mount/user/PID namespaces with cgroups, public endpoint/port allowlists, TLS validation and no direct-route success. Startup and per-run cleanup handle abandoned resources; an old run cannot clean a replacement run's resources.
- Pinned adapters for VLESS, VMess, Trojan, Shadowsocks, WireGuard URI/INI, SOCKS/SOCKS5, Hysteria2/Hy2 and JSON/Custom JSON envelopes. WireGuard preserves multiple peers, IPv6 endpoints, keepalive, reserved values and DNS; Hy2 port hopping is preserved and allowlisted. Hysteria1 is explicitly unsupported by this core, not converted to an incompatible protocol. Unknown/noise/Amnezia extensions remain diagnosed, not guessed.
- Separate JSON conversion, structural repairs and strictly verified TLS hypotheses. Successful model/rule evidence changes trial priority only. Every sample needs its own core and connection evidence. Every fourth available claim prioritizes due retests so new arrivals cannot starve them.
- Output bank management, manual import, filters, batch controls, trial reports, historical downloads and two tokenized HTTPS Custom JSON subscriptions. Setting exposes test concurrency, per-destination timeout, retest interval, evidence TTL and pause.
- Subscription gates: enabled/not deleted, current planner version, exact pinned-core pass, two successful HTTPS destinations, current evidence TTL, canonical identity/candidate hash deduplication. Historical/partial/failed/expired/disabled results are excluded. Candidate bytes are preserved within the JSON array envelope.

## Evidence

Full gofmt/module verification/vet/race/build checks passed using the verified isolated Go toolchain. Eleven unprivileged Python regressions passed, including same-host multiple ports and port-hopping firewall injection rejection. Go tests cover source/archive lifetime separation, deletion tombstones, exact raw/candidate history, stale tokens and lease recovery, global pause, TTL clamping, fair retests, private queue isolation, subscription/version/dedup gates, protocol mapping and secret-free structural model fingerprints.

Browser acceptance passed for existing management and new bank import, disable, reports, empty unverified subscriptions and test settings at desktop/mobile sizes. The isolated supervisor test completed 17 samples including 12 current real samples, with a forced SIGKILL and recovered leases: 3 two-destination HTTPS passes and 7 partial passes. Only the 3 full passes entered its subscription. No owned namespaces/cgroups/payloads remained after cleanup. An initial startup calibration failed and was not counted as acceptance; the final run passed.

Production migration used a stopped-service private backup of matching database, blobs, key and prior binary. All 54 source IDs/revisions/URL hashes and original fetch settings were preserved. The panel moved to schema5; the permanent prober and panel are enabled at boot. The subsequent scheduler-only binary update kept schema5 data intact. Actual release hashes and backup paths are recorded in `.local/p6-deployed.json` rather than inferred from publication.

An early production readback recorded 1,509 retained samples, 26 fresh successful raw variants and 23 deduplicated subscription entries. Repaired-only subscription was then empty: no repaired candidate had yet passed both destinations. These counts are a timestamped observation, not a promise of current availability; the queue and sources continue changing. Public HTTPS certificate verification and both subscription endpoints passed. Final live/browser receipts are stored separately.

## Limits

Android APK/device/emulator testing remains **not_run**. Linux core/HTTPS evidence is explicit in the UI and HTTP headers; it does not certify Android import/VPN mode, arbitrary custom routing branches or UDP payload behavior. JSON wrapping cannot fix missing credentials, a dead endpoint or Hysteria1/Hy2 server incompatibility. Unsupported extension mappings remain preserved for future source-backed work. Testing is bounded by available server capacity and deadlines; a growing queue is processed continuously, not claimed complete at publication. Power-off prevents execution; services resume after boot and networking recovery.

Private operational receipts and real payloads are excluded from Git. The legacy main branch remains separate. A complete reproducible install/update/rollback distribution remains a distinct milestone; this checkpoint ships the permanent worker code and systemd unit.
