# P39: independent Ads egress replacement — 2026-11-11

Activated p39-ebd1c323ba6e0e2d4879, schema23. Full release checks passed,
including419.188s panel race suite and fresh two-slot SIGKILL recovery. All six
services are active and their running executable hashes match. The cutover seeded
724 raw inputs from33 sources; ongoing acquisitions are also registered. Initial
readback showed zero eligible members; never describe pending inputs as classified.


The latest user explicitly requested complete retirement and rebuilding of the
Residential feature. Legacy implementation, worker, UI, API handlers, queues,
classification policies, attestations and historical feature documents are removed.
Do not resurrect the old engine or treat previous MO/ASN observations as proof.

Read docs/EGRESS.md and docs/checkpoints/P39-egress-rebuild-20261111.md.
Schema23 replaces the old subsystem and preserves the published schema22 source
admission fix from c88a43f. Never run a pre23 binary against the new database.
Deployment must hold /var/lock/config-repair-deploy.lock and preserve a matching
stopped database/blob/key/binary backup. All unrelated services and source settings
must survive. Source branch remains go/v2rayng-2.3.6; never force-push.

Evidence is explicitly finite: authenticated Ads TLS peer reports an IP in an
undocumented diagnostic response on the same stream. A second HTTP authority may
not match the certificate, which is recorded. This is not a signed observation of
all real ad requests. Only explicit, fresh Proxycheck v3 Residential classification
can qualify that observed IP; never substitute ASN, country or normal IP echoes.
No promise of 100% coverage, all Google categories, UDP/QUIC, Android or carriers.
Accepted entries are retention-exempt but publication requires fresh proof. Unknown
quarantines; three spaced credible negatives plus outage guard revoke globally.
Keep immutable raw bytes, strict TLS, bounded netns sandboxing and core pins.

# Current P36 video and ingestion fixes — 2026-11-11


# Current P35 authorization and deployment — 2026-11-11

The latest user authorizes outbound-only fixed redirect compatibility, replacing
P33 local listener relays to preserve official native RealPing. No core/server
changes or invented encryption. Read docs/checkpoints/P35-template-realping-20261111.md.
Normalv7/Smartv11/Adsv4 and schema21 fence old final-byte evidence; source controls,
retention, deletion journal and rotations remain intact. All unique input builds
and up to3 live representatives per family qualify the template; individual healthy
publication still needs its own evidence. Raw never changes. Do not overwrite
parallel P34 acquisition work or deploy an older schema binary over schema21.
Official targets remain v2rayNG2.3.15, bundled26.10.31 and latest26.11.9.
Installed release p35-57a1e4adb432b7dcca5c is schema21. Exact runtime canaries,
full release checks and native legacy delay passed; APK/carriers not_run.
Read docs/reports/P35-report-fa.md for solved families and remaining limits.


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


# Current P33 user authorization — 2026-11-11

The latest user authorizes a tested loopback relay only for legacy plaintext VLESS
TCP/RAW with optional HTTP header. This is explicitly labelled NO TLS compatibility,
not added encryption. Build on current official client/core source, preserve actual
endpoint/credentials and exact raw bytes. Keep bounded endpoint-only relay routes,
loopback listeners, deny fallback, unique tags/ports and lifecycle tests in Smart.
This replaces conflicting P32 prohibitions for this driver only. Require actual
final-output core and connection evidence; core-only ordinary publication is no
longer sufficient. Preserve JSONL as an export and add client-native array subscriptions.
The current no-unintended-egress requirement supersedes inherited UseIPv4 socket
fallback: generated socket policy is ForceIPv4, while DNS queryStrategy remains
UseIPv4. Preserve exact final bytes; P21 post-validation proxy shuffle is disabled.
Read docs/checkpoints/P33-unified-client-compatibility-20261111.md.


# Current P32 override — 2026-11-10

Schema19 updates persisted admission-version views and fences old leases before
listeners start. Ordinaryv5/Smartv9/Adsv2 change admission, not wire templates,
leastLoad, UDP/QUIC, sources, TTLs, settings or rotation counters. Stale caches
stay hidden until new validation. Schema18 binary cannot open19; fix forward or
use a stopped-service matched recovery without rewinding the external journal.

The BEGIN_ARRAY log on single-object import is the app's attempted array parse
before single-object fallback, not the fatal core error. Keep individual JSON as
an object and existing full JSONL contract. No Android/carrier test claim.
Read docs/checkpoints/P32-client-compatibility-20261110.md and deployment receipts.


# Current P31 override — 2026-11-10

The latest user restores age-independent retention for accepted records, replacing
P29/P30 fixed-age treatment of those records only. Ordinary retention and health
cache remain panel-configurable10h. Untested inputs stay queued until their first
assessment, even after source/ordinary artifact expiry; manual deletion still wins.

Schema18 fences old policy evidence and leases, preserves sources/settings and
rotations. External authenticated deletion ledger must never rewind. Use a
matching stopped-service DB/blob/key/binary backup; schema17 binary cannot open18.
Read P31 checkpoint and actual deployment/publication receipts before claiming live.

Official Xray matrix remains latest26.11.9 then26.7.28/26.5.9 parallel fallback.
TUIC/Hysteria native driver uses checksum-pinned sing-box1.14.4 in the same strict
sandbox. It mounts only four fixed glibc dependencies read-only; no host directory.
Native common JSON is sing-box format with @shenvpn tag, not Xray import JSON.
Unknown native extensions/complex routes remain UNSUPPORTED. No TLS weakening,
credential edits, Direct fallback or protocol substitution. Ordinary/Smart target
and their envelopes remain unchanged. Native synthetic traffic fixtures use crn;
worker tests crt; production crr. Never cross-clean those namespaces.


# Current P30 override — 2026-11-10


# Current P29 override — 2026-11-10
