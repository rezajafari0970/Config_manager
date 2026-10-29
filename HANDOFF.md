# Handoff — P4a automatic structural candidates

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json and docs/checkpoints/P4a-structural-repair-20261029.md. Verify actual HEAD/status/origin and `.local/p4-published-readback.json` before continuing. The user now authorizes automatic structural repair while original collected bytes remain immutable.

## Current behavior

Independent Go collector retains raw responses and original config spans. A compiled offline CandidateDriver registry produces separate, encrypted structural candidates and reports. Version-pinned JSON rules repair empty object fields, trailing commas and absent empty Custom import containers. Ambiguous/nonempty structures, secrets, protocol identities and unsupported extensions are not guessed. Rules never execute input. Syntax/structural result, core testing, Android import and connection remain different evidence stages.

Input shows repaired/unresolved counts beside original response quality. Output has all/repaired/blocked/unverified filters, per-config diagnostic details and separate repaired download. Raw download/ZIP remain exact. Derived files use exact-original SHA and original TTL/access eligibility; no additional artifact identities are created. Disable/delete/expiry remove repaired download availability. Schema 4 adds repairs and raw-hash lookup index, preserves source/settings/control records and clears conditional validators for fresh diagnosis.

Mixed subscriptions correctly extract contiguous WireGuard sections; distributed headers cannot be safely recombined. Extension fields such as Jc/Jmin/H1 remain original and explicitly unverified for the pinned target. An observed ss URI lacked valid Shadowsocks auth and looked like another protocol; no automatic scheme guessing occurred.

## Verification

Full gofmt/module/vet/race/build checks passed using the verified isolated toolchain. Tests cover surgical byte preservation, unknown fields, large numbers, duplicate keys, nonempty arrays, idempotence, driver panic, mixed extraction, actual automatic repair, report/filter/download access, fixed TTL and orphan collection. Structural fuzz executed 195,337 cases. 24 synthetic pinned-Xray core checks passed; no fetched config was run by Xray. Browser tests cover the mobile/desktop panel, repair filters/details/downloads, original hash/ZIP, restart persistence, source/config bulk controls and expiry. One nil-error dereference found by browser acceptance was fixed and regression-tested before the final deployment.

Isolated real-source audit covered 54 sources and 969 original/config downloads; six Penguin JSON candidates repaired stats, freedom settings and dns.hosts. Samples vary with source content. Production audit and exact installed binary are separate ignored receipts, not inferred from publication.

## Environment

Server `/root/projects/v2ray-config-repair`, GitHub `rezajafari0970/Config_manager`, development branch `go/v2rayng-2.3.6`; legacy main unchanged. Do not touch the unrelated DigitalOcean project, service, proxy or credentials. Publish via GitHub connector and verify via HTTPS git fetch, exact tree, clean worktree and actual executable hash. No force-push.

```sh
export GOROOT="$PWD/.local/toolchain/go" PATH="$PWD/.local/toolchain/go/bin:$PATH"
export GOCACHE="$PWD/.local/go-clean-cache" GOMODCACHE="$PWD/.local/go-mod-verified"
export GOMAXPROCS=2 GOMEMLIMIT=768MiB GOFLAGS=-p=2
bash scripts/check.sh
PLAYWRIGHT_MODULE="$PWD/.local/browser/node_modules/playwright-core" CHROME_PATH=/opt/google/chrome/chrome python3 scripts/panel-browser-test.py
python3 scripts/core-smoke.py --core .local/xray-26.7.28/xray
```

Shared Go/module caches previously failed integrity checks; do not alter them. Passwordless management is explicitly authorized; retain CSRF, Origin, encryption and HTTPS.

Runtime: `config-repair.service`, enabled at boot, restart/watchdog active; `/opt/config-repair/current/config-panel`; loopback18191/publicTLS18443. `/etc/config-repair/panel.json` contains a secret master key; never print it. DB `/var/lib/config-repair/panel.db` schema4, encrypted blobs adjacent. Take private stopped-service SQLite/blob/key backup before migration; restore correct matching schema/binary on rollback.

Ignored receipts: `.local/p4-deployed.json`, `.local/p4-production-audit.json`, `.local/p4-production-browser.json`, `.local/p4-core.json`, `.local/p4-check.log`, `.local/p4-browser.log`, `.local/p4-fuzz.log`, `.local/p4-ci.json`, `.local/p4-published-readback.json`. Temporary real-payload copies/backups are cleaned after acceptance; never assume their blobs still exist. Source settings and user data are preserved; destructive tests use synthetic fixtures.

## Limits / next scope

No Android APK/device acceptance or live VPN connection test. Candidate structure does not establish usable service. No arbitrary input core execution until isolation/egress controls are implemented. Unknown protocol/transport coverage remains open; dynamically uploaded plugins are not implemented. Final reproducible installer/update/rollback is P8. Continue adding evidence-backed repair rules only when actual symptoms justify them; retain original bytes and conservative rejection.
