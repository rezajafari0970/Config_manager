# Handoff — P7 JSON template conversion

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/CONVERSION.md and the P7 checkpoint. Verify actual HEAD/origin/status and private p7 receipts. The user's latest instruction explicitly removes the P6 persistent bank; do not resurrect it based on older checkpoints.

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
