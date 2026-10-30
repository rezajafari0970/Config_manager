# Architecture — P7

The Go panel and collector own source scheduling, immutable encrypted input, SQLite migrations and public management. A separate systemd supervisor owns only isolated Xray execution. Compiled format/protocol/template/feature drivers provide modular boundaries; sources never execute arbitrary plugin code.

## Source and conversion lifecycle

Source revision/token checks guard generation replacement. Memberships associate canonical identity with each source's original hash. Exact encrypted originals are never rewritten. Nonempty fetches replace one source; empty/error retains membership until its original fixed TTL. Identical repeated fetches/304 do not extend TTL. Controls, deletion and expiry invalidate publication atomically.

Schema6 conversion jobs/results reference current eligible membership. They are a transient cache and durable work queue, not a sample bank. Completed results must match the active lease, exact original, deterministic template candidate, pinned rule/core version and stage evidence. There is no history/model/learned-priority store. One source disappearing cannot delete another source's ownership.

## Template and export

The common @shenvpn envelope owns log/DNS/SOCKS/routing/stats/policy. A protocol driver retains real endpoint/auth/security/transport; chain dependencies are renamed consistently. Invalid or ambiguous mapping is diagnosed. Missing VLESS encryption and scoped endpoint DNS bootstrap are deterministic template repairs. Semantic TLS alternatives stay connection-gated.

Export is JSONL: one compact exact core-tested object per line. Core-valid ordinary configs are eligible while their source remains eligible. Connectivity is a separate, time-limited status. Semantic hypotheses additionally require fresh two-destination HTTPS success. Duplicate identities and JSON hashes are excluded. JSONL is intentionally not represented as the pinned Android app's native JSON-array subscription format.

## Execution and failure isolation

Private Unix worker API is protected by directory/socket permissions. Eight maximum sandbox slots enforce configured concurrency. Leases renew during bounded attempts; death/restart recovers uncompleted jobs. The helper runs only checksum-pinned Xray in netns/cgroup/bwrap with no host secrets, non-root UID, read-only filesystem and approved public endpoint egress. Raw errors are reduced to bounded codes. Exact core validation and modified isolated-runtime network testing have separate hashes. Android remains not_run.

SQLite, disk, host and network are shared physical infrastructure; source independence does not imply immunity to host failure. systemd handles reboot/restart, but a powered-off server cannot process work. Upgrade requires a stopped-service matching data/key/blob/binary backup, atomic release symlink, health checks and separate source/deployment receipts.
