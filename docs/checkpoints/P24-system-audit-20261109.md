# P24 — video-led end-to-end reliability and capacity audit

Parent:1578810b04fa442640297d1f29293c0ad4c1d511. P23 is deployed, published and read back; schema13, Smartv8, normal templatev4. All four services are enabled and active. User now requests a complete project audit, improved speed/independence/modularity/compiled driver-plugin architecture, automatic recovery, config fidelity and use of eligible capacity. The supplied110-second screen recording is evidence; no blanket zero-defect or highest-possible-speed claim is justified.

## Observations and investigation queue

- Video: delayed empty counters/export URLs on Input/Output/Smart, empty and apparently unchecked settings while loading, then partial settings arrival. Inspect whether feature refresh waits for unrelated shared state; add truthful loading/error states and protect unsaved changes.
- Video: several sources show expired/empty/small retained data despite recent HTTP200 or many extracted configs. Separate fixed artifact TTL, source ownership, duplicate identity, deletion suppression, conversion errors and fresh connectivity proof before changing admission. Preserve immutable source bytes and exact connection parameters.
- Investigate single shared SQLite connection/read-write contention across stages, bounded retries/cancellation, worker claims/heartbeats, GC/repair and process supervision.
- Inventory self-referential panel subscription inputs and possible feedback/amplification, without printing protected URLs/tokens or silently deleting/marking user sources.
- Audit protocol/transport/dependency mapping, source extraction capacity, Smart pool fairness and disjoint Splash eligibility, API/cache/export parity, security boundaries and cold/restart behavior.
- Review each compiled driver/module boundary against a reproduced fault; avoid cosmetic rewrites or adding an unrequested external LLM dependency.

## Work rules

Record each issue with reproduction, affected stage, concrete patch, tests and timing/evidence. Preserve source settings/data, Splash separation and accepted TTL/security gates. Run relevant tests after changes, then required release checks; update PROJECT_STATE, HANDOFF and this checkpoint. Actual Android/carrier/media tests remain unmeasured without a connected device receipt.

The user requested the highest OpenAI/ChatGPT model. Official model documentation was checked; this session exposes no model-switch or direct OpenAI API execution/key tool. Do not claim a model switch or API run occurred. Continue available authorized analysis and testing.

## Reproduced defects and implementation

1. Shared-state gating prevented Output, Smart and settings from loading when unrelated summary work failed. Each feature now owns an independent bounded polling loop (no overlaps, hidden-page pause, BFCache resume and capped retry delay). Summary polling outside Input is5s, listing polling2s. Independent settings requests fetch only their own small metadata instead of100 conversion/Smart entries and readiness scans.
2. Empty unchecked editable settings and copy buttons on blank links misrepresented loading. Controls start disabled, forms show loading/recovery status, copy activates only after the owning API supplies a URL. The shared settings form driver preserves drafts across polling, updates clean forms, and recovers revision409 by refreshing the revision and requiring explicit resubmission.
3. Output JSON requests could update a detached DOM row after polling replaced it; both viewers now resolve the current row and require its candidate checksum to match. Shared JSON delivery has a20s deadline and validates SHA256/UTF-8. List results are fenced against page changes.
4. All presentation reads previously queued behind a single SQLite writer. A bounded three-connection mode=ro/query_only WAL driver serves selected committed-state reads. Writes, claims, rotation transactions, GC-reference checks and all final transactional gates retain the serialized writer. Both Smart repository scopes apply the same closed SQL mapping to readers. Per-connection pragmas survive reconnect; no schema/settings/rotation change.
5. Acquisition, nested Base64 and explicit restoration used named plaintext temporary files. A common Linux transient-file module unlinks the0600 file immediately and supplies a reopenable /proc/self/fd path to format drivers. Export spooling shares it. Exact-byte/independent-offset extraction and cleanup tests cover this path. Encrypted persistent blobs and immutable source bytes are unchanged.
6. Missing import routing was caught by the first browser run and corrected before deployment. A module-import HTTP regression now checks every local JavaScript import against the actual asset routes.

## Capacity and structural audit

A read-only production audit decrypted content only within the authorized host and emitted no raw endpoints, URLs or credentials. Some source items contain2–4 real proxies; sampled converted custom JSON preserved their counts (e.g.6 items→12 proxies and4 items→16). A count of1 is not proof that child outbounds were discarded. A later source snapshot included explicit missing REALITY keys, malformed ports/Base64, unsupported packet encoding/WireGuard options and unknown VMess/query fields, plus pending/failed/partial/site-unavailable connectivity. Unknown fields observed included VMess name/skip-cert-verify and URI x-sub. These are not silently stripped or treated as a successful conversion. Missing credentials and strict TLS failures cannot be repaired by inventing values or disabling checks. Current exact-origin self-subscription classification found none; the earlier source-name hypothesis was not confirmed.

Fixed artifact TTL, source schedules and fresh exact proof rules remain in force. Successful HTTP acquisition is not connection proof. The UI now displays eligible unique input count separately from requested profile size and ready cache count. No raw source, source setting, outbound credentials, Smart envelope, source/Splash classification or non-Smart export policy is intentionally changed.

## Validation in progress

Targeted WAL-isolation/reconnect/settings tests passed. Browser tests passed with a forced summary503, delayed conversion settings, disabled initial controls, preserved drafts,409 recovery, independent exports, DOM replacement and desktop/mobile/restart scenarios. Full race/vet/build/core checks passed once; final checks repeat after the last read-path/import regression addition. Traffic and all three isolated worker SIGKILL integrations are running. Private evidence uses .local/p24-*; deployment/publication remains pending until final acceptance.

## Recovery-test correction

Initial traffic run failed after the fixture interpreted three random successes as recovery. A diagnostic gate using only alive=false was also too strict: the exact pinned Xray5ca6f4b7 shouldSelectNode excludes fail/all>0.15 while alive stays true until every retained sample fails. The final fixture checks the real alive/delay/failure-ratio predicate, derives a finite deadline from the45s scheduler window plus timeout, and still requires24/24 successful post-failure requests and working A/TXT DNS. It does not change production JSON or relax success criteria. The final isolated receipt observed exclusion at24.61s and verified recovery at26.33s, with11 requests failing during detection. This is synthetic evidence, not a2-second or carrier guarantee. All traffic/core acceptance paths then passed. Failed diagnostic receipts are retained privately.

## Final acceptance and deployment

Final gofmt/module verification/vet/race/build, pinned core,13 Python sandbox checks,14 source-status assertions and API/polling regressions passed. Browser mobile/desktop/restart and fault-injection passed. Ordinary worker SIGKILL recovery completed5 synthetic inputs; normal and Splash each rebuilt55 profiles x50members across3 sources with at most2 active slots. Traffic/DNS/Lib40 checks passed with the corrected observer predicate.

Deployed /opt/config-repair/releases/p24-7f0e103e325c54015669-r1; panel SHA256 7f0e103e325c540156696112a0a48115a0da0226e9d8d1f8026828f66c91a878. All four services are enabled and active. Matching stopped-service backup: /var/backups/config-repair/pre-p24-20261109-022027. Schema13 unchanged; source-definition and all settings fingerprints matched before/after startup. Production quick_check/foreign_key_check passed. Current source count29, Splash-marked7, normal count50, Splash count1. These are observed user settings, not defaults or forced changes.

Public HTTPS checks passed at390/1440 widths: zero page errors/downloads, both full/single scopes,8 single requests, hashes, canonical parity and dynamic namespaces. The user changed Splash count to1 during this audit; such single-proxy profiles have no alternative proxy order. The verifier was corrected to distinguish that legitimate case without changing user settings; normal output showed108 object-order variants. Complete exports ranged68–3574ms as live profiles and proofs changed; do not claim uniform sub2s output.

An independent read-only audit rebuilt8 normal/Splash cached profiles from204 selected inputs: all858 copied outbounds and cached bytes matched the shared composer, logs were off, and both ownership partitions were clean. Settings metadata endpoints had seven-sample local-server medians1.14–1.51ms (max25.8ms); before/after listing data changed, so these are not carrier timings or a controlled universal speedup. Existing normal listing remained about320ms in the live sample.

Source publication is linked to this tested/deployed state and read back into the authorized checkout; exact commit/tree and clean-worktree/runtime hashes are recorded separately in .local/p24-published.json to avoid embedding a self-referential commit hash. Android/carrier/media/real QUIC and model-switch/OpenAI API execution remain not_run. Missing credentials, unsupported semantics and provider/network outages are explicit remaining limits, not fabricated successes.
