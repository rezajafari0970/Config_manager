# Current P34 acquisition recovery — 2026-11-11

The current user requests fixing intermittent acquisition of healthy sources.
A live 186-occurrence response arrived in1.308s and parsed in<1ms while the old
collector repeatedly exhausted its shared30s deadline and owned zero inputs.
Fetch/body consumption now has the configured network timeout; processing has
an independent120s budget and completion30s, all under bounded source admission.
Leases cover those budgets; source edit/delete/shutdown cancellation still wins.

Optional StageBatch commits at most32 metadata records with exact raw/candidate
pins until commit, preserving atomic nonempty generation publication. Reuse a
ciphertext only after complete AEAD/hash verification; repair corruption by
atomic replacement. Reject underlying truncated streams and unauthenticated
EOF at ciphertext record boundaries. Additive blob-reference indexes keep
schema20 and existing controls; GC uses current writer reads and never snapshots
away a new reference. Raw bytes, fixed TTL, manual deletion, ordinary/Smart
proof gates, target cores, source settings and rotations remain unchanged.

Read docs/checkpoints/P34-acquisition-recovery-20261111.md and private deployment,
validation and publication receipts before claiming runtime state. API review
used gpt-6-astra; model output is not network health evidence. Empty upstream
responses and actual network failures remain explicit, bounded retries.

## P34 deployed evidence

Installed release: p34-25bcb3a7bb488e1434d7 (schema20). All six services and five
fresh worker heartbeats were checked after activation; PID executable hashes
match the tested binaries. Source controls, rotations and publication gates
were preserved. All33 sources completed a fresh collection:32 have inventory,
with724 unique inputs at the verification snapshot. The formerly empty worker
source succeeded three times:186 occurrences each,142/151/149 unique inputs,
12.289/3.639/2.848s. Rotating content explains differing unique counts.
Exir still returned an empty body; sub returned incomplete_body and retained
its161 previous inputs. Neither is relabelled a successful fresh import.
Live mobile/desktop rendering verified the recovered count. Final exact-export
build checks and existing real-runtime proof checks are in P34-validation.json.
API review used gpt-6-astra; no ongoing AI dependency was added.
Publication and canonical rebuild identity are recorded privately in
.local/p34-published.json; main remains unchanged.

# Current P33 handoff — 2026-11-11 (Asia/Tehran)

P33 supersedes previous target/admission/subscription and output-shuffle notes.
Official target is v2rayNG2.3.15 / Lib40 / bundled Xray26.10.31, with additional
26.11.9 exact-build validation. Real final-byte connection tests run on26.10.31.
Read docs/CLIENT_COMPATIBILITY.md, the P33 checkpoint and current private receipts.

Schema20 preserves source/settings/rotations, fences old results and records
bounded separate provenance. Every generated normal/Smart/Splash/Ads/classified
client export requires current exact-byte build and real transport evidence.
Raw exports remain explicitly unqualified originals. Native subscriptions use
JSON arrays; old JSONL URLs remain supported for integrations. Published profile
bytes are never shuffled after validation; durable profile rotation remains.

The user explicitly authorizes the narrow legacy VLESS plaintext TCP/RAW HTTP
loopback relay. It adds NO encryption. Unique ports/tags, exact destination allow,
relay deny, carrier handling and graph recognition are common to all composers.
Other incompatible protocols/transports remain explicit; no forged server fields.
Generated socket DNS uses forced strategies (including XHTTP download) to stop
system resolver fallback; global DNS queryStrategy stays UseIPv4. Cold failure
and normal startup fixtures both recorded zero system DNS attempts.

Current receipt files identify actual deployment/publication, not this header.
No Android APK/carrier/long-term/IPv6-only guarantee. RealPing strips relay
inbounds and may fail despite full-profile connectivity. HTTP3 canaries are real
TLS-valid DoH tests; generic QUIC VN alone is not full HTTP3. TC81 was unreachable
from this project server at this test time and is not certified healthy.

# Current P32 handoff — 2026-11-10

The reported legacy VLESS vnext shape passed Xray26.7.28 but failed on26.11.9 with
the exact transport-security prohibition. Inactive tlsSettings do not enable TLS.
One shared sandbox gate now checks exact ordinary and Smart/Splash/Ads bytes on
checksum-pinned26.11.9 before the existing pinned validation/connection workflow.
No protocol/credential/security changes are inferred. Raw inputs remain intact.

Schema19 updates five persisted version predicates and cancels outdated jobs.
Merely changing Go constants would leave installed SQL views using old proofs.
Ordinaryv5, Smartv9 and Adsv2 fence both old results and cached profiles. User
settings, sources, counters, rotations and immutable source bodies are preserved.

Tests include old/new-core regression, public/private VLESS, active/inactive TLS,
REALITY, Trojan, VMess, invalid object shapes and an unused incompatible helper;
failed compatibility is never a claimed network failure. All current-client
checks stay inside existing bounded namespaces, cgroups and filesystem isolation.
Public logs contain bounded codes, never the config or the raw core error.

The Android parser tries an array, logs its mismatch, then accepts a single object.
No root-array rewrite is needed for the supplied input. Full JSONL remains the
requested display/export format, not a claim of native batch subscription support.
Client reimport/update is necessary; already downloaded bytes cannot be recalled.

Parent a0289f9b364a6289bcaae362cf1e0b8fa3e3d910 on go/v2rayng-2.3.6.
Private receipts will identify actual validation, release and publication state.
Android app, carrier connectivity and live QUIC handshakes remain not_run.

# Current P31 decision — 2026-11-10

A route difference and a Residential access network are distinct results.
The user accepts the former under an explicit scoped label without inferring the
latter from country/ASN/ISP. Keep separate public datasets and claims. Paired
controls must be stable across >=3 waves; Ads must be disjoint from the complete
control pool, with valid attributable observations in both raw and common form.
Equivalent IPv6 representations must compare equal. Unbound negative reports
cannot regain deletion authority. Correlated failures latch a cooldown immediately.

The latest instruction restores unlimited age for accepted members while evidence
remains short-lived. All new input hashes enter the fair queue; untested entries
cannot silently expire before first assessment. Manual deletion, tombstone epochs,
current evidence eligibility, unknown quarantine and source/settings independence
remain intact. Optional native drivers cannot redefine the ordinary Xray envelope.

# Current P30 decision — 2026-11-10

Latest-core rejection must trigger bounded official-core compatibility testing,
not a fabricated repair or an unhealthy verdict. A whole raw/template result
from one checksum-pinned core is selected. Two independent controls, repeated
Ads samples, per-family comparisons, exact-IP metadata and separate trust/coverage
requirements remain mandatory. Fullscope tests showed IPv6 candidates on most
domains and IPv4 candidates on googleadservices for the same two working inputs;
this difference alone is not a distinct residential route. See P30 report.

# Current P29 decision — 2026-11-10

The user explicitly changed the residential lifetime requirement from indefinite
to10h shared fixed retention and ordinary health cache10h, panel-configurable.
This supersedes older P28 notes. Ads observation expiry remains10min; no cached
health result or IP metadata replaces a fresh route measurement. Raw ZIP becomes
inline exact raw text. Country/ASN differences are useful diagnostic observations,
not sufficient evidence for Residential or for dedicated routing. P29 adds
per-domain diagnostics, metadata-only plugins, structural import normalization,
visible progress/pagination and default-outbound path correction. See P29 checkpoint
for tests, deployment identity and live proof limitations.

# Project Brain — P24

Current user instruction (2026-10-30) replaces the persistent repair laboratory with a shared JSON conversion flow. Read PROJECT_STATE.json, HANDOFF.md, docs/CONVERSION.md and the P21 checkpoint. Earlier checkpoints describe history only.

Go fetches source links independently, extracts exact encrypted originals, maps supported protocols to the user's @shenvpn full JSON envelope, validates using pinned Xray26.7.28 and tracks connection separately. Output is a clean list plus the all-config JSONL and single-config rotation links. The bank, manual import, historical variants, learned model/rule data and old subscriptions are removed by schema6.

The actual endpoint/auth/transport must be retained, with explicit errors for unrecoverable/unsupported mappings. Common DNS/inbounds/routing/log/policy intentionally replace source globals per the new user instruction. Domain endpoints receive scoped local DoH bootstrap to avoid DNS recursion. TLS hypotheses require fresh connection proof. Core-valid ordinary mappings may be exported with unsuccessful/unknown connection shown accurately. Never claim Android validation; JSONL is not the pinned app's native JSON-array subscription format.

Source ownership, fixed default2h retention, global identity dedup, exact raw bytes, independent revisions/leases/deadlines, encrypted storage, passwordless HTTPS/CSRF, bounded parallel work and systemd recovery remain. Conversion data cannot outlive active memberships. Untrusted execution stays in the proven sandbox; private worker API is Unix-only.

Main boundaries: internal/collect drivers; internal/repair deterministic structural candidates; internal/probe protocol preparation; internal/convert common envelope; internal/panel/conversion*.go transient queue/outputs; cmd/config-probe-worker + scripts/probe-one.py/live-probe.py sandbox validator. No dynamic upload/execution of arbitrary plugins.

P8 adds a separate single-config rotation link. Schema7 stores only the current cycle and pending candidate hashes, with atomic SQLite claims and blob validation outside transactions. The global round is persistent across restart; new configs wait for the next snapshot, unavailable ones are skipped. See docs/CONVERSION.md and docs/checkpoints/P8-single-output-20261030.md. All-config export remains unchanged.

P9 exposes full/single exports per source and source-filtered Output views. Membership is filtered before identity ranking. Schema8 preserves the global rotation while adding independent source scopes with deletion cascades and cancellable per-scope locks. New receipts are source-export-*. Earlier source counts are historical; preserve the actual live source set.

P10 fixes mixed-response extraction: Base64 is recognized alongside JSON/URI siblings rather than only as a whole-body fallback. Padded/unpadded standard/URL-safe wrapped blocks use strict streaming decoding, nested envelopes are bounded, and the original response plus exact decoded payloads stay separate. Full custom JSON stays whole; no child outbound is promoted to an independent config. A TXT source's case-sensitive path was corrected through the revision-checked API to the exact user-provided spelling; never lowercase arbitrary URL paths. Schema8, source/global exports, conversion gates and source settings are unchanged. Read docs/checkpoints/P10-mixed-input-20261030.md and private txt-mixed receipts. Extraction does not imply core/connection acceptance; some mixed-source custom configs remain explicitly blocked.

P11 preserves explicit balancer graphs, repairs empty built-in settings and recognizes a validated XHTTP padding alias. P12 applies the explicit open TCP/UDP/QUIC policy: Vision uses its supported udp443 suffix; existing Mux UDP443 rejection becomes allow without enabling multiplexing. Template v3 revalidates current source-owned outputs. Original bytes, TLS/credentials, source/global output ownership and schema8 remain unchanged. Actual upstream UDP/QUIC support is separate from these local policy flags. Read the P12 section of docs/CONVERSION.md and current checkpoint.

P13 uses template v4, structural v3 and probe-plan v3. URI repair encodes only unambiguous literal spaces inside credentials without changing decoded bytes. TLS cs is preserved as cipherSuites, including existing fingerprint values. Only explicitly requested VLESS packetEncoding=xudp creates a UDP-only XUDP manager (TCP concurrency -1, XUDP concurrency1, UDP443 allow); absent/unknown options are not guessed. Retained REALITY streams require a valid source-supplied key, including password alias; missing keys cannot be reconstructed. Input/Output share precise error descriptions and per-source canonical issue counts. Full validation, actual packet tests, browser, worker recovery and deployment readback are required; current private receipts use hot-*.


P14 adds Smart Output using the supplied smart-parallel-v6b envelope without replacing ordinary output. Read docs/checkpoints/P14-smart-output-20261101.md. Schema9 preserves ordinary/source rotations. Selection/composition lives in internal/smart; Smart persistence/routes in panel/smart*.go; a separate enabled systemd service uses mode smart, its own Unix queue, csm namespaces and instance lock. One cache profile per unique head enforces first-position coverage, with independent source/member usage counters and shuffled followers. Count defaults10; active accepted sources participate up to capacity, tiny/shared sources necessarily reuse/deduplicate followers. Exact source ownership, proof and shortest TTL gate every profile, independently pinned-core validated. Android and live QUIC remain not_run; leastLoad distributes connections, not one connection's bandwidth. Current private receipts use smart-*.

P15 supersedes P14's Smart global merge at the user's explicit request. Every non-outbound field is now the exact ordered supplied template, except loglevel/access/error none and dnsLog false. Smart v2 copies all selected inputs' outbounds without pruning helpers or injecting socket options. Only collision-free tags/references change; selected entry tags run proxy-01 onward. One fixed leastLoad/burst handles selected real pools; source DNS/bootstrap/rules/strategies are not imported. Active loopback chains needing source routes fail explicitly rather than silently changing connectivity. Source/normal output behavior remains unchanged. Schema9 and all settings/rotations survive deployment; old Smart cache revalidates under v2. Private receipts are p15-*.

P16 responds to poor real use by fixing a confirmed Smart admission weakness. Smart v3 requires a fresh passed connection result for every selected input, not only repaired inputs that already required proof. Failed/partial/not_run/stale results cannot fill quotas or capacity; proof must match exact source/raw/candidate ownership. Cache validity is bounded by member/proof deadlines. The P15 envelope, all outbounds, logs-off and user count (50 at this deployment) remain unchanged. Ordinary output still distinguishes structure/core from connection. Read docs/SMART_PERFORMANCE.md for the measured findings and staged client performance plan: no Android speed improvement, UDP loss percentage or optimal parameter claim is supported yet. No bank or unbounded history was added.

P17 supersedes P15/P16 global Smart behavior under the latest performance request. Smart v5 selects the alive leastPing entry, uses burst10s/1/3s, blocks when no observed route is available and fixes app DNS plus exact endpoint bootstrap. All copied outbounds, TLS/credentials and logs-off remain. Single output rotates ready cache only; rebuilding entries cannot freeze it. Fresh source-owned two-destination HTTPS plus Instagram homepage success is required for Smart admission; ordinary output eligibility remains unchanged. Schema9, current count10, all source settings and durable rotations are preserved. See current P17 checkpoint and SMART_PERFORMANCE.md for actual evidence and remaining carrier/Android limitations.


P18 fixes false collection success when manual-deletion controls suppress every acquired item. Schema10 adds receipt counts; deleted items are separated from duplicates, current ownership gates success, missing-membership304 triggers an unconditional retry, and enabled live ownership gates source statistics. The UI never shows zero-current configs as successful and distinguishes pending output validation. Explicit scoped restoration parses the retained immutable response, clears only matching deleted identities, fences old work and requires fresh fetch/validation. All11 live source definitions, schedules, settings and rotations were preserved; normal template v4 and Smart v5 are unchanged. Read docs/checkpoints/P18-acquisition-20261102.md and private p18 receipts; historical counts are not current state.


## Current P19 — user-selected leastLoad
Smart v6 uses leastLoad expected3/maxRTT2s/tolerance0.15. Keep current35members/revision10, burst10s/1/3s, block fallback, DNS, all copied outbounds and fresh site admission. Schema10/ordinary template v4/P18 acquisition fixes remain. Prior leastPing cache is ineligible and rebuilds without resetting settings or rotations. Full checks, controlled traffic, Smart50worker recovery, public full/single output and exact-copy audit passed. Read docs/checkpoints/P19-leastload-20261102.md and private p19 receipts; P17 leastPing performance is historical. No new carrier/Android evidence.


## Current P20 — tunneled dual DoH bootstrap
User's 2026-11-07 template supersedes P19 globals and explicitly adds Smart-only socket policy. See docs/checkpoints/P20-tunneled-bootstrap-20261107.md. Smart v7, same schema10 and ordinary template v4. Common generation supplies existing full/single/individual/API/cached Smart output. Four tunneled resolvers, endpoint-only domains, independent IP bootstrap leastLoad and dynamic equal-width proxy tags; no hardcoded36 servers/domains. Both balancers use expected3/maxRTT2s/tolerance0.15 with block fallback. Burst15/3/5, keepalive45/15, UseIPv4, counters and local metrics11111; logs off. All copied helpers remain; generated block first, DNS/loopback last, no generated direct. Raw and non-Smart data remain untouched. Preserve live source settings and both rotations on deployment.

No independent IP produces a readable blocked-profile diagnosis, never domain-recursive/direct bootstrap. A new fixture uses actual TLS HTTP/2 DoH through synthetic proxy peers and records the independent IP source. UseIPv4 may attempt system DNS on failure/startup: fixture-only firewall containment is not an exported anti-leak guarantee. No Android/operator/throughput or2-second selection claim. Final p20 receipts, not failed intermediate fixture diagnostics, establish runtime/publication evidence.


## Current P21 — RealPing compatibility and per-request shuffle
The user's ping/random-order request supersedes P20 block-first. Smart v8's first outbound is an independent IP proxy because pinned Lib40 MeasureOutboundDelay removes routing, DNS and observation apps. Full runtime retains the requested leastLoad, block fallback and other P20 globals. Cache proof remains versioned and canonical; full/single subscription rendering permutes unchanged proxy objects, keeping an independent IP first. Full profile order and newly created durable rounds are also random. Individual JSON stays byte-exact, headers distinguish rendition/cache hashes, and unfinished turns/settings/ordinary outputs stay intact. No sample endpoints or raw data are altered. Exact-source filter regression is server-side, not an Android APK test; phone was offline. Read P21 checkpoint and private receipts before claiming deployed/publicly verified state.

## Current P22 — pipeline audit and recovery
The user's project-wide improvement request authorizes the cache lifecycle refinement described in the P22 checkpoint. Published Smart cache and pending replacement jobs are independent in schema11. Valid cache survives source-capacity growth and failed replacement, while all exact fresh-proof/ownership/TTL/settings/version gates remain immediate. No JSON envelope or connection data change. Atomic same-content ciphertext replacement fixes corruption recovery; exact content-reference indexes and keyed cache joins reduce repeated database work. Active compiled plugin registries share safe snapshots, queue scheduling has independent bounded backoff, acquisition races validated literal addresses, UI reads only current-feature data and times out hung fetches, and shared unlinked spooling bounds full-export RAM. Read docs/ARCHITECTURE.md for real extension boundaries and limitations. Final receipts and runtime fields, not intermediate test output, establish release status.


## 2026-11-09 — P23 Splash
The user requires a per-source Splash button, bulk classification, independent Smart Splash settings/member count and complete/single links with strict mutual exclusion. This continues the unfinished P22 audit; P21 remains the parent. Implemented one shared SmartPipeline with separate persistent repositories, HTTP token domains, counters/rotation and sandbox worker. Splash ownership takes precedence for duplicated identity/candidate bytes; changing classification invalidates incompatible cache and stale completions without changing original bytes. See P23 checkpoint and current release receipts, not earlier preview notes.

P23 final schema13 retains the schema12 Splash partition while replacing a repeated duplicate-ownership scan with query-local identity/candidate sets. Smart v8 and all member proof/TTL/ownership rules remain unchanged. See the P23 checkpoint for the reproduced public full-export timeout, equivalent-query benchmark and adopted-data rollback evidence.

## Current P24 — independent reads and visible recovery
Read the P24 checkpoint for the video-led audit. Bounded read-only WAL access separates presentation from the serialized writer; independent feature loops and thin settings APIs remove shared-summary gating. Form state,409 recovery, matching-row JSON delivery and anonymous plaintext spools have regressions. Schema13 and both JSON templates are unchanged; preserve every live source/setting/rotation. Do not claim missing credentials/unsupported semantics are repaired, or equate API timings with Android/operator speed. Final validation/deployment/publication receipts determine completion.


## P26 — Smart Ads
The requested third pipeline combines exactly one whole fresh Splash input as Ads exit with ordinary Smart carrier members. Shared selection/composition and scope-specific repositories keep health/ownership/TTL and durable turns independent. Read P26 checkpoint for the chain contract, geosite asset, five-service release and controlled traffic evidence.

P26 final: third Smart Ads scope deployed and verified, with positive REALITY ML-KEM URI-hint compatibility. Its current live availability depends on fresh Splash proof; current server probes do not supply a healthy Ads head. See P26 checkpoint and PROJECT_STATE for final receipts, source/runtime identity and explicit untested phone/session boundaries.

## P27 — production recovery
The user's broad repair request and empty Smart Ads screenshot led to the bounded handoff/cache/export and availability changes documented in the P27 checkpoint. Keep current target/schema/templates/source controls. Health-gated Ads remains unavailable without a fresh qualified Splash head; a readable auto-updating503 is not synthetic connectivity. GPT-6 Astra API source review completed privately. Consult final PROJECT_STATE and p27 runtime/source receipts for release identity and actual verification.


## P28 — independently retained residential Ads evidence
The user's explicit eleven-part instruction overrides the general age-limited lifecycle for accepted residential records. It requires independent health/egress/network axes, real attributed scope-bound evidence, strict Unknown/Partial/Unsupported handling, no residential verdict from ASN differences, retained data independent of sources/2h age, immediate publication quarantine and confirmed global revocation with backup-resistant tombstones. The implementation, synthetic acceptance matrix, real screening limits and final R2 deployment are in docs/checkpoints/P28-residential-20261110.md. All current sources/settings were preserved. Three residential outputs and their UI are live but empty because authorized egress and independent specialist evidence are not configured. Native forced IPv6/UDP/QUIC and full balanced-path coverage remain missing. Do not turn experimental same-TLS candidates into trusted evidence. Source/runtime receipts are authoritative; old checkpoints do not establish current behavior. The API performed sanitized gpt-6-astra reviews only, with no per-connection or deletion authority.


## P32 verified release

Mandatory ordinary/Smart/Splash/Ads compatibility on checksum-pinned Xray26.11.9
is deployed with schema19 and ordinaryv5/Smartv9/Adsv2. All six services were
verified against running executable hashes. A stopped-service matched backup
was taken before migration; sources, settings, rotations and Residential policy
were preserved, and the external revocation journal was not rewound.

Full race/vet/build/Python/JavaScript checks,11 real core cases, the user's supplied
config reproduction, actual DB migration rehearsal, controlled traffic and all
four worker integrations passed. Normal, Splash and Ads Smart recovery fixtures
covered55 profiles per scope. Synthetic browser passed after replacing a racy evaluate with navigation-aware
waitForFunction for automatic recovery. The earlier scheduling timeout under
load and navigation-race failure receipts are retained.
The system Go race runtime was corrupt; a checksum-verified official isolated
Go1.27.1 toolchain completed the full suite without editing shared toolchains.

Private receipts: .local/p32-deployed.json, p32-tested-runtime.json,
p32-live-check.json, p32-live-browser.json and p32-published.json. Live single
outputs were checked as served bytes on26.11.9 when ready. The browser check
ignored the existing panel TLS certificate; it does not certify public TLS.
Android APK/carriers/real upstream QUIC remain not_run. Pending/rejected raw
inputs are retained; a rejected plaintext endpoint has not been made encrypted
or connectable by this client-side change. Refresh/reimport downloaded profiles.
