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

## P10 mixed input
- Read docs/checkpoints/P10-mixed-input-20261030.md. Preserve URL path case and distinguish upstream HTTP errors from parsing failures.
- Mixed Base64/JSON/URI extraction must preserve exact decoded payloads and whole custom JSON documents. Never split child outbounds or invent delimiters/credentials. Keep strict padding checks, bounded envelope nesting, cancellation, malformed-sibling isolation and source ownership/dedup regressions.
- Collection success is separate from conversion/core/connectivity. Mixed source originals may remain blocked by the existing explicit conversion gates.

## P11 balanced conversion and source evidence
- Read docs/checkpoints/P11-exir-repair-20261030.md. Template v2 and structural v2 are current; source/TTL ownership and schema8 remain unchanged.
- Explicit universal balancer routes preserve original pool tags, strategies, fallback, observers and chain dependencies. Never substitute one arbitrary member or make loopback the implicit default. Infrastructure tags must not match pool/observer prefixes.
- Source summary counts use Output's canonical ranking after source filtering; original-response diagnostics are not connection evidence.
- Mixed IPv4/IPv6 pools can be tested through IPv4 without changing IPv6 members. The existing inet default-drop sandbox remains mandatory; a pool pass is not a per-member health claim.

## P12 open client traffic
- Template v3 applies the user's explicit TCP/UDP/QUIC policy. Retain SOCKS UDP and tcp,udp routing; QUIC is carried over UDP, not a routing network enum.
- Normalize only known VLESS Vision UDP443 and existing Mux rejection gates. Preserve null/absent mux, explicit native-UDP skip, concurrency, endpoints, credentials and TLS. Do not enable multiplexing or invent another wire protocol.
- Real isolated TCP/UDP/UDP443 regression is available through XRAY_TRAFFIC_TEST=1; use fresh SOCKS UDP associations for negative controls. This synthetic echo evidence is not a live QUIC/HTTP3 handshake or a promise of every upstream server's UDP support.

## P13 shared URI compatibility and source diagnostics
- Read docs/checkpoints/P13-shared-repair-20261030.md. Current template v4, structural v3 and probe-plan v3 supersede earlier version notes.
- Literal userinfo space repair must preserve decoded credential bytes, raw source bytes and all other URI bytes. Ambiguous delimiters/invalid escapes remain blocked.
- TLS cs maps to cipherSuites without rewriting its order or fingerprint. Explicit VLESS packetEncoding=xudp maps to an XUDP manager with one UDP stream and TCP multiplexing disabled. This source-requested mapping is the narrow exception to P12's no-new-mux rule; never add it to inputs without that explicit option or silently discard unsupported encodings.
- Missing/invalid REALITY key/shortId diagnostics apply to retained chains, pools and nested download streams. Never fabricate keys, borrow another config's proof, downgrade security, or label a failed HTTPS probe connected.
- Canonical issue counts must use the same source-filtered representative as Output. Pending/stale results are not final failures; raw-response format status remains distinct.

## P14 independent Smart Output
- Read docs/checkpoints/P14-smart-output-20261101.md and docs/CONVERSION.md. Schema9 is additive; ordinary template-v4, source/global URLs and turns must stay unchanged.
- Use one cached profile per canonical head, persistent source/member counters, no duplicate IDs/candidate hashes, random follower order, and an independent durable first-position round. Small-source reuse is necessary for per-profile source coverage; never promise conflicting global no-repeat semantics.
- Preserve accepted source-specific proof and dependency graphs. Namespace tags/selectors/costs/references, retain pinned weight semantics and explicit real-proxy fallback. No fabricated credentials or direct fail-open.
- Smart composition must pass the exact pinned core in the sandbox. csm production and cst tests are disjoint from normal crw/ctw. Never expose Smart worker routes on TCP or run arbitrary configs directly.
- Cache validity depends on all members/TTL/proofs/settings and current usable capacity/coverage. Blob I/O stays outside SQLite transactions; tests must cover source growth, revocation, stale completion and cross-Store rotation.
- Run full checks, Smart50-member SIGKILL worker integration, observed failover TCP/UDP/UDP443 fixture and inline browser/restart checks. These are not Android or QUIC handshake receipts.

## P15 exact Smart envelope — supersedes P14 global composition
- Read docs/checkpoints/P15-smart-envelope-20261101.md. Smart v2, schema9 and normal template v4.
- The user explicitly requires ALL selected input outbounds copied and every other global field fixed to the supplied template, with all logs off. Never prune DNS/direct/block helpers, inject keepalive/bootstrap DNS, or carry source globals into Smart.
- Preserve every outbound value except required collision-free tag/reference renaming. Deterministic field order, proxy-01 onward, complete copied helpers, final template direct/block. Immutable originals remain untouched.
- A selected source balancer contributes its entire selected real proxy pool to the single fixed global leastLoad; do not import inner strategies/costs/rules/observers. An active loopback-dependent chain cannot be represented by the fixed routes and must fail explicitly. Inactive helpers are still copied.
- Fixed expected3/maxRTT2s/tolerance0.15, the exact three DoH servers/two rules/burst settings and ordered root keys are contract-tested. Never call pinned-core acceptance proof of exact template equality.
- Version changes revoke old leases/cache without resetting settings or either rotation. Run independent envelope/all-outbound-value tests, pinned-core1/2/10/50, logs-off packet test, worker SIGKILL and browser/restart.

## P16 Smart connection quality
- Read docs/checkpoints/P16-smart-quality-20261101.md and docs/SMART_PERFORMANCE.md. Smart v3 keeps P15 JSON unchanged; admission now requires fresh source-owned two-destination HTTPS success for every selected input, including requires_connection=0 conversions.
- Failed, partial, not_run and expired proofs are ineligible. Use min(artifact expiry, proof deadline); stale completions/old cache cannot bypass current ownership/proof checks. Source quotas apply only to eligible sources; never fill requested capacity with unverified members.
- Normal output remains separately core-gated. Preserve user count/settings, all selected outbounds and immutable originals. A balanced input's route pass does not prove every inner outbound.
- Do not equate HTTP probe failure with UDP packet loss or core acceptance with speed. The video shows startup/stalling, not a measured cause. Android throughput/p95/UDP loss remain unmeasured.
- Any future speed profile changing expected/burst/DNS must be separately compared; no undocumented JSON hysteresis/adaptive capability. Do not claim native TCP session migration or zero loss.

## P17 responsive Smart and available single output
- Read docs/checkpoints/P17-smart-responsive-20261101.md and docs/SMART_PERFORMANCE.md. The renewed performance request authorizes the explicit P15/P16 envelope changes described there. Smart v5; normal template v4 and schema9 remain.
- Preserve all member outbound values except namespaces, logs off, current user count (10 at final verification), immutable originals and ordinary/source exports. Do not reset settings/rotations.
- Smart uses leastPing with burst10s/sampling1/timeout3s and block fallback. DNS interception covers socks and the pinned app's injected tun; scoped endpoint-only bootstrap must not include SNI/Host/app domains. Non-IP DNS forwards via the proxy loopback, never directly.
- Admission requires fresh source-owned code https_multi_site_verified: existing two public HTTPS passes AND a successful TLS-valid Instagram homepage request. General connection status/export remains independent. No bank or unbounded diagnostics; failures auto-retest under current TTL.
- One first-position round snapshots only smart_ready. Remove unavailable heads before deciding whether to start a new round. Newly ready entries join next round. Return503 with explanatory text/Retry-After only when none are ready; never successful empty204 because other heads are warming. HEAD/prefetch do not consume.
- Controlled comparison proved DNS A/TXT, endpoint bootstrap, lower latency under a deliberately delayed peer and recovery after preferred-peer failure. It did NOT test carriers, Android APK, real QUIC, throughput or zero packet loss.
- Run complete checks, scripts/smart-responsive-test.py through its Go test with XRAY_TRAFFIC_TEST=1, both worker integrations after probe changes, browser/restart, public single navigation and fresh-proof audit. Private p17 receipts include interim and final runtime; use final deployed/published receipts, not an earlier preview.


## P18 truthful acquisition
- Read docs/checkpoints/P18-acquisition-20261102.md. Schema10 is additive; normal template v4 and Smart v5 stay unchanged.
- HTTP/extraction is not storage or export evidence. Separate deletion suppression from ordinary duplicates; zero current memberships never show green collection success.304 requires retained response and live memberships. Historical receipt counts must not override current ownership.
- Explicit recovery may clear only matching deleted identities parsed from the source's retained unexpired response; global shared identity semantics must stay explicit. Fence stale jobs, preserve unrelated/disabled controls and require new collection/validation. Later deletion still wins. Vault reads stay outside transactions.
- Preserve source schedules, enabled state, fixed TTL, settings and rotation state. Distinguish stored-but-pending output from ready/connected output, including rapidly rotating one-item sources. Test suppression/recovery in the real browser as well as both completion paths.


## P19 leastLoad — supersedes P17 strategy
- The user explicitly selects leastLoad: expected3/maxRTT2s/tolerance0.15. Smart v6; preserve burst10s/1/3s, block fallback, DNS, site admission, all copied outbound values and current35-member setting. Schema10/ordinary template v4 stay unchanged.
- Read docs/checkpoints/P19-leastload-20261102.md. Old leastPing results are historical. Controlled tests must validate requested distribution/DNS/failure behavior without claiming leastLoad always chooses minimum RTT.
- Invalidate previous strategy cache/leases while preserving settings/rotations. Verify real full/single output and copied-value equality. Existing client JSON needs reimport; no carrier or2-second guarantee without measurement.


## P20 tunneled bootstrap — supersedes P19 Smart envelope
- Read docs/checkpoints/P20-tunneled-bootstrap-20261107.md. Smart v7; schema10, ordinary template v4 and source collection stay unchanged. Preserve current live settings/rotations, not historical counts.
- The common Compose generator serves every existing Smart profile/cache/full/single/API path. Two tagged tunneled DoH bootstraps (Google then Cloudflare), two general resolvers (Cloudflare then Google), timeout2500ms, parallel queries and stale cache match the requested template. Endpoint domains come only from actual connection endpoints, never SNI/Host or arbitrary extension fields.
- Main leastLoad keeps all selected entries. Bootstrap leastLoad selects only independent IP entries including their dialer dependencies and XHTTP download endpoints; no eligible IP yields smart_bootstrap_no_ip. Never fall back to domain entries or a generated direct route. Equal-width numeric tags based on the total flattened pool prevent Xray prefix-selector collisions above99.
- Copy every input outbound/helper. Only namespace references and the explicit Smart socket policy UseIPv4/keepalive45/15 change; all other values and raw/non-Smart bytes stay unchanged. Substitute dynamic envelope fields BEFORE inserting copied outbounds, so unknown empty domains/selectors remain intact.
- Generated block is first; generated DNS/loopback helpers are last; no generated freedom. Burst15s/sampling3/timeout5s, system counters and localhost11111 metrics; all logs off. Old v6 caches and occupied leases must be rejected without resetting settings/rotation.
- UseIPv4 is retained explicitly. Xray can fall back to the system resolver after internal DNS failure, including startup. Do NOT claim this JSON alone guarantees DNS leak prevention. The closed test firewall blocks and separately counts those attempts; it is not part of exported JSON.
- Verify exact envelope/values, dynamic1/2/5/7/10/15/50/100+ pools, independent-IP chains, no-IP failure, TLS HTTP/2 DoH routing, cold failure, metrics, TCP/UDP, worker SIGKILL, browser/API/cached-byte parity. Keep test CA/firewall/accelerated observation substitutions confined to fixtures. No Android/carrier/2-second guarantee.


## P21 RealPing and per-request order — supersedes P20 block-first
- Read docs/checkpoints/P21-realping-random-order-20261107.md. User explicitly requests fixing ping and random order on every Smart subscription GET. Smart v8; normal template v4 and schema10 unchanged.
- AndroidLibXrayLite v26.7.28 (Lib40), commit d1d167b508457ebfd650c9e99bfca50c054c98df, removes router/DNS/observers in MeasureOutboundDelay. Generated default must be an independent-IP real proxy, never block/direct/domain-dependent. Full running config still uses both leastLoad pools and block fallback. A group RealPing measures only that default, not aggregate pool health.
- Keep all other P20 globals and copied values. Tags stay attached to their objects, with equal-width prefixes. Canonical cache remains deterministic, versioned and pinned-core gated; individual /smart/json/{id} stays byte-exact for the UI.
- Full subscriptions shuffle profile order and proxy object slots per request; single subscriptions shuffle proxy slots. New durable single rounds snapshot ready IDs in random order; unfinished rounds, HEAD/prefetch, ownership/TTL gates and ordinary/source rotations remain intact. Do not equate profile head ID with actual first outbound. Retain legacy X-Smart-First-ID and add X-Smart-Profile-ID. Body SHA and canonical cache SHA must be distinguished.
- Random order can repeat by chance and cannot offer multiple first proxies when only one independent IP exists. Do not invent endpoints, rewrite tags on delivery, or change cached connection proofs. Raw copied object bytes and non-outbound fields must survive permutation exactly.
- Run exact Lib40 app-filter reproduction in the closed sandbox, unchanged full-routing dead-default failover, DoH/cold failure, randomized1/2/5/7/10/15/50/100, full checks, worker SIGKILL and browser/restart. Build scripts/testdata/libping-fixture.go against the pinned Xray source using scripts/build-libping-fixture.sh, then set XRAY_LIBPING_BINARY for traffic tests. No Android/carrier guarantee without an actual device receipt.

## P22 pipeline recovery — supersedes immediate invalidation on capacity growth
- Read docs/checkpoints/P22-pipeline-recovery-20261108.md and docs/ARCHITECTURE.md. Schema11 is additive; Smart v8 JSON/normal template v4 remain unchanged. Preserve live source/settings/rotations and immutable plaintext.
- Published Smart cache/provenance is independent of jobs. Additional source coverage/capacity requests background replacement; safe current cache stays available. Member revocation/failed or expired exact proof/settings/version/TTL still hides it immediately. Publish only through validated completion; canceled/stale reads must not revoke healthy cache.
- Vault reacquisition may replace ciphertext atomically only for identical content-addressed plaintext. Retain checksum/AEAD, pins and vault-outside-transaction lock order. No plaintext response cache on disk: full-export spools are0600 and unlinked immediately.
- Compiled registries return snapshots without holding locks across plugin callbacks. Worker polling and HTTP dialing remain bounded/cancellable. Do not weaken destination/TLS/sandbox gates for speed.
- Run the full final checks, both worker SIGKILL integrations, pinned traffic/core and browser/restart tests. Measure comparable fixtures; do not represent database/API latency as operator speed. Schema11 additive rollback/re-upgrade must preserve user data and be tested before deployment.


## P23 Splash isolation — supersedes unpartitioned Smart membership
- Read docs/checkpoints/P23-splash-20261109.md and docs/ARCHITECTURE.md. Schema13 includes P22 recovery and the schema12 Splash partition; Smart v8/ordinary template v4 are unchanged. Existing sources default ordinary; preserve actual live counts/settings.
- Reuse SmartPipeline and common composition through closed repository drivers. Normal smart_* and splash_smart_* settings, jobs, cache/provenance, counters, rotation, heartbeat, public token domains and private routes are separate. Never fill Splash from ordinary members or vice versa.
- Splash ownership wins duplicate identity/candidate conflicts while retained membership exists. Filter current source classification before selection and at ready/renew/complete. Toggle edits are revision-fenced and invalidate old leases; immutable raw bytes/ordinary outputs remain unchanged.
- Independent csp production/cpt test resources and splash service/lock. Never point test cleanup at csp/csm/crw. All worker routes stay Unix-only. Shared immutable vault and conversion evidence are not separate host resources.
- Pool fingerprints bypass blocked retry delay only when eligible inputs change. Preserve unchanged-pool backoff; exclude renewed expiry timestamps from the fingerprint.
- Run full race/vet/build/core/traffic, ordinary+Smart+Splash SIGKILL worker tests, strict two-way/duplicate isolation, source lifecycle, migration/rollback, browser/mobile/restart. Verify public normal and empty/new Splash surfaces without marking a user source automatically.
- Legacy schema10 downgrade is allowed only before Splash adoption (no marked sources or Splash heads); refuse it afterward rather than reintroduce mixed output. Preserve live data, take matching stopped-service backup, publish/read back the exact tested tree and separately verify runtime hashes.

- Schema13 replaces the correlated duplicate-ownership OR scan with two non-null current-query reservation sets. Preserve exact identity/candidate precedence, including retained disabled Splash membership and absent conversion rows. Never persist these sets or bypass current proof/TTL gates. Verify adopted schema12 migration/rollback/re-upgrade with all table-row fingerprints and equivalent query results.
- Normal/Splash integration fixtures must use their own bound ephemeral loopback listeners, scope-specific readiness files and private binaries. Never infer fixture readiness from a shared fixed port; preserve cst/cpt resource separation.


## P24 independent reads and browser recovery
- Read docs/checkpoints/P24-system-audit-20261109.md. Schema13, Smartv8, ordinary templatev4 and all source/connection policies remain unchanged.
- Store has a serialized writer plus a bounded mode=ro/query_only WAL reader. Never move claim/rotation/mutation transactions or GC reference checks to the reader, or perform vault I/O inside transactions. Reconnect pragmas and committed-state visibility are mandatory.
- Browser features poll independently; thin settings APIs must not enumerate output rows. Preserve unsaved drafts, require explicit resubmission after409, disable unknown initial settings/copy links, and apply JSON only to the current matching-hash row. Keep the asset import HTTP regression.
- Acquisition/envelope/restoration/export plaintext spools must be unlinked immediately; format drivers reopen the /proc/self/fd path while its owner is alive. Do not reintroduce named plaintext caches.
- leastLoad failure detection tests must use the pinned shouldSelectNode predicate (alive, delay, fail/all tolerance), not three randomly successful requests. Burst scheduler interval is interval*sampling; do not claim a2-second failure bound from maxRTT. Preserve the user-requested JSON envelope.


## P25 inventory and receipt accounting
- Read docs/checkpoints/P25-inventory-accounting-20261109.md. Schema14 adds only source receipt/recovery metadata; Smartv8 and normal templatev4 remain unchanged.
- Current unique membership, extracted occurrences, body-local duplicates, pre-existing/shared identities, disabled inputs and output/connectivity proofs are different counts. Legacy unmeasured receipts remain unknown; never infer unique count from the old combined duplicates field. Whole custom JSON remains one input with all internal outbounds preserved.
- Queue enabled sources before every expiry cascade, cap next scheduling at retained expiry, force unconditional requests for incomplete inventory and keep recovery retries bounded. Do not extend fixed TTL, clear deletion controls, enable sources, mix Splash or invent fresh proof. Retention recovery may precede the user's normal random schedule without altering its min/max settings.
- State summary reads use one readonly WAL snapshot and one expiry moment. No claims/mutations/rotation/blob I/O in that context. Preserve ordinary/source/Smart links and current settings.
- Require duplicate/partial304/expiry/restart/backoff/snapshot regressions, full checks, browser outage→expiry→automatic refill, worker recovery, migration/rollback and public source/API verification before reporting deployment.


## P26 independent Smart Ads
- Read docs/checkpoints/P26-smart-ads-20261110.md. Schema15; Smart Adsv1 is a third compiled pipeline. Existing Smartv8 and ordinary templatev4 stay unchanged. Preserve every current source/setting/rotation and the P25 fixes.
- Exactly one whole fresh Splash input occupies position0; main inputs come exclusively from ordinary offers. Every eligible Splash candidate is a head. Position-specific ownership and exact fresh proof gate claim/renew/complete/read, and source classification cannot silently swap roles. Count is main members plus one Ads input.
- ComposeAds reuses Compose; keep all outbounds and namespaces. Only terminal supported Ads transports gain the main carrier. Preserve existing chains/download streams, reject ambiguity/cycles/unsupported custom dialers, keep Ads entries out of main/bootstrap, and never create direct fail-open. Balanced Ads source remains a whole input with its own exit pool.
- Preserve the supplied Ads domains and require checksum-pinned geosite asset. Source-health proof and core validation do not prove a real end-to-end Ads session. No Android/operator/QUIC/zero-leak guarantee.
- Independent csa production/cat worker test/csta traffic resources, service, locks, token domains and private Unix routes. GC protects both pending and published Ads blobs. All five services require separate runtime hash verification.
- Require full checks, all four worker recoveries, controlled chain/failed-exit/negative-bypass TCP/UDP/UDP443, browser/restart, additive migration/rollback, stopped-service backup, public export verification and exact source publication/readback.

- Positive support-x25519mlkem768=true/1 is an explicit VLESS REALITY URI compatibility hint, not an Xray JSON field. Preserve native candidate/fingerprint/credentials and raw bytes; reject disable, malformed, duplicate and mismatched-security uses. Never use hint acceptance as connection proof.

## P27 production recovery
- Read docs/checkpoints/P27-production-recovery-20261110.md. Keep schema15/templates/target and all live sources/settings/rotations unchanged.
- Advance HTTP validators only with an accepted generation; rejected bodies clear validators and use bounded unconditional recovery. Increased retention must never extend existing deadlines.
- Repository drivers implement exact revision/token-fenced Abandon for known discarded claims. Smart committed handoff failures clean up only their own token; unknown/crashed outcomes keep lease recovery.
- Cache cleanup errors propagate. Bound only failed blob/generation recovery, not successful cross-store turn competition. Full Smart spools revalidate exact cache provenance before successful headers; zero/revoked full output is503.
- Shared availability is nonconsuming and scope-token protected. Browser-only waiting HTML must never replace successful JSON or leak into VPN clients' responses. Keep bounded polling and mobile/recovery regression.
- Match actual error signatures, never incidental certificate vocabulary. XHTTP download endpoint allowlisting follows native extra precedence; public-only firewall/TLS/proof gates remain. Show connection diagnostics even for core-valid exportable rows.
