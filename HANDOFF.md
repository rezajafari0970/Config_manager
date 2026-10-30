# Handoff — P12 open TCP/UDP/QUIC traffic

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/CONVERSION.md and the P12 checkpoint (P7/P8/P9/P10/P11 are historical context). Verify actual HEAD/origin/status and private p7 receipts. The user's latest instruction explicitly removes the P6 persistent bank; do not resurrect it based on older checkpoints.

Authoritative server tree: /root/projects/v2ray-config-repair. GitHub: rezajafari0970/Config_manager, branch go/v2rayng-2.3.6. Legacy main and unrelated DigitalOcean infrastructure are outside scope.

Schema6 drops all bank/probe persistent history/model tables and uses transient conversion jobs/results/settings/worker, tied to active source membership and fixed TTL. Output lists conversion/core/connection status, offers JSON copy, and one JSONL export link. There is no bank/manual import/variant history/old subscription endpoint. Source originals remain encrypted and immutable. Core validation is required; ordinary mappings do not claim connectivity. Semantic hypotheses require fresh two-destination HTTPS success.

Pinned Android source expects a JSON array/single object for Custom subscription, not multiple newline JSON objects. The user explicitly requested JSONL. UI/docs state the distinction; Android testing remains not_run. Hysteria1 cannot be translated to a supported wire protocol merely by renaming.

The existing config-repair-prober.service now validates conversions using convert.Version, private Unix queue and the unchanged owned crw sandbox boundary. scripts/conversion-worker-test.py uses separate ctw resources and lock, allowing isolated integration alongside production. No production sources/settings may be changed for tests. Source collection and validation remain separate services.

Use the verified isolated Go environment; shared caches previously failed integrity checks and must not be altered:

```sh
export GOROOT="$PWD/.local/toolchain/go" PATH="$PWD/.local/toolchain/go/bin:$PATH"
export GOCACHE="$PWD/.local/go-clean-cache" GOMODCACHE="$PWD/.local/go-mod-verified"
export GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=2
XRAY_TEST_BINARY="$PWD/.local/xray-26.7.28/xray" bash scripts/check.sh
PLAYWRIGHT_MODULE="$PWD/.local/browser/node_modules/playwright-core" CHROME_PATH=/opt/google/chrome/chrome python3 scripts/panel-browser-test.py
python3 scripts/conversion-worker-test.py --live-canary 16
```

Runtime: config-repair.service and config-repair-prober.service, loopback18191/publicTLS18443. Private /etc/config-repair/panel.json contains the master key; never print it. DB /var/lib/config-repair/panel.db, blobs adjacent. A schema5 binary cannot open schema6; rollback requires the matching stopped-service backup listed in .local/p7-deployed.json. Source revision, binary/tree, deployment and CI receipts must be read separately. Final reproducible installer remains a future release task.

Post-deployment storage correction: SQLite connection pragmas are DSN-bound so replacement connections retain foreign-key cascading and durability. Startup reconciles only legacy orphan transient rows. Tests force physical connection churn; production foreign_key_check is empty. The initial schema5 migration is recorded separately in .local/p7-schema6-migration.json, and .local/p7-deployed.json is the final schema6 storage-fix deployment. Read both when selecting a matching rollback.

## Inline export correction — 2026-10-30

The existing JSONL URL now responds with `text/plain; charset=utf-8` and `Content-Disposition: inline`, so Chrome displays the exact line-delimited JSON instead of downloading an unrecognized NDJSON media type. URL, token and serialization are unchanged. Existing browser coverage now navigates to the export and checks visible response bytes and zero download events. Public HTTPS navigation passed on desktop and emulated Android. Latest binary-only deployment and publication receipts are `.local/inline-export-deployed.json` and `.local/inline-export-published.json`; schema6, all54sources and settings were preserved. Rollback for this header-only update is the previous schema6 release recorded in the deployment receipt; no database restoration is required.

## Current P8 single-config rotation — 2026-10-30

Schema7 adds export_rotation_state and export_rotation_queue. `internal/panel/rotation.go` is a separate PanelModule and stores only the current round number and pending candidate hashes. `/api/conversions` now returns `single_export_url`; Output displays both links. One GET consumes one exact compact JSON, inline text, with no cache. The global queue survives restart and wraps only after its current snapshot is exhausted. New configs join the next round. Deleted/disabled/expired/unaccepted entries are skipped; missing blobs are requeued for conversion. Empty output is204; HEAD/prefetch never consumes. Blob I/O remains outside DB transactions; a write-locked final claim checks current eligibility, cycle and queue head. Delivery after a client disconnect cannot be guaranteed without client acknowledgements.

Current release and hashes are in PROJECT_STATE.json. Latest receipts: `.local/rotation-deployed.json`, `.local/rotation-live-browser.json`, `.local/rotation-published.json`. Full checks, synthetic browser/restart, two-Store parallel rounds with GC, isolated worker SIGKILL recovery and live public inline/parallel checks passed. All54sources/settings were preserved. The matching stopped-service backup is /var/backups/config-repair/pre-rotation-20261030-121235. Schema6 binaries reject schema7; rollback must restore matching DB/blobs/key/release while both services are stopped. Do not use the previous header-only rollback instructions for this migration.

## Current P9 source-specific output — 2026-10-30

Every source has independent full JSONL and single-config links, shown under its Input row and in the paginated/searchable Output source list. `/output?source={id}` scopes the list and individual JSON copies; `/api/conversions?source={id}` returns scoped links/entries. `/api/export/sources` lists link metadata without input URLs. The source membership predicate is applied before canonical ranking so no representation/proof is borrowed from another source. Shared configs appear in every owner source and only once globally.

Schema8 migrates the existing global queue/cycle unchanged and adds source-bound rotation scopes. The source foreign key cascades its queue on deletion. Per-scope cancellable locks isolate blob waits; SQLite guards remain authoritative across processes. Mode/source-bound tokens prevent scope substitution. Retention, generation replacement, disable/delete, validation gates and raw bytes are unchanged. Read docs/CONVERSION.md and docs/checkpoints/P9-source-exports-20261030.md.

Latest receipts: `.local/source-export-deployed.json`, `.local/source-export-live.json`, `.local/source-export-published.json`. Current release/hashes are in PROJECT_STATE.json. There were 7 source definitions at deployment; all were preserved, along with settings and global cycle 78. Earlier54-source counts describe older snapshots and must not be restored. The matching stopped-service DB/blob/key/release backup is /var/backups/config-repair/pre-source-export-20261030-125245. Old schema7 binaries reject schema8; rollback requires the matching backup while both services are stopped. Full checks, browser/source isolation/restart, isolated sandbox worker recovery and live scoped exports passed. Android app/VPN remains not_run.

## Current P10 TXT/mixed extraction — 2026-10-30

The TXT failure was a case-sensitive path mismatch (capitalized filename versus the user's lowercase filename), not a TXT/MIME restriction. The existing source was edited via the CSRF/revision-checked API with all timing/settings retained. Do not auto-normalize path case or restore the wrong URL.

`internal/collect/envelopes.go` now extracts embedded Base64 blocks within JSON/URI text, including independent encoded lines and wrapped streams. Strict decoding retains padding checks; malformed envelopes cannot silently corrupt payloads or discard good siblings. Eight encoding layers bound recursion, not item count/size. The registry is reused after decoding. Original full JSON dependencies remain intact; no child-outbound splitting.

Full checks/core matrix, synthetic browser/restart and an isolated worker rerun passed. The first worker test failed startup cleanup; `.local/txt-mixed-worker.log` records it, while `.local/txt-mixed-worker-retry.log` records the successful diagnostic rerun. No worker/sandbox production code changed. Initial TXT snapshot had95unique items; a newer response had94unique from101received, all94passed core/twoHTTPS and exported. Live mixed response had7items (6JSON,1Base64); all7originals matched response/decoded hashes. Their conversion remains blocked (5ambiguous selection,1structural unresolved,1unknown query parameter). Do not describe extraction as repaired connectivity.

Binary-only schema8 deployment: `.local/txt-mixed-deployed.json`, release/hashes in PROJECT_STATE.json. All7source definitions/revisions and settings matched before/after deployment. Backup: /var/backups/config-repair/pre-txt-mixed-20261030-132831. Rollback can switch to the previous schema8 binary while services are stopped; do not replace current data/user edits for this same-schema change. Other receipts: `.local/txt-mixed-check.log`, `.local/txt-mixed-browser.log`, `.local/txt-mixed-live.json`, `.local/txt-mixed-final-output.json`, `.local/txt-mixed-byte-check.json`, `.local/txt-mixed-published.json`. Current source counts/proofs can change on each upstream refresh. No raw captures or source URLs belong in Git.

Final boundary regression also proves a wrapped JSON child object is never promoted to an independent config while its parent is incomplete. Targeted before/after receipts are .local/txt-mixed-boundary-before.log and .local/txt-mixed-boundary-after.log. The final full checks/browser run passed again; worker binary remains unchanged.


## Current P11 Exir conversion repairs — 2026-10-30

Template v2 and structural v2 are deployed. Read docs/checkpoints/P11-exir-repair-20261030.md and the P11 section of docs/CONVERSION.md. Empty DNS/blackhole settings arrays, explicit balanced multi-proxy routes and the conflict-checked XHTTP x_padding_bytes alias are now covered. Balancer selectors use prefix semantics; preserve original pool/chain/observer tags and first real default proxy. Do not collapse a pool into one arbitrary server or let helper tags enter it.

Input now counts canonical source-owned conversion evidence, instead of summing historical raw repair records. Original response diagnostics remain separate. Template-version changes invalidate stale export proofs and requeue active originals. Mixed public IPv4/IPv6 pools can be tested through IPv4 while retaining all original members; the inet sandbox default-drop boundary is unchanged. IPv6-only network testing remains pending.

Twenty current/historical private Exir representations passed core after conversion; network outcomes varied. Fresh production snapshot at18:52:27Z had7ready/7exported,6two-HTTPS passes,1IPv6-only pending and1structurally repaired original. All7original hashes matched. Full race/build/pinned-core tests,11Python tests, synthetic browser/restart, isolated worker SIGKILL recovery and public desktop/emulated Android inline checks passed. This is not an Android app/TUN receipt.

Latest deployment/publication receipts: .local/exir-repair-deployed.json and .local/exir-repair-published.json. Release and three hashes are in PROJECT_STATE.json. The updated live-probe.py is part of the release alongside panel and worker; rollback all three together. Schema8 and all7source definitions/revisions/settings were preserved. Backup:/var/backups/config-repair/pre-exir-repair-20261030-185005. Same-schema rollback preserves current user data and changes. Do not restore old source counts or infer current outcomes from these snapshots. No raw fixtures or private captures belong in Git.


## Current P12 open TCP/UDP/QUIC traffic — 2026-10-30

User explicitly wants TCP, UDP and QUIC allowed in generated configs. Template v3 preserves the existing SOCKS UDP/sniffing/tcp,udp routes and opens the pinned core's local Vision/Mux UDP443 gates. Vision receives its supported -udp443 suffix, stripped on wire by Xray. Existing Mux absent/empty/reject UDP443 policy becomes allow; explicit skip is retained. Null/absent mux stays null/absent; multiplexing and concurrency are not changed. Originals, secrets, TLS/Reality and endpoints remain unchanged.

Read docs/checkpoints/P12-open-traffic-20261030.md and the P12 section of docs/CONVERSION.md. The optional root-only packet regression uses the existing sandbox with synthetic pinned-certificate VLESS peers and fresh SOCKS UDP associations: before gates, TCP/ordinary UDP pass and UDP443 fails; after policy all pass. It does not claim upstream QUIC/HTTP3 negotiation or Android app validation. The private localhost echo fixture uses an explicit narrow freedom finalRules allow for its loopback endpoints, because the pinned server core blocks private destinations by default. Production sandbox boundaries are unchanged.

Full checks/pinned core, browser/restart and isolated worker recovery passed. Latest receipts are .local/traffic-deployed.json, traffic-live.json, traffic-published.json and traffic-{check,browser,worker,targeted}.log. Schema8 and actual live source definitions/revisions/settings were preserved. Runtime hashes/release are in PROJECT_STATE.json. Same-schema rollback switches to the prior release while retaining current user data; use the matching stopped-service backup recorded in the deployment receipt.
