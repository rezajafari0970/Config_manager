# Architecture — P26

The independent Go service owns collection, encrypted content and SQLite publication. Four separate supervisors own ordinary validation, ordinary Smart validation, Splash Smart validation and Smart Ads validation. Protocol preparation and JSON composition are pure transformations; network evidence comes only from the pinned sandbox. No service depends on ChatGPT, the remote-management connector, or a browser session to keep running.

## Active module and extension boundaries

| Responsibility | Implementation / contract | Isolation |
| --- | --- | --- |
| Scheduling, source revisions and generations | collect.Engine / collect.Repository | Per-source context, lease, timeout, cancellation and retry |
| Acquisition transport | collect.FetchDriver, HTTPDriver | Per-job HTTP transport; public-address gate before literal-IP dialing; at most two dials with250ms stagger and3s attempt deadlines |
| Exact extraction | collect.FormatDriver / Registry | Compiled plugins; new IDs take priority; replacement of an existing ID preserves its position; immutable decoded payloads |
| Content diagnostics | collect.InspectionDriver / InspectionRegistry | Annotation only; panic becomes unknown, never connectivity proof |
| Structural candidates | repair.CandidateDriver / Registry | Offline proposals; immutable originals; later core/connection gates remain mandatory |
| Shared plugin catalog | driver.Registry[T] | Synchronized registration; copied ordered snapshots; no registry lock during callbacks |
| Protocol adapters | probe preparation/adapters | Preserve endpoints, credentials, TLS and dependencies; unsupported protocols fail explicitly |
| Ordinary JSON envelope | convert | Template v4; source-specific proof and ordinary export policy |
| Smart selection/composition | smart | Version v8; dynamic namespaces, both leastLoad pools, safe default and unchanged requested envelope |
| Published Smart cache | panel/smart_cache.go | Schema13; validated bytes and exact member provenance independent of in-progress jobs |
| HTTP features | panel.PanelModule + feature route files | Compiled route registration, Host/Origin/CSRF gates; worker API never on public TCP |
| Queue polling | worker.Queue / worker.RunSlot | Transport-independent, per-slot bounded idle/error backoff; context cancellation |
| Execution | config-probe-worker + trusted Python sandbox adapters | Protected Unix socket, pinned core, netns/cgroup/bwrap; independent resource prefixes |
| Content storage | blob.Vault | AEAD chunks, content digest, atomic replacement and publication pins |
| Subscription delivery | panel/export_spool.go | Exact JSONL/count/hash; bounded working RAM, immediately unlinked private temporary file |
| Browser features | web/app.js + feature modules + poll/settings-form drivers | Independent loops, bounded retries,20s deadlines, loading gates and preserved drafts |
| Database access | Serialized writer + three read-only WAL connections | Claims/mutations/rotation gates stay atomic; committed-state UI reads avoid writer queue |
| Plaintext transient files | internal/transient.NewFile | Unlinked0600 inode; reopenable /proc/self/fd path for Linux format plugins |

Compiled extension registration is supported for format, inspection, repair and panel modules. This is not arbitrary runtime plugin upload or a promise that any new protocol works without an adapter and acceptance tests. Legacy generic interfaces in driver.go describe evidence types; active transport/preparation contracts are the concrete contracts listed above.

## Splash publication repository

Each source has an optional Splash classification (existing sources default to ordinary). SmartPipeline uses one builder, selector, composer and HTTP module, bound to one of two closed repository drivers. The fixed SQL-identifier vocabulary maps only the Splash instance to splash_smart_* tables/views. No client-controlled SQL or arbitrary runtime plugin upload is accepted.

Ordinary and Splash have independent settings/revisions, pending heads and member provenance, validated cache, source/member usage counters, durable single rotation, worker status, private Unix routes, public token domains and per-scope locks. Services use disjoint csm and csp sandbox resources (cst/cpt in integration tests). Collection, immutable content-addressed blobs, conversion evidence, SQLite and the host remain shared resources. This is logical publication isolation, not separate hardware.

Admission first filters by current source classification and exact source-owned proof. If a canonical identity or candidate is owned by a Splash source, ordinary Smart excludes it even when duplicated by an ordinary source. Splash never fills capacity from ordinary sources. Ordinary template-v4 exports remain unchanged. Changing a source flag immediately revokes incompatible ready-cache membership and completion/renewal eligibility. Revision fencing cancels an obsolete acquisition lease without deleting retained bytes or extending TTL.

A fingerprint of the eligible pool lets a previously blocked composition retry immediately when membership changes; an unchanged pool retains backoff. The fingerprint excludes expiry timestamps to avoid retry storms on mere proof renewal.

## Source and publication lifecycle

Original response/config plaintext remains exact. Ciphertext may change when identical bytes are acquired again; a completed/fsynced atomic replacement repairs corruption at the same content address. Existing open readers retain their inode. Pins and database references protect publication from collection. Vault I/O remains outside database transactions. Never recursively acquire the serialized writer from a transaction. Explicit read-only methods use the WAL reader driver; no transaction or GC reference check was moved to a stale snapshot.

Accepted nonempty generations replace only that source's memberships. Empty/error results retain prior data until fixed TTL.304 and repeated acquisition do not renew TTL. Source disable/delete/edit, config controls and expiry invalidate current ownership; stale workers cannot resurrect it. Current source counts override historical acquisition receipts.

Ordinary outputs remain exact template-v4 candidates with pinned-core acceptance. Semantic repair hypotheses additionally need fresh HTTPS evidence. Smart admission additionally requires source-owned fresh https_multi_site_verified proof. No profile may borrow another source's proof, retain a revoked member, or extend artifact retention.

## Safe replacement instead of cache outages

smart_heads/smart_members hold queued or running work. smart_cache/smart_cache_members hold only completed, exact core-validated publication. CompleteSmart atomically replaces both cache bytes and provenance after checking the active lease, revision, hash, all current member proofs and deadlines.

smart_ready checks cache safety at every read independently of job status. More usable sources/capacity schedule an improvement through smart_refresh_needed; a still-valid cache remains available during that work and after a failed replacement. Lost ownership, failed/expired proof, changed settings/template, or cache TTL still hides it immediately. Missing/corrupt bytes revoke only the matching cache generation and queue rebuilding; a canceled or merely stale read does not cancel a healthy concurrent builder.

Both durable rotations retain their existing scope/cycle. Smart full/single delivery keeps per-request random order and independent-IP first; canonical individual bytes remain exact. Schema13 does not reset source/settings/rotation/ordinary data or alter Smart v8 JSON.

## Bounded recovery and resource use

The collector is deadline-bound and supervised. Workers use per-slot leases, heartbeats, startup cleanup/self-test and independent idle/error polling backoff up to2s. A killed worker recovers through lease expiry; active slots remain bounded. systemd supervises whole-process failure. These mechanisms recover transient faults; they cannot invent missing credentials or make unreachable upstreams reachable.

Queries use exact keyed ownership joins and indexed content references. UI polling requests only the current feature's payload; the old unscoped API remains compatible. Large full exports spool validated lines instead of keeping the full subscription in memory. Spool files are0600 and unlinked before writing, so normal close, cancellation and process death leave no named plaintext export. Storage failures occur before successful response headers.

SQLite, disk, CPU and networking remain shared host resources. Per-source isolation does not imply immunity to host or operator failure. A phone/operator receipt is required for Android/TUN, throughput, media playback, jitter or loss claims.

## Upgrade and rollback

Take a matching stopped-service DB/blob/key/release backup. Verify migration on an isolated database copy, including source/settings/rotation/ordinary-state fingerprints. Schema12 adds Splash tables and a source flag alongside the P22 validated cache. The tested pre-feature rollback drops only additive transient structures/columns, resets user_version to10, and starts the prior release to recreate its views. It preserves current source/settings/rotation data. Legacy rollback is refused once any source is Splash or its queue is populated: the older builder cannot enforce classification. Use a feature-aware release after Splash adoption; never silently expose Splash inputs to the old unpartitioned builder. A matching full backup remains mandatory.

Publish only tested source, read back its exact Git tree, and verify installed and running binary hashes independently. See the P22 checkpoint and private receipts for this release's measurements and limitations. A standalone checksum-pinned installer/update/rollback product remains a separate release task.


Final sandbox correction: the new csp/cpt prefixes are explicitly registered. Existing worker/test prefixes previously shared their IPv4 /30 addresses despite different resource names. The sandbox now reserves disjoint address blocks for all active workers and trusted fixture prefixes, retaining crw’s original block and exact cleanup names. Address disjointness and bounds have an unprivileged regression; all real worker/traffic tests must run against this final adapter.


Schema13 only replaces normal smart_inputs with equivalent non-null Splash identity/candidate reservation sets evaluated by the current query. The original correlated OR join caused repeated scans under an adopted Splash source and held the shared SQLite connection long enough for a full Splash export to exceed30s. This is a query-plan correction, not a relaxed admission/cache gate. The schema12 predecessor supports Splash adoption; a guarded13→12 rollback restores its exact view without changing table rows. The older10 downgrade remains forbidden after adoption.

## P24 reader and UI recovery

Store owns one write connection and at most three read-only connections. Open initializes migrations/key checks before creating the reader; both pools close with Store. mode=ro and query_only(1) prohibit accidental reader mutations. Every connection carries foreign_keys, busy_timeout and synchronous pragmas. SmartPipeline wraps either connection with the same closed scope mapper. Source lists/metadata, conversion summaries/exports and Smart presentation reads use this driver; claims, lease checks, writes and rotation consumption remain on the existing writer. WAL is concurrency isolation, not independent hardware or immunity to long CPU/disk pressure.

GET settings endpoints return small metadata independently of output readiness. Feature polling cannot be blocked by shared summary failure; retries are bounded and one request per loop can be in flight. Settings drivers preserve dirty forms and use revision-aware explicit resubmission for Smart conflicts. A disabled initial control is unknown/loading, not an asserted saved false. JSON viewers apply completed data only to the current matching-hash row.

Acquisition, nested envelope decoding, explicit restoration and full exports share an anonymous plaintext spool; persistent originals/candidates remain encrypted. Format plugins continue receiving a reopenable path. The Linux /proc requirement matches the deployed server and is exercised in tests.


## P25 receipt and retention recovery
Schema14 stores body receipt counters/time independently of current membership and output evidence. A new body measures extracted and unique occurrences plus newly inserted identities; presentation derives within-body repeats and pre-existing/shared identities without reusing the legacy combined duplicate count. Unknown legacy receipts are explicit. Whole JSON input identity remains distinct from its internal outbound count.

Before any expiration cascade, affected enabled sources are queued for fresh acquisition; next_run is capped by the earliest retained expiry. Startup reconciliation repairs already-swept gaps, while partial/missing inventory disables conditional validators. Failed recovery backs off5seconds to5minutes and never extends fixed lifetimes or clears user controls. The normal configured interval is unchanged; retention is an independent recovery trigger.

The state endpoint reads one WAL snapshot with a fixed presentation timestamp, preventing mixed generations/TTL moments within one response. Writer and export ownership gates remain unchanged. Schema14→13 rollback is additive and preserves Splash tables/settings; stop services before changing the schema marker and selecting P24. Re-upgrade discards unmaintained receipt measurements, not source data.


## Smart Ads (P26)

A third closed SmartPipeline repository driver owns ads_smart settings, jobs, validated cache, usage and durable turns. Its inputs are live immutable evidence from both existing pools, while position-specific gates require one whole Splash head and ordinary main followers. ComposeAds calls the common Smart composer, retains all namespaced outbounds and attaches terminal supported Ads dialers to a main leastLoad loopback. Nested balanced Ads inputs retain a distinct exit pool. The independent csa worker core-validates outputs before cache publication; source health, core acceptance and whole-chain connectivity remain separate evidence. The UI, public/canonical/full/single URLs and private Unix queue are isolated under the smart-ads scope. The same GC and lifecycle framework protects and revokes all three caches.

VLESS REALITY's positive support-x25519mlkem768 URI extension has a narrowly scoped native-Xray compatibility adapter. It creates no nonfunctional JSON property and cannot disable ML-KEM; unknown or incompatible options fail explicitly.
