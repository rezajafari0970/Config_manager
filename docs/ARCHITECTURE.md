# Architecture decisions

## ADR-001: one Go control plane, bounded independent workers
Start with a modular server process and explicit driver interfaces. Use worker pools for independent source and candidate work. Native cores and optional external plugins run out of process; no Go .so ABI dependency. No unbounded goroutine/process creation. A process supervisor provides restart; durable job leases and revisions provide recovery correctness.

## ADR-002: source truth versus health
A source row owns URL (secret), schedule bounds/unit, revision, next deadline and lifecycle status. An immutable fetch generation owns received records and their provenance. Shared artifact identity is byte-based; do not collapse different credential/transport/remark variants by guessed semantic identity. Membership is many-to-many.

Database transaction: verify source still enabled and revision/generation still current, record candidates/evidence, swap active membership, invalidate output cache. A stale job must fail its compare-and-swap. A single source empty response does not delete good artifacts belonging to another source.

Health evidence binds artifact hash, target, rule/core version, location, checked_at and expires_at. Fetch freshness cannot renew health. Temporary fallback uses prior membership only while health is fresh. Unknown tester/network failure suspends approval rather than deleting all history. Removed private payloads and quarantine have bounded retention.

## ADR-003: storage
SQLite on local persistent disk, WAL, short serialized writes and explicit migrations is the initial single-node choice. Benchmark actual source count and probe volume before finalizing throughput limits. Storage driver can later use PostgreSQL. Do not share a SQLite WAL file across hosts. No operational database is created in phase P0.

## ADR-004: repair contract
raw -> diagnosis -> minimal representation repair -> candidate -> structural check -> matching core -> matching Android import/runtime -> network evidence -> publication eligibility. Exact eligibility policy is explicit per stage; no implicit pass on unknown.

Custom JSON conversion is an equivalent representation, not a new server or wire protocol. Preserve routing, DNS, chains, credentials, TLS guarantees and unknown fields. No broad reserialization of healthy data. Ambiguous/duplicate-key/truncated input is not guessed into correctness. Rules carry versions, input/output hashes, evidence and rollback scope. Learning data contains error categories and rule statistics, not reusable secrets.

## ADR-005: execution boundary
Fetcher enforces HTTP(S), DNS/IP checks on every redirect/connect, bounded decode depth/bytes/time, no localhost/private/metadata egress by default and per-origin limits. Core sandbox blocks filesystem escape, unwanted inbounds/APIs and unrelated network egress. Do not execute untrusted JSON as root. Controlled core smoke fixtures are the only core execution implemented initially.

## ADR-006: Android is a separate oracle
Capture APK hash/package/version, native library identity, Android version/ABI and V2RayNG settings. Test actual clipboard/subscription/file paths, counts, preserved endpoint metadata, start/stop, full tunnel traffic and Custom behavior with statistics and TUN choices. Hosted emulator or attached test device is required; parser emulation alone cannot satisfy P6.

## ADR-007: management UI
Persian RTL, mobile layout, login-first, secure sessions, CSRF defense and rate limits. Add/edit/delete/enable/disable sources, random scheduling bounds, run-now, last/next fetch, empty-source fallback state, counts, repair diffs and independent core/Android/network results. A URL's credentials must be redacted in logs and ordinary list views. Worker work continues after the browser closes.

## ADR-008: release discipline
Independent server directory and service names. GitHub code only, no private snapshots. Test clean candidate; publish immutable version/checksums; install atomically; health check; restore prior version if startup fails. Preserve user source data and credentials across upgrades. Source, deployed binary and database migration receipts must be separately verified.


## ADR-009: P3 user amendment (supersedes conflicting ADR-002/005/007)

Collection is immutable raw acquisition, without auto-repair or claims of health. Fixed two-hour configurable retention replaces health-based fallback for this raw pool. Byte hashes address encrypted blobs; conservative protocol identity deduplicates outputs while each membership stores its own original hash. Large/unknown configurations fall back to exact byte identity. Memberships are swapped in short transactions after staged extraction; stale revisions/leases are rejected. Cleanup and publication share reference/pin protection.

Passwordless Input / Output / Setting replaces login-first UI. The user explicitly authorized public management. CSRF, HTTPS, encrypted secrets and public-IP-only dialing remain. Artificial byte/count quotas are removed; streaming/file-backed parsing, bounded work, deadlines, disk admission guard and OS resource budgets keep resource use controlled. Every source gets a separate transport, context and state; SQLite/scheduler/disk remain shared infrastructure, so physical isolation of all possible failures is not claimed.

Driver boundaries: collect.FetchDriver, collect.FormatDriver/Registry, collect.Repository, and panel.PanelModule.Register. Built-in modules and extensions are compiled and tested together. This is not a runtime marketplace or an unsandboxed dynamic code-upload mechanism. Core execution remains outside this raw pipeline.
