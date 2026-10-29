# Handoff — P6a persistent probe bank

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/PROBE_BANK.md and docs/checkpoints/P6a-permanent-probe-bank-20261029.md. Verify actual HEAD/origin/status and private p6 deployment/publication receipts. Older handoff below is historical; current user authorization supersedes transient sample lifetime and permits explicitly labelled server-verified subscriptions before Android testing.

Production panel now uses schema5. Original samples and derived trial history survive source replacement/expiry/deletion; explicit bank delete leaves a tombstone. Every accepted collector generation/manual import queues new samples. A protected Unix socket connects the unprivileged panel to a separate root supervisor; only the isolated core executes untrusted configurations. Both systemd units are enabled. Public panel has Input / Output / Setting with bank management, reports, manual import, probe settings and two tokenized Custom JSON subscriptions. All 54 source URLs/revisions and original fetch settings were preserved.

Core/connection evidence is versioned and time-limited. Android remains not_run. Full success requires two HTTPS destinations. Retest failure, explicit disable/delete/retest and evidence expiry exclude entries from the subscription. Historical candidates remain encrypted. Learned outcomes prioritize rules but never approve another sample. Hysteria1 is explicitly unsupported by pinned core; unknown extensions remain retained/diagnosed.

Full Go/module/vet/race/build and 11 Python regressions passed. Browser acceptance covers desktop/mobile, old raw management and new bank actions/import/settings/reports/subscription gates. Isolated worker test completed 17 samples including 12 real: 3 full passes, 7 partial, with SIGKILL and lease recovery. Production continuously processes a larger evolving bank; consult current API rather than treating checkpoint counts as current health.

Private receipts: .local/p6-check.log, p6-browser.log, p6-worker-test.json, p6-worker-diagnostic.log, p6-deployed.json, p6-live.json, p6-production-browser.json, p6-published-readback.json, p6-ci.json. Pre-migration stopped-service data/blob/key/binary backup is under /var/backups/config-repair; exact path is in the deployment receipt. A P4 binary cannot open schema5: rollback needs its matching backup. Later schema5 binary updates do not roll back data.

Do NOT run scripts/probe-worker-test.py while production prober is running: its cleanup owns all crw resources. To repeat isolation tests, stop only config-repair-prober.service, run the isolated test, then restore the service. Do not stop the collector or touch unrelated DigitalOcean infrastructure. Use verified isolated Go toolchain/cache from the prior handoff below.

---

# Handoff — P5a temporary probe laboratory

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/probe-lab.md and docs/checkpoints/P5a-temporary-probe-lab-20261029.md. Verify HEAD/origin/status and .local/p5-published-readback.json before continuing. The user accepts server-side tests from any network; actual Android acceptance remains pending.

Go preparation CLI and temporary Linux sandbox driver are implemented. The complete snapshot run considered 868 configs: 93 reached both HTTPS destinations, 196 reached one; 192 expired/disappeared and 10 changed before testing. 870 attempts include alternatives. No TLS hypothesis passed the two-destination approval gate. A final-code canary considered 18 originals: one two-destination pass and two partial passes. These are timestamped Linux results, not healthy export approvals.

The evidence bank is .local/probe-bank/evidence.db and stores no raw configs/logs/passwords. Driver/CA/core/planner hashes are recorded per run. Initial CA-less calibration runs were invalidated and excluded from aggregation. Source originals, production sources/settings, collector and panel binary were not replaced. Panel DB stays schema4; installed production release remains the P4a release.

Use the verified isolated Go toolchain below. `scripts/check.sh` now also builds config-probe-plan and runs unprivileged Python regressions. Run `python3 scripts/live-probe.py --self-test` before any live batch; a failed positive egress control is an infrastructure blocker, not a config failure. Never expose the privileged CLI through the passwordless panel. Temporary resources clean up on ordinary completion/SIGINT/SIGTERM; crash/SIGKILL cleanup and durable job leases are required before always-on integration.

Private receipts: .local/p5-check.log, p5-python-tests.log, p5-fuzz.log, p5-final-sandbox.log, p5-live-full-ca.log, p5-final-canary.log, p5-published-readback.json, p5-ci.json. See checkpoint for scope and limitations. No plaintext real-config copies were retained.

## Prior production handoff (still applicable)

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
