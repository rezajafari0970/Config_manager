# P13 — Shared URI compatibility and source diagnostics

## Problem and changes

The Hot screenshot combined one malformed original with multiple downstream conversion failures under a generic warning. Inspection of all current sources found14Hot inputs missing the REALITY key,3URIs with unmapped TLS cipher/packet options across Hot/Afrak, and1URI whose short VLESS ID contains unescaped spaces.

The shared converter now preserves TLS cs as cipherSuites, recognizes explicit VLESS XUDP without enabling TCP multiplexing, and safely percent-encodes only literal userinfo spaces. Original credential bytes and all acquired blobs remain unchanged. All4previously blocked repairable cases pass pinned-core validation; their fresh private HTTPS checks failed, so no successful connection is claimed. The14missing-key cases remain rejected with a precise actionable explanation. No key, endpoint or transport is invented.

REALITY preflight covers URI/custom JSON, retained chains/pools and nested streams, including the core's password alias. Input and Output share localized diagnostic text; source-level issue counts follow canonical ownership ranking and separate pending work from final failures. The new asset is explicitly served; browser testing caught and fixed its initially missing allowlist registration.

Pinned source references at5ca6f4b7d4dc20a881d4330e498892697627ec0c: infra/conf/transport_security.go (cipherSuites, fingerprint, REALITY publicKey/password), transport/internet/tls/config.go (colon-separated suites), common/uuid/uuid.go (short ID behavior), app/proxyman/outbound/handler.go (independent TCP/XUDP managers), proxy/vless/outbound/outbound.go (UDP wire encoding). v2rayNG DTO retains cipherSuites; actual Android import/TUN is not_run.

## Verification

- Full formatting/module/vet/race/build checks and11Python sandbox tests passed.
-22protocol template fixtures and4balancer strategies passed the pinned core.
- Real isolated TCP/UDP/UDP443 tests passed Vision, Mux and explicit-XUDP before/after controls.
- Synthetic browser passed Input/Output/Setting, inline JSONL, restart, expiry, mobile and desktop. Public TLS browser passed exact missing-key message, no overflow/JS errors and no forced downloads.
- Isolated worker: 9items, 8core passes, 3two-HTTPS passes; SIGKILL/lease recovery passed.
- Private captured cases: 19examined,4newcorepasses,0newtwoHTTPSsuccesses,14missingkeys. Captures are transient under .local and excluded from Git.
- Production audit 2026-10-30T20:19:37.334136+00:00: 321unique inline JSONL objects, zero old job versions, schema/foreign keys healthy. Source summaries are recorded in PROJECT_STATE.json; counts are snapshot evidence and change with upstream feeds.
- Android app/TUN and actual QUIC/HTTP3 handshake remain not_run. A source without its authentication key or with a nonresponsive server cannot be made connected by changing JSON alone.

## Deployment and continuity

Template v4, structural v3, probe-plan v3; schema8 unchanged. All8source definitions/revisions and settings matched across deployment.

Release: /opt/config-repair/releases/p13-2115fff62b6b32d45137
Panel SHA256: 2115fff62b6b32d45137fd19c950d20ba8d676326a10bc0caf3751ab428d14a4
Worker SHA256: 5628c5475bf59b30951ae54552ff06c86fb3fe9a0d573e9845b39ae106b47d9d
Unchanged sandbox driver SHA256: c979512fb5ecd21e2cf97b0d73169aa338684b1cbdd40ab36a325b7b149199de
Stopped-service matching backup: /var/backups/config-repair/pre-hot-20261030-201502

Both services are enabled/active; installed process hashes were verified separately. Same-schema rollback switches the release and retains user data. Private receipts: hot-{check,browser,worker,canaries,audit}.log, hot-deployed.json, hot-live.json, hot-public-browser.json, hot-published.json. Publish only the explicit source/test/documentation allowlist; no originals, URLs, DB, keys or raw logs.
