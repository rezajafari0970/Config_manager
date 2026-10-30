# P7 — shared JSON template, bank removal

User instruction on 2026-10-30 supersedes P6. Inputs now feed one @shenvpn JSON envelope; Output contains a list and one JSONL export. Each exported configuration is one compact JSON object on one line. The persistent experiment/repair bank, manual import, variant history, model/rule learning and old subscriptions are removed.

## Changes

- Added internal/convert: fixed common log/DNS/SOCKS/routing/stats/policy, actual protocol/credentials/transport retained, vnext mapping where lossless, dependency-safe custom chains, explicit unsupported/ambiguous diagnoses, endpoint-domain bootstrap DNS, default VLESS encryption and separately gated JSON/URI Trojan TLS hypotheses.
- Schema6 removes all eight bank/probe tables and seeds transient leased conversion work from current active memberships. Source update/disable/delete, config controls and fixed TTL determine lifetime. Stale completion cannot publish; raw input bytes remain immutable encrypted blobs. Previous worker settings are preserved in the new table.
- Reused the tested private Unix and root sandbox boundary. Removed learned priority and persistent laboratory code. Exact core validation remains distinct from isolated-runtime HTTPS proof and Android acceptance. Candidate completion must match deterministic conversion of its exact raw hash.
- Output is a simple responsive list with one JSONL link; failed/pending mappings show diagnosis and are not exported. Ordinary core-valid mappings show connectivity separately; semantic hypotheses need fresh actual connection before export. Legacy bank routes/assets return404.

## Source and compatibility evidence

Target: v2rayNG2.3.6 / Lib40 / Xray26.7.28. Reviewed pinned v2rayNG13138ddd and Xray5ca6f4b source for Custom parser, JSON settings and local DoH bootstrap. Exact upstream commits are in PROJECT_STATE.json/target.lock.json.

The Android batch parser handles URI lines, then a JSON array or single object. JSONL is the user's requested output, not a native Custom subscription format for this version. UI/docs state that distinction and allow copying each JSON. Android import/VPN is not_run. Hysteria1 and unsupported WireGuard extensions remain explicit incompatibilities, never guessed into another protocol.

## Validation

- Full gofmt, module verification, vet, race tests and four binaries passed using verified project-local Go1.27.1 caches.
- 19 complete template fixtures passed the exact pinned Xray binary checksum: VLESS TLS/Reality/WS/XHTTP/gRPC/HTTPUpgrade/HTTP header, VMess, Trojan, SS, Hy2/Hysteria2, SOCKS5, WireGuard URI/INI, JSON, Custom/JSON envelopes and domain bootstrap.
- Nine Python sandbox regressions passed. Template fuzz:240,370 executions, no failure.
- Isolated browser acceptance passed Input, clean Output, Setting, JSONL shape/dedup, old route removal, restart persistence, TTL and390/1440px overflow checks. Browser proof is explicitly synthetic and is not connection evidence.
- Final isolated worker acceptance completed21samples including16 read-only production canaries:19core passes,10two-destination HTTPS passes,1partial. Export19ordinary core-valid mappings. Hysteria1 and one WireGuard extension were diagnosed unsupported. Four synthetic documentation-address endpoints were blocked by network policy. SIGKILL/restart/lease recovery passed, max3parallel. Test ctw resources are separate from production crw resources.
- Production schema5→6 migration preserved54source definitions/revisions and source/conversion settings by fingerprint comparison. Both services remain enabled/active. Installed process executable hashes match the deployed binaries independently of publication.

## Deployment and recovery

Release: /opt/config-repair/releases/p7-31975c1a7562b175b53d.
Panel SHA256:31975c1a7562b175b53dd15a8b475280accfd7321bf5bb3834e053054279ff82.
Worker SHA256:4dd17163b3607c372990df88701e0b660fc16cbfb151a433045999db445e8acb.

Stopped-service matching schema5 DB/blob/key/release backup: /var/backups/config-repair/pre-p7-20261030-105800. Old private legacy lab files were archived there and removed from active project storage. Restore matching old data and binary together for rollback; never start a schema5 binary on schema6 data. Normal unreferenced-blob GC reclaims former bank payloads.

Private receipts: .local/p7-check.log, p7-matrix-test.log, p7-fuzz.log, p7-browser.log, p7-worker-test.json, p7-deployed.json, p7-production-browser.json, p7-published-readback.json and p7-ci.json. Historical counts are timestamped samples, not permanent health guarantees. Source publication/readback and CI are verified separately after the checkpoint is committed; a commit does not contain its own hash.

## Post-deployment SQLite recovery fix

Read-only integrity audit found orphaned conversion results and memberships after database/sql replaced a physical SQLite connection. The previous one-time PRAGMA setup did not cover replacement connections. Foreign keys, busy timeout and FULL synchronous mode now live in the escaped file DSN and apply on every connection. Startup reconciles only transient rows whose parent has already gone, then synchronizes current conversion jobs.

Regression tests force connection recycling and inject legacy orphan rows; cascading deletion, durability settings, source/raw preservation and startup reconciliation pass. Full checks passed again. The first guarded rollout correctly rolled back when remaining legacy membership orphans failed its integrity gate; the complete reconciliation then deployed successfully with zero foreign-key violations. Sources/settings stayed unchanged throughout. The final storage-fix backup is /var/backups/config-repair/pre-p7-db-fix-20261030-111226. Original schema5 migration receipt is .local/p7-schema6-migration.json; .local/p7-deployed.json now describes the final schema6 binary update. Template/worker algorithms are unchanged by this fix.

## Export isolation under concurrent cleanup

An aggregate export can outlive a source/expiry snapshot. A missing/corrupt candidate must not fail unrelated configurations. JSONL now skips unavailable candidate bytes and requeues only the exact still-current result for regeneration; a stale snapshot cannot revoke a newer candidate. Individual downloads report unavailability and trigger the same recovery. Regression deletes one synthetic candidate blob, verifies the other JSON line is exported, then verifies automatic regeneration and stale-result protection. Full checks passed again. Final deployment backup: /var/backups/config-repair/pre-p7-export-fix-20261030-112116. Latest deployment is .local/p7-deployed.json; the storage-fix receipt is preserved in .local/p7-db-reconcile-deployed.json.

## Inline display correction — 2026-10-30

The export previously used `application/x-ndjson`, which caused browser navigation to download it even though API verification passed. The same URL now serves `text/plain; charset=utf-8` with `Content-Disposition: inline`. JSONL bytes, serialization, token, eligibility and conversion behavior are unchanged. The existing export regression checks both headers, and the browser regression actually navigates to the URL, checks rendered text equality and asserts no download event.

Complete check.sh (including pinned-core matrix and sandbox Python regressions) and both synthetic browser phases passed. Public TLS-verified desktop and emulated Android navigation also passed with zero downloads and exact response text. The binary-only release is /opt/config-repair/releases/p7-inline-d5c22a1edbca89f2b361; panel SHA256 d5c22a1edbca89f2b3610dfa28975a403b100167d10bdc1ca36b5d132dcaac8e. All54source definitions and settings, schema6 and the worker binary were preserved. Only the panel service was restarted. Private receipts: `.local/inline-export-check.log`, `.local/inline-export-browser.log`, `.local/inline-export-deployed.json`, `.local/inline-export-live-browser.json`, `.local/inline-export-published.json`. Android app/VPN evidence remains not_run.

Isolated worker integration also passed with9samples (4read-only canaries):8core passes,2two-destination connections and SIGKILL/120-second lease recovery. The initial documentation-address-only run finished before its polling loop could observe and kill a worker; the bounded canary rerun exercised that recovery assertion successfully. No worker logic changed. Receipt: `.local/inline-export-worker.log`.
