# P48: observed Ads countries and publication policy

Schema 27 supplements route policy ads-egress.v5. The user's new country request
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
| scripts/egress-country.py | Versioned provider registry, SQLite range and optional MMDB drivers, fresh agreeing country facts |
| scripts/egress-country-update.py | Verified official DB-IP Lite download, bounded import, immutable SHA256 dataset and atomic manifest |
| internal/egress/country.go | Independent validation of countries for already-qualified Ads observations |
| internal/panel/egress_country.go | Schema migration, policy compare-and-swap, country history, eligibility and counts |
| internal/panel/web/egress-countries.js | Search, counts, shared membership, persistent allowlist and unsaved-selection protection |
| deploy/config-repair-country-update.* | Separate daily updater with memory/time limits and least required write paths |

Lookup happens in trusted helper memory before the per-job HMAC scrubs plaintext
exit IPs. Persist only country, provider identity, immutable dataset SHA256,
release time, observation time and the existing job-local source token. No keys,
credentials or per-IP network services are added. IPv4 and IPv6 range keys are
fixed-length packed addresses; lookup is indexed. The existing lab limitation
for IPv6-only entry servers is unchanged by supporting IPv6 geolocation.

/etc/config-repair/egress-country.json is root-only and versioned ads-country.v1.
It contains one to four providers with name, driver, absolute immutable path,
sha256 and released_at. The driver registry admits sqlite-ranges and mmdb only.
MMDB is optional and needs the separately installed maxminddb Python package plus
an authorized dataset; no licensed sample dataset is treated as global coverage.
Missing providers, bad pins, gaps, reserved/unknown countries, disagreements or
data older than45days produce unknown. Every configured provider must agree.
Provider errors have a bounded30s reload backoff and never expose IPs in messages.

Default data is the official DB-IP Lite monthly IP-to-country CSV, licensed
CC BY4.0. The panel visibly links https://db-ip.com/ for attribution. The updater
uses verified HTTPS, checks the publisher's raw CSV SHA1, validates address-family
bounds, ordering and non-overlap, builds indexed SQLite and pins it with SHA256.
The manifest switches atomically only after validation; failure retains the last
good dataset, but expired evidence never publishes. Old immutable files remain
available for in-flight readers. No database is committed to the repository.

Official provider documentation:
- https://db-ip.com/db/download/ip-to-country-lite
- https://db-ip.com/db/format/ip-to-country-lite/csv.html
- https://support.maxmind.com/knowledge-base/articles/maxmind-geolocation-accuracy

## Allowlist and shared configurations

All250 recognized country/territory codes (ISO alpha-2 plus XK) are explicitly
selected initially. This is a persisted list, not an implicit wildcard. An empty
selection publishes nothing. API: POST /api/egress/countries with revision and
countries; existing same-origin/CSRF enforcement applies. Concurrent stale writes
return409, invalid/duplicate codes422. Changes survive process restart.

A configuration is exported only if route evidence is currently valid, every
valid Ads observation has fresh agreeing country facts, and ALL countries seen
for it during the last24h are allowed. This is subset matching, not intersection.
TR+MO is withheld with TR alone. Missing/unknown country cannot bypass the filter.
Countries from accepted full reports accumulate for24h; a later single-country
round does not immediately erase a previously observed disallowed country.
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

Archived first-inconclusive results receive at most30minutes to finish one deep
attempt before ordinary age retention resumes. Active leases are protected.
Deep completion is persisted only on accepted completion, not on claim, so a
crashed claim cannot skip its retry. IPv6-only preflight failures retain their specific code (the old error-code
regex discarded the digit6) and stop redundant core fallback; this is an explicit
lab limitation, not a dead-config verdict. Manual deletion still wins; country metadata
is removed with revocation/expiry cleanup. Previously accepted route members keep
the established lifetime exception, quarantine, fresh-publication and loss rules.

Run scripts/check.sh, scripts/egress-worker-test.py with existing browser dependency
environment, a private TestEgressPrivateMigrationRehearsal, exact live canary and
1000-input egress-throughput-test.py. Fixtures are isolated and never qualify real
production inputs. Country tests cover all output formats and HEAD, TR/MO subset
logic, source/provider conflicts, missing/stale evidence, empty policy, revision
conflicts, unchanged proof generations, and archived deep retries. Browser tests
cover390/1440px, saved policy, page reload, unsaved selection and all four empty
outputs. Python tests cover both families, boundaries, corrupt datasets, provider
disagreement, malformed range imports and exclusion of unauthenticated/control IPs.

Hold /var/lock/config-repair-deploy.lock, preserve a matching stopped DB/blob/key/
release backup, and never run schema26 binaries against27. Explicitly chmod the
release directory0755 after mkdir: a private operator umask otherwise prevents
the unprivileged panel service from traversing it. Keep country data/config0600
inside root-owned0700 directories. Preserve P47 Smartv14, SmartAdsv7, payload_v1,
ordinaryv9, strict TLS, all source controls and unrelated controllers.
