# P3 — immutable collector and passwordless modular panel

## Request and resulting behavior

The user replaced login with public passwordless management, required Input/Output/Setting, bulk import and auto names, seconds/minutes request intervals, removal of legacy quotas, strict original bytes, deduplication and fixed configurable retention (default two hours). These instructions supersede conflicting P2 plans. Existing repair CLI remains separate.

Implemented Go fetch/format/repository boundaries, encrypted blob storage, per-source isolated requests/deadlines/revisions, durable leases, staging and atomic publication, fixed TTL and GC. Known config identity and per-source original bytes are distinct; deleting one source leaves the other's own original representation. Empty/error fetches preserve previous membership until expiry. Repeat content/304 do not extend expiry. Raw individual/response/ZIP downloads make no connectivity claim.

UI was rewritten with independent modules, flat responsive layout, mobile drawer, source accordions, bulk import accordion, one interval expression field and locally bundled OFL Vazirmatn font. Authentication tables/flags/screens were removed; same-origin CSRF/HTTPS remain. Artificial 1000-source, 16KiB request, long URL/name and 30-day interval limits were removed. Display pagination and physical resource/time/egress safeguards are not ingestion quotas.

## Verification

- Isolated Go 1.27.1: module integrity, gofmt, vet, race tests and both binaries passed.
- Vault exact bytes/encryption/dedup/pins/GC/tamper/truncation; format URI/JSON/Base64/INI extraction and raw spans passed.
- Safe HTTP actual-dial IP checks including IPv4-mapped/translated addresses, redirects, conditional response and per-job transport behavior passed.
- Schema-1 migration, wrong-key rejection, revision conflicts, atomic replacement, stale completion, per-source raw variants, global dedup and fixed expiry passed.
- Bulk 1003 sources exceeds old body/count limits; positive long intervals and URL/name values pass without old quotas.
- Slow source plus concurrent fast and panicking sources, cancellation/delete and recovery passed; deterministic concurrency regression repeated ten times under race detector.
- Chrome 154 / Playwright core 1.56.1: desktop 1440 and mobile 390, drawer, status, bulk/edit/pause/delete, raw SHA, whole response, exact ZIP, restart persistence and expiry/settings passed. Screenshots inspected.
- A separate transient service received real watchdog heartbeats, was deliberately stalled with SIGSTOP, and restarted automatically with a new PID; production was untouched.
- Public TLS Input/Output/Setting loaded without login or browser errors; live worker and default 7200-second retention verified.
- Upgrade used protected SQLite backup API snapshot and schema-2 migration. Production zero-source count preserved. Other nginx sites unchanged. Running executable SHA256: `470372c33160cc8fed056b654ecd5c7f5f84e46ea6381aca259716be50676dad`.

A timing-sensitive test originally assumed goroutine order; explicit test-driver synchronization now verifies overlap reliably. Shared host Go race and module-cache integrity faults are avoided with isolated verified caches; unrelated installations were untouched.

## Explicit boundaries

No live proxy connection or Android acceptance was performed. Output is raw and unverified; stats empty arrays, unknown fields and malformed connection parameters are preserved, not repaired. Unknown/large config identity falls back conservatively to exact bytes. Compiled plugin interfaces are available; dynamically uploaded code is not. Universal uptime, limitless hardware and proof of no bugs are not claimed. Final automatic installer remains P8.

Private operation/browser/recovery/publication receipts live under ignored `.local/`. A source commit cannot contain its own hash; exact branch readback and deployed-revision receipt are recorded after publication.
