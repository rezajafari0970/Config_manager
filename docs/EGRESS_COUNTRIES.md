# P54: observed Ads country consensus and any-country publication policy

Schema 29 supplements route policy ads-egress.v5. The user's new country request
supersedes the old prohibition on country lookup, but does not restore the old
ISP/Residential classifier. Residential remains the project's finite split-route
label. Country is a geolocation estimate for the exact IP reported by the same
authenticated Ads TLS peer; never infer it from entry IP, SNI, ASN, latency, DNS,
ordinary control-IP echoes or a model. The three-domain/TCP scope and diagnostic
limitations in EGRESS.md still apply. Neither the same-stream diagnostic nor
GeoIP is a guarantee of all ad traffic or the physical country of every request.

## Modules and contract

| Module | Responsibility |
| --- | --- |
| scripts/egress-country.py | Versioned provider registry, SQLite range and optional MMDB drivers, independent lineage consensus and fresh country facts |
| scripts/egress-country-update.py | Verified official DB-IP Lite download, bounded import, immutable SHA256 dataset and atomic manifest |
| scripts/egress-country-providers-update.py | Independent DB-IP/IPLocate updates, official Git LFS digest verification, bounded streaming CIDR import and last-good files |
| internal/egress/country.go | Independent validation of countries for already-qualified Ads observations |
| internal/panel/egress_country.go | Schema migration, policy compare-and-swap, country history, eligibility and counts |
| internal/panel/web/egress-countries.js | Search, counts, shared membership, persistent allowlist and unsaved-selection protection |
| deploy/config-repair-country-update.* | Separate daily updater with memory/time limits and least required write paths |

Lookup happens in trusted helper memory before the per-job HMAC scrubs plaintext
exit IPs. Persist only country, provider identity, immutable dataset SHA256,
release time, observation time and the existing job-local source token. No credentials are needed. If the two local providers disagree or lack data,
only the transient public Ads address is queried over verified HTTPS to GeoJS.
GeoJS uses MaxMind GeoLite and counts as the MaxMind lineage, not a fourth source.
The bounded batch has a3s timeout,30s failure backoff and4096-entry process-local
HMAC cache. No plaintext queried addresses are persisted or logged. IPv4 and IPv6 range keys are
fixed-length packed addresses; lookup is indexed. The existing lab limitation
for IPv6-only entry servers is unchanged by supporting IPv6 geolocation.

/etc/config-repair/egress-country.json is root-only and versioned ads-country.v2.
It holds one to eight local provider descriptors (name, independent lineage,
driver, immutable path, SHA256 and original release time) plus explicit geojs=true.
SQLite ranges and optional authorized MMDB are the only local drivers. Every
observation needs at least two independent agreeing lineages and a strict majority;
a tie is unknown. Multiple mirrors of one lineage cannot manufacture votes.
Provider loading is isolated: a bad replacement retains that provider's last-good
reader until its original freshness expires. One failed source cannot close other
readers. Manifest failure also preserves existing readers without renewing age.
Database age is limited to45days. GeoJS facts use checked_at and a24h maximum,
not an invented database release date. Live response digests contain no raw IP.
The Go evaluator independently validates consensus, lineage and evidence ages.

Default data is the official DB-IP Lite monthly IP-to-country CSV, licensed
CC BY4.0. The panel visibly links https://db-ip.com/ for attribution. The updater
uses verified HTTPS, checks the publisher's raw CSV SHA1, validates address-family
bounds, ordering and non-overlap, builds indexed SQLite and pins it with SHA256.
The manifest switches atomically only after validation; failure retains the last
good dataset, but expired evidence never publishes. Current and one previous immutable dataset per provider remain on disk; open
immutable readers retain file descriptors during atomic replacement. No database is committed to the repository.

IPLocate's free country feed is obtained from its official GitHub repository,
pinned to the file's commit and Git LFS SHA256/size. ZIP size, CSV header, CIDR
bounds, record count, non-overlap and SQLite integrity are checked before atomic
publication. Its source-data license is CC BY-SA4.0; attribution is visible in the
panel. Any redistribution of an adapted database must preserve that license.
The daily provider updater isolates download failures and retains last-good files.
More vendors are not automatically better: registered-country/ASN databases are
not physical geolocation votes, and duplicated underlying data is not independent.

Official provider documentation:
- https://db-ip.com/db/download/ip-to-country-lite
- https://db-ip.com/db/format/ip-to-country-lite/csv.html
- https://www.iplocate.io/docs/data-feeds/free-ip-to-country-database
- https://github.com/iplocate/ip-address-databases
- https://www.geojs.io/docs/v1/endpoints/country/
- https://www.geojs.io/
- https://support.maxmind.com/knowledge-base/articles/maxmind-geolocation-accuracy

## Allowlist and shared configurations

All250 recognized country/territory codes (ISO alpha-2 plus XK) are explicitly
selected initially. This is a persisted list, not an implicit wildcard. An empty
selection publishes nothing. API: POST /api/egress/countries with revision and
countries; existing same-origin/CSRF enforcement applies. Concurrent stale writes
return409, invalid/duplicate codes422. The save response includes the publication
count from the same writer transaction; the UI displays this count explicitly. Changes survive process restart.

A configuration is exported only if route evidence is currently valid, every
valid Ads observation has independent lineage consensus and fresh country facts, and AT LEAST ONE country seen
for it during the last24h is selected. This is set intersection (ANY), superseding
the previous ALL/subset policy by explicit user instruction. TW+MX+CY qualifies
with TW alone, CY alone or TW+CY, even when MX is unselected. MX alone also qualifies;
an unrelated selection or an empty selection does not. Missing/unknown country
cannot bypass the filter. Countries from accepted full reports accumulate for24h.
Filtering selects whole configurations; it does not constrain their actual exit
to a selected country. All observed countries stay visible. Schema29 changes only
the publication view, retaining selection, revision, evidence and queue state.
No finite observation can guarantee an unseen or future exit country.

Counts are distinct raw configuration hashes per country before the country
selection, restricted to currently route-and-geo-qualified members. One shared
configuration appears once in each observed country; totals therefore can exceed
the unique count. Shared, known, published and country-unknown counts are separate.
The ordinary/Smart/Splash pipelines retain their independent qualification rules.
This allowlist governs the four Residential Ads links only.

The default accepted page and configs.json, configs.jsonl, next.json and raw.txt
use egress_export_eligible. Existing egress_eligible remains the route-health and
retention contract. Filtering does not delete healthy retained configurations.
All exports recheck policy revision, observed-country set and eligibility in the
writer transaction before releasing bytes. Exact config bytes/formats and stable
subscription URLs are preserved. Proof-only TTL renewal is excluded from the
semantic generation; current expiry is still checked. Two bounded retries handle
concurrent generation/rotation changes; explicitly pinned stale generations or
persistent conflicts remain409. Empty/expired/filtered pools return503, no fallback.

## Retention, verification and operations

Validated staged inputs and committed memberships both register the raw hash in
one durable queue, with owner provenance. Revived age-deleted inputs reset stale
assessment counters, leases and evidence; manual exclusion fences still win.
Ordinary retention never removes unresolved or previously accepted egress input.
Raw bytes remain encrypted and referenced by the queue. This is forward durability,
not a claim to reconstruct bytes garbage-collected before this migration.

See EGRESS_LIFECYCLE.md for the independent payload contract, editable15-minute
health schedule, evidence expiry, uncertainty retention and confirmed-unhealthy
revocation. Country filtering never deletes a healthy stored configuration.

Run scripts/check.sh, scripts/egress-worker-test.py with existing browser dependency
environment, a private TestEgressPrivateMigrationRehearsal, exact live canary and
1000-input egress-throughput-test.py. Fixtures are isolated and never qualify real
production inputs. Country tests cover all output formats and HEAD, TR/MO and TW/MX/CY ANY
logic, schema28-to29 migration without policy/evidence loss, source/provider conflicts, missing/stale evidence, empty policy, revision
conflicts, unchanged proof generations, and archived deep retries. Browser tests
cover390/1440px, saved policy, page reload, unsaved selection and all four empty
outputs. Python tests cover both families, boundaries, corrupt datasets, provider
disagreement, malformed range imports and exclusion of unauthenticated/control IPs.

Hold /var/lock/config-repair-deploy.lock, preserve a matching stopped DB/blob/key/
release backup, and never run pre29 binaries against schema29. Explicitly chmod the
release directory0755 after mkdir: a private operator umask otherwise prevents
the unprivileged panel service from traversing it. Keep country data/config0600
inside root-owned0700 directories. Preserve P47 Smartv14, SmartAdsv7, payload_v1,
ordinaryv9, strict TLS, all source controls and unrelated controllers.
