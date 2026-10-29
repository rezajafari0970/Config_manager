# Handoff — P3 immutable collection

Read AGENTS.md and docs/PROJECT_BRAIN.md first. The latest explicit user request removes authentication and prohibits changing any collected config byte. Keep raw collection separate from the existing opt-in CLI repair and future healthy exports.

## Verified implementation

Input / Output / Setting use separate backend modules, templates and JavaScript modules. Bulk line-based import, automatic parent-path names, seconds/minutes interval expression, edit/delete/pause/run, live outcomes and mobile drawer are active. Independent bounded fetch jobs have separate clients/contexts, persisted leases, revision rejection, atomic generations and panic recovery. Encrypted streaming blobs and staged metadata avoid collecting every config in RAM. Original source response and extracted spans are retained byte-for-byte. Conservative semantic identities deduplicate known formats while memberships preserve each source's original representation. Settings has fixed 2-hour retention and independent request timeout/concurrency. Expired data is excluded immediately and garbage-collected every 5 seconds. Repeat fetch/304 cannot renew lifetime; a later fresh 200 may re-collect it.

## Tests and environment

Module verify, format, vet, race tests and reproducible build pass with official isolated Go 1.27.1. Chrome 154/Playwright core 1.56.1 passed desktop/mobile, public passwordless TLS pages, bulk/CRUD, independent errors, SHA/raw/ZIP preservation, fixed expiry and full process restart. Slow/panicking source isolation also passed ten race-enabled repetitions after replacing a timing-dependent test assertion with an explicit test-driver barrier.

The host's shared Go race object and shared modernc.org/libc directory failed integrity/build checks. Keep using the isolated project caches; no global toolchain/module directory was repaired or changed.

```sh
export GOROOT="$PWD/.local/toolchain/go" PATH="$PWD/.local/toolchain/go/bin:$PATH"
export GOCACHE="$PWD/.local/go-clean-cache" GOMODCACHE="$PWD/.local/go-mod-verified"
export GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=2
bash scripts/check.sh
PLAYWRIGHT_MODULE="$PWD/.local/browser/node_modules/playwright-core" CHROME_PATH=/opt/google/chrome/chrome python3 scripts/panel-browser-test.py
```

## Runtime and publication

Server checkout: `/root/projects/v2ray-config-repair`; development branch `go/v2rayng-2.3.6` in `rezajafari0970/Config_manager`. Legacy main remains unchanged. GitHub connector writes, HTTPS origin fetches/readbacks; default server SSH does not have repository push authorization. Do not use unrelated credentials.

Independent service `config-repair.service`, binary `/opt/config-repair/current/config-panel`, loopback `127.0.0.1:18191`, HTTPS port `18443`. Runtime config `/etc/config-repair/panel.json`, DB `/var/lib/config-repair/panel.db`, encrypted blob directory adjacent to DB. Schema 2 migration drops old admin/session/attempt tables while retaining URLs/schedules. The existing production database contained zero sources before and after upgrade; no example URL was inserted into it. systemd restarts exits and uses a 45-second watchdog with scheduler/DB-gated notifications. Nginx's old body cap was removed only for this site. Other nginx site files were verified unchanged.

Protected SQLite-backup-API snapshot and config/unit/site backups precede migration under `.local/p3-upgrade-*`. P2 rollback requires its schema-1 DB backup plus old binary. Private receipts: `.local/p3-deployed.json`, `.local/p3-production-browser.json`, `.local/p3-recovery.json`, `.local/p3-browser/75d0ffb4deb8/receipt.json`; publication readback/CI receipts are separate and created after source commit. Verify actual git HEAD/worktree, runtime executable SHA and these receipts before future work.

## Limits and next work

Raw Output does not mean valid import or working VPN. Prior 13 Xray synthetic checks remain historical evidence only. Android import/runtime and real proxy connection are still not_run; no adb/emulator/KVM/authorized device was available at preflight. No repair or automatic JSON conversion runs in the collector. Compiled driver/module boundaries exist; dynamic plugin upload/marketplace does not. Physical resources, deadlines, safe destination policy and bounded parallelism still apply despite removal of business quotas. Do not claim absolute fault isolation across shared host/DB/disk or universal semantic deduplication.

Next: expanded fixture/performance coverage, exact-client acceptance and separately approved validated export, then P8 reproducible install/update/rollback. Do not re-enable repairs in raw Output or restore login without a new user request.
