# Working contract — P7 JSON template conversion

Read docs/PROJECT_BRAIN.md, PROJECT_STATE.json, HANDOFF.md and the current checkpoint. Verify actual HEAD, origin, worktree and runtime receipts. The user's 2026-10-30 instruction supersedes the persistent P6 laboratory: remove the bank, collect from links, convert to the shared @shenvpn JSON envelope, and emit one compact JSON object per line.

## Scope
- Independent Go project, server deployment only. Never modify unrelated DigitalOcean infrastructure.
- Target remains V2RayNG 2.3.6 / Lib v40 / Xray 26.7.28. Read pinned source before version-sensitive mapping; preserve reference commits.
- Keep legacy main intact. Work on go/v2rayng-2.3.6. No force-push.
- Passwordless management is explicitly authorized; retain HTTPS, Origin/CSRF and encryption.

## Conversion and evidence
- Original acquired bytes remain immutable encrypted blobs. Derived JSON uses the requested common log/DNS/inbound/routing/stats/policy, actual protocol credentials/transport, and remarks @shenvpn.
- Never copy the example endpoint/auth/HTTP transport to unrelated configs, invent secrets, change wire protocols, or weaken TLS.
- Preserve necessary outbound chain dependencies. Ambiguous selection, missing dependencies and unsupported protocols must remain explicit failures.
- Conversions and their jobs/results are transient, owned by current enabled memberships and fixed artifact TTL. No permanent sample/history/model/rule bank or manual bank import.
- JSONL is one compact object per line, no array or text. Pinned-core acceptance is mandatory for export. Semantic repair hypotheses additionally require fresh two-destination HTTPS evidence. Ordinary core-valid output may lack connectivity; UI must show that accurately.
- Android import/VPN remains not_run without an actual APK/device receipt. Pinned v2rayNG's Custom JSON subscription parser accepts an array or a single object, not JSONL; do not call JSONL a native compatible subscription. Individual JSON can be copied for import.
- No arbitrary config executes outside the proven core sandbox. The root supervisor consumes only the protected Unix queue. Never expose private claim/renew/complete/status on TCP.
- No raw configs, URLs, secrets, logs or production DB in Git. Tests use synthetic credentials/documentation addresses, except explicit isolated read-only canaries with private transient files.

## Lifecycle
- Each source has independent contexts, randomized schedules, timeout and revision/lease checks. Bounded parallel work, deadlines, retry/backoff, crash recovery and startup supervision remain required.
- Accepted nonempty generation replaces only its source's membership; empty/error retains prior data until fixed TTL. Repeated fetch/304 does not renew TTL. Identity dedup never rewrites raw bytes.
- Source/config deletion, disable, edit and TTL invalidate conversion claims; stale workers cannot resurrect output. No DB calls through Store while its sole SQLite connection is held in a transaction/rows.
- Preserve sources/settings on deployment. The explicit removal of the bank is authorized. Take matching stopped-service DB/blob/key/binary backup before schema migration. Do not infer deployment from a commit.

## Checks and release
- Run scripts/check.sh, pinned-core template matrix, relevant browser and sandbox worker integration. Test idempotence, ambiguity, exact bytes, NDJSON, dedup, migration, source lifecycle and crash recovery.
- scripts/conversion-worker-test.py uses distinct ctw resources and an isolated lock; production owns crw resources. Never use a test cleanup against production resources.
- Update state, handoff and checkpoint together. Publish tested source; fetch/read back the exact revision and verify installed binary hashes separately.
- Final automated installer is still a separate release task requiring reproducible checksum-pinned install/update/rollback tests.

- SQLite foreign_keys, busy_timeout and synchronous pragmas must be applied to every physical connection via the DSN; one-time Open PRAGMAs do not survive connection replacement. Retain reconnect/cascade and startup reconciliation regressions.

## P8 single-output rotation
- Read docs/checkpoints/P8-single-output-20261030.md for the current additive schema7 change. Preserve the all-config export and independent source lifecycle.
- The single link uses one global durable round across clients; snapshot only unique eligible candidate hashes. New arrivals join the next round. HEAD/prefetch must not consume turns.
- Validate blobs outside SQLite transactions to preserve the vault-to-DB garbage-collection lock order; recheck eligibility and queue head atomically before consuming. Do not claim exactly-once network delivery after an interrupted response.

## P9 source-specific output
- Read docs/checkpoints/P9-source-exports-20261030.md. Schema8 preserves the global queue and adds independent source scopes; never reset existing global turns during migration.
- Filter source membership before window ranking/deduplication. Never substitute another source's raw representation or proof for a scoped result.
- Source deletion cascades only its scope. Locks are per scope, cancellable and reclaimed; vault I/O remains outside transactions. Preserve both global URLs.
