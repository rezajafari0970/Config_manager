# Handoff — P3b acquisition and bulk management

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json and the current checkpoint first. Verify the actual git HEAD, working tree, runtime executable hash and publication receipt. The user explicitly restricts current work to collection/settings; immutable raw data and passwordless management remain required.

## Current state

Separate Input / Output / Setting route/frontend modules run on the existing independent Go service. Source and config checkboxes support bulk enable/disable/delete; explicit clear-all buttons are available, with confirmation. Source actions are revision-checked transactions and cancel only affected jobs. Config disable excludes ordinary downloads/ZIP but permits admin preview; expiry-bound tombstones prevent in-flight/repeated resurrection after deletion. Clear-all-configs revokes job leases/revisions, clears stage/response/artifact data and preserves URLs/settings. User production data was not cleared during acceptance.

Per-source integer timeout seconds override the shared Setting value; empty/zero inherits. Hidden 10/15-second HTTP transport cutoffs were removed. Android interval input now uses a text keyboard. A corrupt source record no longer blocks all due records. Partial malformed content no longer discards recognized siblings. InspectionDriver plugins label raw items and contain inspection panic; no repairs or arbitrary core execution occur. Enabled state, request outcome and content quality are distinct. Current effective parallel capacity is visible.

## Evidence

Go module verification, format, vet, race tests and build pass. Tests cover source CAS and atomic rollback, stale completion, source timeout isolation, header waits exceeding the former 15-second cap, config disable/delete/clear under concurrent collection, fixed control TTL, malformed siblings, JSON string envelopes, byte preservation and inspection-plugin failure.

Chrome 154 / Playwright core 1.56.1 passed 1440/390px layouts, drawer, bulk source/config actions, destructive confirmations, actual default/per-source timeout fields, expiry and full restart persistence including disabled configs. Current fixture receipt: `.local/p3-browser/6d3655bf94f0/receipt.json`. Screenshots inspected.

An isolated copy of the actual 54 sources completed a fresh sweep in 12.91 seconds with 16 concurrent jobs: 44 readable, 1 structurally invalid, 6 empty, 3 unrecognized in that sample. All 950 original/config downloads matched their stored SHA256; zero identical raw payloads had multiple identities. Sampled Penguin JSON contained stats arrays and a freedom settings array; raw files were retained unchanged and only annotated. Outcomes vary per fetch and do not establish VPN connectivity. The final installed version has its separate production audit receipt.

## Environment / build

Server `/root/projects/v2ray-config-repair`; GitHub `rezajafari0970/Config_manager`, branch `go/v2rayng-2.3.6`; legacy main is unchanged. GitHub connector publishes; HTTPS fetch verifies. Do not use unrelated project credentials. Shared Go race object and shared libc module cache previously failed integrity checks. Use isolated verified caches:

```sh
export GOROOT="$PWD/.local/toolchain/go" PATH="$PWD/.local/toolchain/go/bin:$PATH"
export GOCACHE="$PWD/.local/go-clean-cache" GOMODCACHE="$PWD/.local/go-mod-verified"
export GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=2
bash scripts/check.sh
PLAYWRIGHT_MODULE="$PWD/.local/browser/node_modules/playwright-core" CHROME_PATH=/opt/google/chrome/chrome python3 scripts/panel-browser-test.py
```

## Runtime / operational receipts

`config-repair.service` is enabled at boot. Binary `/opt/config-repair/current/config-panel`, loopback 127.0.0.1:18191, TLS port 18443. Runtime `/etc/config-repair/panel.json`, schema-3 SQLite `/var/lib/config-repair/panel.db`, adjacent encrypted blob directory. Passwordless public administration is explicitly authorized; keep HTTPS/CSRF. Settings and all 54 sources were preserved at migration. No nginx site or unrelated DigitalOcean service changed.

Before upgrade, stop this service, snapshot via SQLite backup API and preserve the master key and blob files privately. Rollback from schema 3 requires the schema-2 backup plus old binary. Runtime candidate SHA256: `c295a35a3a01516257ab67e797b08e0a4cd7fae15028ec24cf6b01f9485395d6`. Do not infer deployment from Git publication.

Ignored private receipts: `.local/p3b-deployed.json`, `.local/p3b-production-audit.json`, `.local/p3b-recovery.json`, `.local/p3b-published-readback.json`, `.local/p3b-ci.json`. Exact publication/CI receipts are written after the source commit. Temporary real-source blob/raw audit copies are removed after verification; keep only sanitized receipts. Rebooting the shared host was not part of acceptance. A separate SIGSTOP watchdog test checks recovery; physical power-off cannot sustain fetching, but boot enablement resumes the daemon.

## Outstanding scope

No real proxy connection or Android import/runtime acceptance. Existing CLI repair stays opt-in and separate. Dynamic uploaded plugins are not supported; extension interfaces are compiled. Known-format identities are conservative; no universal semantic-dedup proof or bug-free/absolute-uptime guarantee. Final installer remains P8.
