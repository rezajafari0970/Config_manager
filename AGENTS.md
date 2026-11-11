# P45 activated throughput update — 2026-11-12

Activated p45-535993edea4187d12b9c; schema25 and ads-egress.v5 retained.
Read docs/reports/P45-report-fa.md and P45-validation.json. The latest user target
is1000 first assessments within600s, superseding300s below. Real production queue
measurement completed1000 distinct pending first assessments in354.586s;
393 were conclusive. UNKNOWN/UNSUPPORTED must never be sold as conclusive.
The default is min(31,2*CPU+2), with80% initial and20% follow-up lanes, bounded
memory and idle stealing. Urgent accepted expiry/manual checks retain priority.
Deep sampling uses two paired waves plus selective third retries; minimum two
paired observations per domain/variant, strict TLS, exact raw, pins and privacy
remain. Flat plaintext VLESS dispatch selects its compatible core directly.
Six additive metadata indexes prevent repeated archive scans. Do not reset v5
proofs or remove these indexes. Full race/release, migration, crash recovery,
reference split canaries and mobile/desktop checks passed. All four exports
and six running binary hashes were verified. Keep the shared deployment lock,
matched stopped DB/blob/key/release backup and no-force-push requirements.

# P44 activated user override — 2026-11-11

Activated p44-98fd9205fb5e28243764, schema25, ads-egress.v5. Read
docs/reports/P44-report-fa.md and P44-validation.json. Full checks, migration,
crash recovery and mobile/desktop browser passed; six running hashes match.
The real1000-input run assessed497 in300s and finished in736.832s; the requested
1000/300s target was NOT met. A fresh isolated Fastly canary qualified the new
project definition in23.142s; it is not a claim of ISP class or a new panel input.
The current Alan splash input also qualified in production and appeared in the
accepted-only page; all four exports returned200. Zero current memberships were
missing queue registration at verification. Older unresponsive Alan hashes are
not the current rotating-source input.

The latest user explicitly defines Residential as repeated dedicated Google Ads
egress compared with control sites. This supersedes ALL older provider/network
class requirements below. Read docs/EGRESS.md (v5). Never infer household/ISP class
from this product label. Remove classifier dependence, keep unknown distinct from
NO_SPLIT, and accept only repeated paired same-family Ads/control disjointness
in raw and template. Store job-local HMAC source tokens rather than plaintext
exit IPs; no ASN/country lookups. Two fast waves, three deep waves, minimum two
paired observations per domain and variant. Preserve strict TLS, exact raw,
source settings, retention exceptions, deletion fences and isolation. Benchmark
1000 distinct real inputs and report actual300s performance honestly. Do not
count UNKNOWN/UNSUPPORTED as conclusive results. Deployment still requires the
shared lock and matching stopped DB/blob/key/release backup; no force-push.

# P43: Alan Ads route assessment correction — 2026-11-11

Activated p43-80df8988c7565dcc30c7, schema25, ads-egress.v4. Preserve P42 strict
TLS, P40 repairs and all source controls. Read docs/reports/P43-report-fa.md,
P43-evidence.json, P43-validation.json and docs/EGRESS.md. Alan splash has measured
Ads/control split, but the sampled pool is mixed:3 Residential and7 Business
among10 exact-IP classifications; other new rotating IPs are unknown. Never turn
that into a wholly Residential configuration or infer class from country/ASN.

The dedicated egress-preflight permits only named-HTTPS-scope resolver exceptions
for inspection, executes original bytes and retains the endpoint-only firewall.
Source direct DNS remains blocked in the lab; no DNS leak qualification. Shared
ordinary/Smart preflight and main worker are unchanged. The Cloudflare control
uses its fixed hostname, bounded cached A resolution, SOCKS IPv4 and strict TLS;
never add a control IP to the direct firewall. IPv6 pairing remains independent.
XHTTP/deep waves are sequential with bounded parallel destinations. Current
first attempts and one deep retry precede retained backlog; partial first attempts
retry after30s. Manual rechecks precede current first attempts. Accepted expiry
priority, exact proof, retention, three-negative revocation and exclusions remain.
Partial evidence and source names are visible without eligibility. Optional
Proxycheck key is root-only in /etc/config-repair/egress-intel.json. No configured
key exists; unregistered100/day quota is exhausted, and the counter was not reset.
Post-activation the current Alan entry became unreachable from the test host:
production and isolated probes timed out, host TCP timed out and SYN-SENT was
observed. Current UNKNOWN is honest; prior split/mixed evidence is historical.
The1000/5-minute target is not a qualification claim. Hold the existing deploy
lock and matched DB/blob/key/release backup; no pre25 binary or force-push.

# P41 activated after P42 — 2026-11-11

Activated p41-7d775a88f934ae9ae0be, schema25, egress v3; preserve all
P42 strict-TLS and P40 parser/source fixes. Read docs/reports/P41-report-fa.md,
P41-validation.json and docs/EGRESS.md. All six executable hashes match.
The1000-real-input isolated run completed in749.939s, with375 initial
assessments in300s. The1000/5-minute target was NOT met. Unknown and unsupported
are not definitive classifications. Eight slots were more reliable than31 on
this shared four-CPU server; default is min(31,2*CPU), with bounded memory.
Persistent trusted helpers/netns always kill cores, deny egress and erase private
runtime files between jobs. Shared driver defaults and main worker binary remain.
Never label split egress alone Residential. Exact-IP classifier quota is exhausted;
fresh explicit class and full evidence still gate all four Residential exports.
The supplied Fastly config again showed ADS_SPLIT, not eligible Residential proof.
Accepted retention, outage guard, revocation, raw bytes, pins and source controls
remain. Hold the common deploy lock and matching stopped DB/blob/key/release
backup; never run pre25 binaries or rewind external deletion evidence.

# P42: strict TLS and safe disk reclamation — 2026-11-11

Activated p42-aa3602141aca9cc15b1b, schema25. Normalv9 / Smartv13 / Adsv6;
probev7 / client-compatv4. Read docs/reports/P42-report-fa.md and P42-validation.json.
Only strict TLS candidates are generated; immutable raw inputs remain unchanged.
Preserve server fields and pins, including XHTTP download streams. Never restore
allowInsecure=true to generated candidates to work around an invalid certificate.
All ten former VMess TCP TLS bypass candidates build on both pinned cores after
repair; individual connection proof remains mandatory. Full release/race/migration
checks and six canonical executable comparisons passed. Source controls survive.

Do not run pre25 binaries against this database. Deployment holds the common lock
and keeps a matching stopped DB/blob/key/release recovery set. Current and previous
verified offhost DigitalOcean recovery generations and quarantined DB evidence
were retained. Old logs/caches/backups were removed only after activity/reference
checks. Never sweep active worktrees or install caches blindly. Preserve concurrent
P41 work in its isolated branch; integrate this schema/version change before deploy.
Two pre-existing Bad message files in an old DO repair worktree were not touched.
No Android/carrier, all-provider-transaction or whole-disk-integrity claim.

# P40: conservative shared repairs and multi-core audit — 2026-11-11

Activated p40-36c40d9853bcde96cbe2, schema24. Normalv8 / Smartv12 / Adsv5;
probe v6 / share-compat v2 / client-compat v3. All six services run the tested
executables. Preserve the P39 egress replacement and all source-admission controls.
Do not run pre24 binaries against this database. Deployment must hold the existing
lock and retain a matching stopped DB/blob/key/binary backup. Never force-push.

Read docs/reports/P40-report-fa.md, P40-validation.json and the P40 checkpoint.
The immutable snapshot covers732 current inputs from33 sources plus10 historical
or user examples.721 inputs have accepted candidates on26.11.9/26.10.31/26.7.28;
21 remain preparation-blocked (19 missing REALITY key, one missing port, one bare ty).
Twelve previously blocked inputs are repaired, including six of the prior17.
There are zero FINAL input core rejections in this snapshot; ten ORIGINAL VMess TLS
candidate variants are rejected for allowInsecure, with strict alternatives accepted.
This is not a claim that721 endpoints connect.32 ordinary and17 Smart representatives
were network-tested; each production output still requires its own exact-byte proof.

Source-derived v2rayNG2.3.6/2.3.15 Import/Custom transformations were checked.
Actual Android APK and Iranian carriers were not run. IPv6-only runtime tests are
pending because the isolated lab is IPv4-only; do not mark those servers broken.
All four active normal/Smart/Splash/Ads exact-byte canaries passed both cores
and real bundled HTTPS/DoH/DNS TCP+UDP/QUIC VN/HTTP3. Splash/Ads initially
returned503 during requalification and then independently became ready. Preserve
that honest gate; do not publish stale or core-only profiles.

Repairs preserve exact raw data and server meaning: fragment/percent VMess decode,
explicit cipher alias, exact empty placeholders, numeric port spelling, bounded
REALITY metadata and complete IPv6 literal brackets. Missing data is never guessed.
Unknown active extensions stay explicit. Legacy plaintext compatibility adds NO
encryption. Version changes fence prior proofs and schedule independent revalidation.
Private snapshots, endpoint/credential reports and receipts remain in .local.

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
