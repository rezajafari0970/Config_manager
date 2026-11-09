# Handoff — current P21 RealPing and random delivery

Read AGENTS.md, docs/PROJECT_BRAIN.md, PROJECT_STATE.json, docs/CONVERSION.md and docs/checkpoints/P21-realping-random-order-20261107.md. Verify actual HEAD/origin/status and private p21 receipts. Earlier checkpoint sections below are historical. The latest request applies tunneled DoH, independent IP bootstrap, burst15/3/5, keepalive45/15 and local metrics to all existing Smart outputs; preserve raw/non-Smart data and the bank-free lifecycle.

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

## Current P13 shared URI compatibility — 2026-10-30

Read docs/checkpoints/P13-shared-repair-20261030.md, PROJECT_STATE.json and docs/CONVERSION.md P13 before continuing. Template v4/structural v3/probe-plan v3 fix shared userinfo-space and TLS cipher/XUDP mapping. Explicit XUDP uses a single UDP stream with TCP mux disabled; no unsolicited Mux. REALITY missing keys remain actionable failures; no security downgrade or fabricated credentials. Source diagnostics use the same canonical ownership choice as Output.

Full checks,22pinned protocol fixtures,4balancer fixtures,3real packet controls, browser and isolated SIGKILL recovery passed. Four previously blocked private samples became core-valid, but their fresh HTTPS probes did not connect.14Hot inputs lacked REALITY keys at inspection. Actual live counts can change; consult the current API rather than restoring historical sources/revisions. Android and QUIC handshake remain untested.

Deployment preserved schema8 and all current definitions/settings, used a stopped-service matching backup, and verified both running binary hashes. Current receipts are .local/hot-deployed.json, hot-live.json, hot-public-browser.json and hot-published.json; runtime hashes are in PROJECT_STATE.json. Initial browser failure was a missing static-asset allowlist entry, now covered and fixed. No source sample bank was restored.


## P14 Smart Output — 2026-11-01

Read docs/checkpoints/P14-smart-output-20261101.md and current PROJECT_STATE.json before further work. Additive schema9, Smart v1, normal template v4 unchanged. Current release /opt/config-repair/releases/p14-3ab4b9d7c8d076f24092; all three services active/enabled. Independent Smart full/single URLs are returned by /api/smart, with count10 configurable in Setting.

Coverage, dedup, full first-position cycles, restart, member lifecycle, current pool growth and source-change lease revocation are tested. Exact-core50-member worker/SIGKILL integration and observed synthetic TCP/UDP/UDP443 failover passed. Public browser checks passed; production counts remain transient. Android app/TUN and live QUIC handshake are not_run. Originals and all normal/source settings/rotation scopes were preserved. Smart JSON viewer uses the current row after an asynchronous refresh.

Publication must read back the exact branch commit and compare running binary hashes separately. Parent before this phase:15cb39b22fc5b168b86fe7423496767a7a37b89f. The final receipt is .local/smart-published.json (not committed). No subagents, unrelated infrastructure changes or example endpoint imports were used.

## P15 exact ordered Smart envelope — 2026-11-01

User explicitly wants all outbounds copied, every other field matching the supplied sample and logging completely disabled. Read docs/checkpoints/P15-smart-envelope-20261101.md. Smart v2 replaces v1 global merging: fixed root/nested order and values, loglevel/access/error none plus dnsLog false; all copied outbound values preserved except tags/references, no unsolicited keepalive or DNS bootstrap. Whole selected real pools join the one global leastLoad. Active loopback-dependent routes fail explicitly; inactive helpers are retained.

Full checks and pinned-core1/2/10/50 passed; logs-off observed TCP/UDP/UDP443 fixture, browser restart and59-profile/50-member SIGKILL sandbox test passed. Production byte comparison rebuilt85cached profiles from850accepted inputs and confirmed3400copied outbounds plus two template helpers per profile. These are point-in-time counts, not promises about a changing source pool. Android/TUN and live QUIC remain not_run.

Release:/opt/config-repair/releases/p15-b003f05d4d978a956f0e. Schema9 unchanged. Stopped-service backup:/var/backups/config-repair/pre-smart-envelope-20261101-062348. All10source definitions, all settings and ordinary/Smart rotation fingerprints were preserved. Both panel and shared worker hashes were verified on all three enabled/active services. Receipts: .local/p15-deployed.json, p15-audit.json, p15-public.json, p15-{check,browser,worker}.log and p15-published.json. Same full/single URLs; cache rebuilds under v2. Publication parent478491b43432b5855a927f7ee7dd21485411bef6. Never infer deployed bytes from a Git commit.

## P16 Smart fresh-connection admission — 2026-11-01

Parent15f74516d6980b855ae33e701368463bf2effccc. User reports poor speed/stability in a screen recording. The video shows core startup and application loading stalls; it does not measure UDP loss or establish a unique cause. A live audit found unverified inputs admitted by the prior core-only branch, including102of650member selections in one ready-cache snapshot.

Smart v3 now requires fresh source-owned connection=passed for every input. Cache expiry is min(member lifetime, proof deadline); all current ownership gates remain. Failed/partial/not_run/stale inputs are excluded until successful retest; sources without fresh usable members have no forced quota. P15 exact JSON, logs-off, all selected outbounds and ordinary output remain intact. User count50/revision4 preserved. No permanent bank, schema change or network parameter tuning.

Full checks, explicit quality lifecycle/source-isolation tests, browser/restart and59profile/50member SIGKILL sandbox integration passed. Current production audit:8ready profiles,400selections,0without fresh proof. Cached-byte recomposition also passed. These are time-specific counts; client speed/p95/UDP loss remain unmeasured. See docs/SMART_PERFORMANCE.md for the staged comparison plan; do not market this gate as completed client optimization.

Release:/opt/config-repair/releases/p16-0da8ca3cf86edbb5ec23; backup:/var/backups/config-repair/pre-smart-quality-20261101-065158. All10source definitions/settings and normal/Smart turns preserved during deployment. Both binaries verified on three enabled services. Receipts:.local/p16-deployed.json,p16-quality.json,p16-audit.json,p16-public.json,p16-published.json and p16-check/browser/worker logs. Same-schema rollback uses matching prior binaries and retains live user data; it reintroduces the old weaker admission, so only do so deliberately.

## Current P17 — responsive Smart and single output
Read docs/checkpoints/P17-smart-responsive-20261101.md first. Current Smart v5 deliberately supersedes the fixed P15 envelope under the user's latest performance request: leastPing, burst10/1/3, explicit block fallback, SOCKS/TUN DNS interception, proxy-routed non-IP DNS and exact endpoint bootstrap. Complete copied outbounds/credentials and logs-off remain. Current user count is10; preserve live settings rather than restoring older50-member notes.

Single Smart rounds contain only eligible ready cache; pending rebuilds no longer force empty204. Unavailable entries are skipped, new ready heads join the next cycle; zero ready returns503 with a readable message. Ordinary/source single behavior and URLs remain unchanged.

A live two-profile canary exposed Instagram TLS failure despite general HTTPS success. The normal sandbox driver now adds a bounded Instagram homepage check after both public checks. Only source-owned fresh https_multi_site_verified results enter Smart. Ordinary connection='passed' still means the two original destinations passed; code distinguishes site failure. Candidate timeout budget accounts for the third request. No schema change or normal-template version change; deployment requeues old passed results for site validation without deleting normal output.

Final receipts: .local/p17-final-check.log, p17-final-browser.log, p17-final-worker.log, p17-conversion-worker.log, p17-responsive.log, p17-deployed.json, p17-public.json, p17-audit.json, p17-site-canary.json and p17-published.json. Earlier p17 preview used Smart v4 before site admission; do not confuse its receipt with final v5. Runtime checksums and actual release are in PROJECT_STATE.json after deployment. Same-schema rollback selects the matching prior binaries/driver; do not restore an old DB over newer user data.

Controlled request latency/failover results are lab-only; real carrier/Android/QUIC/throughput remain not_run. Instagram homepage success on server does not prove media/app behavior or each nested member of a balanced source. Existing Custom JSON must be reimported to receive the new client envelope.


## Current P18 acquisition — 2026-11-02

Schema10 adds only stored/suppressed receipt counts. Current source ownership now overrides historical HTTP/content success, and deletion suppression is explicit rather than hidden as duplicates. The scoped restore button undoes only deleted identities from that source's unexpired retained response, fences old jobs and queues fresh fetch/validation. Subsequent deletion remains authoritative. Source schedules and enabled states were preserved. Final public browser observed Speedray31/31 printable and large Agood92/92; rotating single Agood1 pending/0 printable was accurately amber. No Android/network-speed claim. See P18 checkpoint for tests, exact release/hashes and the two distinct migration vs same-schema rollback procedures. Current runtime/publication pointers are in PROJECT_STATE.json; private p18 receipts are authoritative.


## Current P19 — user-selected leastLoad
Smart v6 uses leastLoad expected3/maxRTT2s/tolerance0.15. Keep current35members/revision10, burst10s/1/3s, block fallback, DNS, all copied outbounds and fresh site admission. Schema10/ordinary template v4/P18 acquisition fixes remain. Prior leastPing cache is ineligible and rebuilds without resetting settings or rotations. Full checks, controlled traffic, Smart50worker recovery, public full/single output and exact-copy audit passed. Read docs/checkpoints/P19-leastload-20261102.md and private p19 receipts; P17 leastPing performance is historical. No new carrier/Android evidence.


## Current P20 — Smart v7
Use the P20 checkpoint and PROJECT_STATE runtime pointers. The generator is shared across all existing Smart output/cache/API paths. Normal source/global outputs stay template v4. Dynamic endpoint domains never include SNI/Host; equal-width flattened proxy numbering avoids prefix collisions above99. Bootstrap requires an independently IP-only endpoint/dependency graph. No-IP profiles block with smart_bootstrap_no_ip. All original helpers survive; only tag references and requested Smart socket policy change copied values.

Requested UseIPv4 permits system-resolver fallback when core DNS fails. The isolated TLS HTTP/2 fixture distinguishes actual proxied DNS from direct/system attempts and blocks egress with a test-only firewall; do not claim that firewall exists in exported JSON. Do not silently switch to ForceIPv4 or weaken TLS. The request did not authorize replacing the user's endpoint domains with resolved IPs. Existing Android Custom JSON needs refresh/reimport; no current APK/carrier evidence.


## Current P21 — RealPing and random order
Smart v8 fixes Lib40 group RealPing by placing an independently IP-addressed proxy first. Exact pinned Lib source strips routing/DNS/observers, so the former first block necessarily fails. Full normal routing keeps both leastLoad pools, block fallback and all other P20 settings. Group ping remains one proxy's result, not every member's health.

Full/single subscription delivery randomizes proxy order per GET, retaining independent-IP first and every object/tag/reference unchanged. Full also randomizes profile order; newly created single rounds are random while unfinished turns persist. Canonical individual/API cache bytes remain stable; body and cache hashes are separate. The stale UI leastPing/always-first-head wording is corrected. See P21 checkpoint and .local/p21-* receipts for checks/runtime/publication; preserve current live counts/settings, not historical snapshots. Phone was offline during investigation; APK/carrier validation remains not_run.

## Current P22 — independent publication and pipeline recovery
Read the P22 checkpoint and updated architecture. Schema11 separates Smart cache from jobs: growth schedules a background improvement, while exact proof/ownership/settings/version/TTL remain immediate safety gates. Template v4 and Smart v8 JSON, random delivery and ordinary/source rotations are unchanged. Atomic ciphertext replacement heals same-hash corruption without changing plaintext. Shared compiled registries are race-safe; worker idle polling and source IP fallback are bounded; UI state reads are scoped and fetches recover after timeout; full exports use unlinked spools instead of whole-subscription RAM buffers.

Require final p22 check/worker/browser/traffic receipts, isolated migration/rollback, stopped-service backup and separate runtime/source readback. Preserve actual current source count/settings. Performance figures are database/server measurements; phone/operator tests remain unperformed. Never restore an old production DB over later user changes when downgrading the additive cache schema.


## Current P23 — independent Smart Splash (includes unfinished P22)
Read the P23 checkpoint. Schema13 and the schema12 source Splash flag, source toggle/bulk/edit UI, independent count/enabled settings and full/single/canonical links use the existing Smart v8 generator. smart_* and splash_smart_* repositories separate queue/cache/provenance/counters/rotation/worker state. Duplicate identity/candidate ownership is reserved for Splash and excluded from ordinary Smart; raw and ordinary exports remain unchanged. Production csp resources are separate from csm/crw, tests use cpt.

P22 recovery is included in this release, not a previously deployed P22. The browser fixture's domain-only preparation failure is now completed truthfully; pool growth bypasses blocked backoff through an inventory fingerprint. Do not complete a preparation Issue as a synthetic core pass. Preserve all requested Smart v8/P20/P21 globals and every input connection value.

Private p23-validation, dbbench-rollback, deployed, audit, public and published receipts determine final status. Legacy P21 rollback is guarded against any Splash adoption to avoid cross-contamination. No phone/carrier/Android evidence is added.


Final server release: /opt/config-repair/releases/p23-828164866449b4c7c9a5-r2. Schema13; all four services enabled, running and sandbox-healthy. Matching backup: /var/backups/config-repair/pre-splash-query-20261109-013708. Sources/settings/rotations/ownership were fingerprint-equal across the stopped migration. Normal count 50 / revision 11 retained; Splash defaults10members with independent settings. All six final checks passed; normal and Splash worker tests each built55profiles of50members across3sources and recovered after SIGKILL. Browser mobile/restart/toggle/export checks, simultaneous namespace/route isolation, canonical Compose audit and public HTTPS verification passed. Runtime panel SHA256 828164866449b4c7c9a54d3152372ca05c968cc4a759d6643018195b23516d6c; worker SHA256 3b8960ded16dffda5a9ddc947968bff3128daf10232042af8fa25657630706cd. Exact source commit/tree is in .local/p23-published.json, independently read back from origin. Independent cloud-browser navigation stopped at ERR_CERT_COMMON_NAME_INVALID; it was not bypassed. Server-side TLS validation confirmed the exact IP SAN and TLS1.3; public/browser checks refer to the server-side verifier. No Android/carrier test or blanket zero-defect/performance guarantee.


P23 published and runtime read back at1578810b04fa442640297d1f29293c0ad4c1d511. Continue the user’s broader video-led P24 audit; see docs/checkpoints/P24-system-audit-20261109.md and current PROJECT_STATE. No new P24 production patch has been deployed yet.

P24 implementation: independent bounded UI polling and thin settings metadata endpoints; preserved drafts/409 recovery/loading gates; current-row checksum-protected JSON viewers; three read-only WAL connections alongside the serialized writer; shared unlinked plaintext spools. Full check and browser tests passed. Initial traffic recovery test was probabilistic; pinned Xray shouldSelectNode excludes fail/all>0.15 even while alive remains true, and burst scheduler window is15*3=45s. The fixture now observes the exact exclusion predicate with a window-derived deadline. Final traffic/worker acceptance, backup/deploy/public readback/publication are pending. No production P24 patch has yet been activated.

P24 final: all required final checks and three worker recoveries passed; public HTTPS browser and8-profile exact composer/copy audit passed. Deployed /opt/config-repair/releases/p24-7f0e103e325c54015669-r1 with matching stopped-service backup and unchanged source/settings fingerprints/schema13. Four enabled services are healthy. Current settings ordinary50/Splash1 are user-owned; preserve them. See P24 checkpoint and .local/p24-{validation,deployed,public,audit,tested-runtime,published}.json; final publication receipt stores exact readback SHA/tree/runtime hashes. Earlier failed recovery diagnostics and the pre-degenerate-case public verifier are historical, not final evidence. No further P24 code/deployment action is pending after the publication receipt. Actual phone/carrier tests and standalone installer remain separate unverified work.
