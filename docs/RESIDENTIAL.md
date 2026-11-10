# P31 — attributable Ads differences and native protocol drivers

This section supersedes older fixed-age and single-category notes below.
The health, Ads attribution, scoped route comparison and exact-IP network class
are independent. The route-comparison contract is `ads-route-paired.v1`.

`ADS_DIFFERENT` requires healthy core-validated raw and common variants, a single
fully understood proxy path, at least three fresh waves for a listed Ads domain /
IP family / transport, and valid administrator-trusted flow attestations. Every
wave has successful Cloudflare and ipify controls on matching variant/path/family/
transport. Their entire IP pool must remain one canonical IP; the Ads IP set must
be disjoint. Any rotated controls, cross-wave overlap, replayed connection/nonce,
missing paired coverage or unavailable attribution leaves that scope unknown.
Canonical IPv6 spelling compares equal; mapped IPv4 cannot claim an IPv6 cell.
This is an observed scoped difference, not proof that all future sessions or all
Google Ads use a dedicated path. ASN/country diagnostics do not decide acceptance.

`ACCEPTED` retains the strict Residential gate: a specialist and an independent
source must agree on the exact fresh observed IP, and every configured matrix
cell/path in raw and common form must qualify. Equal Ads/control IP is allowed.
Network Mobile/Datacenter/Unknown remain separate from route differences.

The Residential page presents two labelled datasets with three links each. Old
`/residential-subscription/` links still mean verified Residential within policy;
new `/ads-different-subscription/` links mean scoped different Ads egress only.
Each dataset's JSONL, durable rotating single JSON and exact original raw.txt use
one current membership/generation. Use `?generation=` to bind multiple requests;
changed generations409, empty/unavailable eligibility503, no stale substitute.
Raw bodies are byte-exact with one LF between members, not ZIP or a self-delimiting
container. Native templates are sing-box JSON with @shenvpn outbound tag; Xray
protocols keep the existing @shenvpn envelope. The formats are explicitly labelled.

Accepted members in either category are exempt from ordinary age cleanup, source
removal and blob GC. Evidence expiry still removes them from every active output.
First-assessment jobs also remain queued beyond ordinary TTL until actually
assessed; this does not grant publication or retain failed assessed inputs forever.
Duplicate acquisition never grants fresh evidence. Confirmed loss needs three
spaced owner-bound fresh votes; a shared failure opens the five-minute destructive
circuit from the first correlated vote, including when another worker stalls.
Route loss must cover every previously accepted differing scope; classification
outages/Datacenter metadata alone cannot delete a route-difference member.
Manual deletion remains global. Tombstones, record revisions, epochs and external
authenticated ledger fence stale jobs/backups; already downloaded bytes remain
outside server control.

Core plugins: official Xray26.11.9 first;26.7.28 and26.5.9 bounded parallel fallback.
One whole raw/template report wins. Native TUIC/Hysteria1 URI and simple native
TUIC/Hysteria1/2 JSON use official sing-box1.14.4 (closed SHA manifest). No conversion
between incompatible wire protocols. Native unknown extensions, chained/balanced
routes and unsupported IPv6 execution are explicit; hostname bootstrap is marked
incomplete. Its glibc loader and three libraries are mounted read-only individually
inside nobody/bwrap/netns/cgroup isolation. No Direct or host-DNS allowance is added.

Validation includes synthetic native TCP/UDP and upstream-failure fixtures, both
output pools, concurrency/reopen, age/source/GC, nonce/IP canonicalization, stale
completion, ambiguity/outage/circuit, schema18 replay and private production-clone
migration. These are not live Residential evidence, real Google QUIC, Android or
carrier tests. Without authorized egress evidence and specialist data, public
certification stays empty rather than adopting candidate IPs.

# P30 core compatibility and route-method correction

Residential execution uses scripts/residential-matrix.py and the closed official
registry scripts/residential-cores.json. Latest26.11.9 runs first. An incomplete
connection result triggers26.7.28 and26.5.9 in parallel, each in an independent
namespace/slot. Maximum two jobs and four simultaneous fallback cores. The owner
checks core hashes and all existing raw/template/revision/trust bindings; the
UI shows actual selected version and per-version core/health/error results.
Unsupported, unavailable, schema rejection and connection failure are distinct.
A successful older core does not certify a newer client. Normal and Smart output
validation retains its separate26.7.28 target.

One complete report wins; no observations are borrowed across versions. Driver
input is limited to8MiB, streamed stdout to4MiB, each attempt to105s, whole Go job
to4min. A verified parent PID and pre-read watcher prevent orphan work. SIGTERM
stops new attempts and reaps existing children; cleanup attempts every physical
slot with bounded timeouts. Cleanup failure stops reuse and triggers restart.

Screening starts with three Ads domains and two controls. A healthy first wave
expands to the complete explicit domain list. A seen IPv4 Ads candidate triggers
1.1.1.1/cdn-cgi/trace and api.ipify.org controls if two independent IPv4 controls
are absent. These are still separate control connections, not Ads-IP evidence.
Control failures get at most one fresh retry. Dual-stack controls, IPv4 controls,
certificate/hash/status/stage and actual-core provenance are recorded separately.

The second HTTP authority on the same TLS connection is checked against its SAN.
This check does not establish the semantics of an undocumented diagnostic page.
The IP from /sorry remains CANDIDATE_ONLY without verified origin/meaning and
trusted correlated proof. A3xx response is never followed. Only /robots.txt and
diagnostic/control paths are used; no ad impression or click is generated.

These additions do not supply forcedIPv6, UDP/QUIC, exhaustive balancer-path,
authorized server-flow or specialist IP-classification evidence. Partial scope
and unknown class remain ineligible for full-scope Residential publication.

# Residential Google Ads evidence — P29

## P29 lifetime and diagnostics update

The latest user instruction supersedes P28's unlimited retention: residential
records now have a fixed `retain_until`, based on first receipt and the shared
panel retention setting (production target 36,000 seconds). Re-fetch, successful
retest and increasing retention cannot extend an existing generation. Shortening
retention shortens its deadline. The ordinary health cache is separately set to
36,000 seconds. Ads evidence remains limited to 600 seconds and is rechecked at
300-second jittered intervals. Read-time expiry never waits for cleanup.

The raw output is `raw.txt`, `text/plain`, inline: each acquired original body is
unchanged, with one LF between records. It is not a self-delimiting serialization
for arbitrary multiline inputs. The old authenticated raw ZIP URL redirects to
the new stable text URL. JSONL and single JSON formats retain their contracts.

The diagnostic summary is independent of acceptance: repeated same-connection
candidate IPs are labelled CANDIDATE_ONLY unless the trusted method verifies them.
Comparisons require both independent controls in the same wave, family, transport
and raw/template variant. Equal IPs do not reject an all-residential configuration.
Different IPs, ASN or geography never grant publication. Fresh IPinfo and IPWhois
metadata are collected with exact-IP checks, bounded HTTPS requests, HTTP/1.1 ALPN,
TTL cache and rate-limit backoff. These metadata-only plugins always return Unknown
network class and cannot substitute for configured specialist classification.

The dashboard shows tested/pending counts, per-domain samples, rotation and
corroborated country/ASN differences, with pagination instead of hiding all but
100 records. Evidence records retain publisher/time/digest. No AI call runs per
connection, classifies an ASN or authorizes deletion.

## Contract and current capability

The evidence engine has three independent axes: connection health, observed Ads
egress, and the observed IP's network class. An IP/ASN/country difference is not
proof of residential access or dedicated Ads routing. Equal control/Ads egress
does not disqualify a residential configuration. Decisions describe finite
observations, never permanent behavior or all Google Ads.

Default policy `shen.residential.v1` uses explicit domain-list revision
`google-ads-explicit.2026-11-10.1`, ten domains, IPv4/IPv6, TCP/UDP/QUIC,
three samples, ten-minute evidence lifetime and five-minute jittered recheck.
Every matrix cell and selectable path must qualify in both the raw import (with recorded deterministic structural repairs only)
and common `@shenvpn` template. Policy, trust, template and raw
changes invalidate dependent evidence. Current Xray target remains 26.7.28.

**Current live limitations:** the built-in network driver screens TCP HTTPS,
initially three explicitly configured DoubleClick domains, using two independent control services and repeated
waves. No native IPv6-forced, UDP or QUIC proof driver is supplied. All unresolved
multi-outbound/balancer/geosite/reverse paths remain incomplete. Other core
drivers return `UNSUPPORTED`; reachability is not silently mapped to unhealthy.
There is no configured authorized egress observer or specialist IP provider on
the initial deployment. Consequently no live input may be asserted residential
on the basis of this implementation's experimental diagnostic. Empty public
outputs are the required safe result, not a fallback to ordinary Smart servers.

## Evidence chain

`internal/residential` contains versioned data contracts, policy, driver registry,
path inventory, signed observation verification and exact-IP classification.
`internal/panel/residential_*` owns leases, independent retention, tombstones,
publication, management and metrics. `config-residential-worker` supervises
bounded persistent worker slots; each assessment starts its own isolated core.
The Python core, observation and intelligence adapters are separate modules.

The experimental same-TLS method makes a TLS-verified safe `/robots.txt` request
to the real Ads hostname and then tries a diagnostic on that same stream.
It follows no redirect and creates no ad impression or click. A recognizable IP
and reflected nonce do **not** authenticate diagnostic semantics or origin:
`same-tls-candidate-v1` always remains `UNKNOWN`. The returned IP is labelled a
candidate, not proven user egress. A control connection's IP never becomes an
Ads IP. No traceroute, ping, DNS answer, certificate or destination IP is used
as user-egress evidence.

An optional authorized observer bridge accepts only
`server-flow-ed25519-v1`. The signed payload binds the audience, exact raw and
runtime hashes, policy, variant, domain, family, transport, selected path, time,
wave, nonce, connection ID, observed IP and response hash. The Go owner verifies
the signature against an operator-registered capability, its scope and expiry.
The observer must actually correlate server-side routing/outbound/egress-flow
evidence with the test. Merely returning a signed guess is not a valid method.
An operator must document how that correlation is established before trusting
the capability. The provided bridge does not create such access or telemetry.

Exact-IP facts require two independent configured groups, including a specialist,
with fresh explicit network classifications. Residential, Mobile, Datacenter
and Unknown are separate. Conflicting classification, country or ASN yields
Unknown. No ISP-name, non-hosting boolean, country or ASN allowlist exists.
Facts include exact IP, country, ASN, provider, acquisition/expiry times and
response digest. TLS provider responses are collected by the trusted supervisor;
a domain-separated HMAC authenticates its facts at the owner boundary. This
authenticates collection provenance, not the provider's inherent accuracy.
The sandbox and API review model cannot access that key. Provider groups and
specialist capabilities are operator trust assertions that need independent
documentation, not self-reported labels from a remote response.

## Configuring optional private adapters

`SHEN_RESIDENTIAL_CONFIG` may name an administrator-controlled JSON file containing
`policy` and `trust` matching the Go contracts. No public endpoint can register
proof capabilities. Use the exact structs in `internal/residential/types.go`.
Changing these values invalidates current proofs and in-flight jobs.

`/etc/config-repair/residential-observer.json` is a private file with version
`shen.residential.observer.v1`, an HTTPS `url` without URL credentials/query,
and optional absolute `token_file`. The bridge sends only public test metadata,
hashes and nonces, never raw configurations or credentials. It accepts only
scope-identical, signed receipts. No redirects or environment proxy are used.

`/etc/config-repair/residential-intel.json` has version
`shen.residential.intel.v1` and a bounded `providers` list. Each provider uses
`driver: https-json-v1`, a stable `id`, HTTPS `url` containing `{ip}`, documented
`fields` for echoed `ip`, explicit `class`, `country`, `asn`, and a `classes` enum
mapping to Residential/Mobile/Datacenter. An optional private `token_file` supplies
an Authorization header. Mapping a non-hosting boolean to Residential is invalid.
The default cache path is `/var/lib/config-repair-residential-intel/cache.json`.
Use mode 0600 for private adapter files/tokens. Nothing is preconfigured with
invented providers, keys, model-generated classifications or network facts.

Intelligence caches are keyed by exact IP and provider/configuration digest,
bounded and expiring, with per-provider concurrency, coalescing, limited retries
and circuit breakers. Their reuse never replaces a new path observation.
Core network namespaces admit only the selected proxy endpoints and loopback;
no host DNS or Direct alternative is enabled for the tested configuration.
Independent infrastructure controls deliberately run outside that tunnel and
are labelled separately; their result cannot stand in for a failed tunnel.

## Retention, quarantine and deletion

Schema 16 creates residential raw-hash records and independent encrypted vault
references without source/artifact foreign keys. All input memberships enter
the queue, including Splash, ordinary sources and unsupported protocols. Original
bytes remain immutable. Accepted originals and templates survive ordinary source
deletion/disappearance and garbage collection until their fixed shared retention
deadline. Duplicate fetching does not extend it.
Unknown/partial inputs remain available for re-evaluation. Initial unknown audit
history is limited to two results; previously accepted history to twenty.

Evidence has a shorter independent lifetime than the fixed configuration deadline.
Every export checks present eligibility; expired, suspect or uncertain formerly
accepted configurations are quarantined across ordinary and Smart outputs.
An IP-provider outage, conflicting data, a single timeout or shared infrastructure
failure cannot authorize deletion. Three fresh spaced credible failures are
required, with fresh independent infrastructure controls for connection failures.
Rechecks prefer another local worker slot. Correlated failures across retained
members trip a destructive-action circuit breaker while publication stays closed.

Confirmed revocation removes operational memberships, derived results, dependent
Smart caches/heads, source responses, queued turns and independent blob references.
Encrypted unreferenced bytes are removed by the existing garbage collector.
Audit/tombstone metadata remains to prevent resurrection. Manual deletion is
permanent until an explicit future restore mechanism is implemented; re-fetching
or a new label cannot bypass its identity tombstone. Automatic re-entry requires
a new lease/epoch and genuinely post-revocation observations. Older completions
cannot publish. Deleting a source alone does not delete retained residential data before its deadline.

The append-only authenticated deletion ledger and high-water mark must live
**outside the ordinary DB/blob backup tree**. Append+fsync precedes DB commit;
startup replays pending valid events. Missing/truncated committed history blocks
publication. A partial uncommitted tail is repairable only beyond an intact
authenticated watermark. Startup invalidation happens before public and private
listeners; restored proofs must be observed again. Restore the DB/blob/key set
without rolling back the external ledger. Do not downgrade to a schema-15 binary
or restore an old runtime that does not enforce the ledger. P28 rejects schema17.
Previously admitted responses and downloaded files cannot be recalled.

## Outputs and UI

`/residential` and `/api/residential` show actual eligibility count, independent
axes, explicit scope, reasons, evidence history, worker state, measured latency,
throughput and CPU/RAM metrics. They expose stable signed URLs for:

| Format | Bytes and rotation |
| --- | --- |
| `configs.jsonl` | One compact common-template JSON per line; no array/wrapper |
| `next.json` | One JSON per GET; durable independent round per link, shared by consumers |
| `raw.txt` | Exact original bodies, one LF separator; inline plain text |

raw.txt preserves each original body; one LF separates members, so arbitrary multiline input is not self-delimiting.
All three derive from the same eligible snapshot and generation. Supply the
`generation` query parameter to reject changes during a three-output comparison.
No members yields HTTP503 and Retry-After, never stale or datacenter fallback.
HEAD/prefetch do not consume turns. Allocation is unique within a round; delivery
after disconnect is not exactly-once. New members join the next round.

Encrypted blob reads and disk spooling happen outside DB write transactions;
final admission atomically checks current membership, versions, validity and
ledger. Legacy raw/response/JSONL/Smart paths have an additional common fence,
including conditional 304 responses. Admission is the publication linearization
point; the transaction is not held while a slow client receives bytes.

## Validation and deployment

Read `docs/checkpoints/P28-residential-20261110.md` for actual test receipts and
limitations. Synthetic signed/provider fixtures are never live residential proof.
`scripts/check.sh` includes race tests and driver/parser checks. The new
`scripts/residential-worker-test.py` uses disjoint `crt` resources, bounded live
read-only canaries and SIGKILL recovery; production uses `crr`. Do not run a
production cleanup from a test. Core hash and source/runtime manifests remain
pinned. AI reviews are offline, redacted source reviews with recorded model,
tokens and estimated costs; the detector and exports require no OpenAI API.
