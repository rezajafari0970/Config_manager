# Handoff — P9 independent source exports

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/CONVERSION.md and the P9 checkpoint (P7/P8 are historical context). Verify actual HEAD/origin/status and private p7 receipts. The user's latest instruction explicitly removes the P6 persistent bank; do not resurrect it based on older checkpoints.

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
