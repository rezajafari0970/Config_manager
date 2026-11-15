> Historical release description. The sole current policy is [RESIDENTIAL_POLICY.md](RESIDENTIAL_POLICY.md).

# Named Ads routing profiles (v8)

Residential is this project's label for a verified destination-specific Ads split.
A complete DoubleClick profile qualifies even when other Ads destinations use the
ordinary path. It does not mean a household ISP or that every Google Ads domain,
transport, country or future request uses one dedicated physical proxy.

## Compiled policy

`internal/egress/profiles.json` defines two independently evaluated profiles:

| Profile | Required members |
| --- | --- |
| `doubleclick.v1` | `googleads.g.doubleclick.net`, `pubads.g.doubleclick.net`, `securepubads.g.doubleclick.net` |
| `ads-serving.v1` | `pagead2.googlesyndication.com`, `adservice.google.com`, `www.googleadservices.com` |

Every member must independently qualify in both exact raw and output-template
runtimes. Profiles cannot be assembled from different members or variants.
The profile/target registry digest is part of the persisted implementation marker
and decision. An API response cannot supply policy or classify a live connection.

## Evidence and decision

The shared route-map engine validates pinned core/config/runtime bindings, strict
TLS and same-stream diagnostic responses, bounded timestamps and exchanges, address
family, wave and unique nonce. The driver keeps a job-local HMAC equality namespace,
canonicalizes source IPs, and serializes each attempt once. Every replayed copy is
invalidated, including copies in another attempt.

For each separated Ads destination, require at least two waves in which the same
valid observation has a completed nonempty request/response exchange and stable,
same-family, same-wave witnesses from at least two control groups. One group must
be non-Ads Google. `www.google.com` and `ipv4.google.com` belong to the same group.
Reachability, a timeout or a local blackhole is not source-address evidence.
Rotating Ads source addresses are allowed. Cross-wave control-pool collisions
prevent that destination from qualifying; separate and shared waves remain mixed.

Any complete qualifying profile yields `RESIDENTIAL`. Shared/unobservable domains
outside it remain visible and do not veto it. Only repeated, noncontradictory
sharing for every tested profile member in both variants yields terminal
`NO_SPLIT`. Everything else stays `UNKNOWN`, with bounded follow-up.

Diagnostics label finite `PARTIAL` versus `ALL_TESTED` Ads coverage and expose each
profile and destination. They never convert these labels into universal coverage.
The Google peer's undocumented source diagnostic is an observation on the measured
TLS stream, not a signed statement about every real ad request. Destination IP,
ASN ownership and general IP-check sites are not substituted for Ads source proof.

## Country, safety and publication

A complete country's evidence is required for all observed sources of each located
profile. A second qualified profile with unavailable country facts cannot suppress
a fully located profile. Publication applies the existing ANY selected-country
policy to complete profile-country observations, retaining its 24-hour window.
Changing the set of located profiles clears the derived rolling country cache;
temporary route/GeoIP uncertainty retains history but withdraws current eligibility.

Iran consensus independently scans all recognized Ads, including shared domains
and other attempts. Minority Iran conflicts also block publication without being
promoted to confirmed guard hits. Existing independent-round confirmation,
quarantine, revocation journal, aliases and manual exclusions remain authoritative.
Unknown is never manufactured into a clean Iran result.

Health is the completed Ads application exchange on the qualified profile, not a
bulk speed claim. General upload/download tests remain diagnostics. Raw bytes and
exact tested template bytes are preserved in all four formats; profile annotations
are panel/API metadata and response headers, not untested configuration edits.
One identity publishes once, even when both profiles qualify. Counts and exports
use the existing authoritative eligibility snapshot and revalidation gates.

## Lifecycle and migration

Schema 35 fences older binaries. A version change invalidates old publication and
in-flight leases; legacy all-or-nothing `NO_SPLIT` conclusions require new v8 proof.
Unaffected unsupported-driver conclusions and deleted entries remain unchanged.
Reports, decisions, attempt/first-completion history, metrics/reset clock, sources,
manual exclusions and sticky Iran guards survive. This is not another statistics
reset. `tested=0` means a current-version proof is pending; historical receipt time
remains in the retained report, while lifetime first-completion stays unchanged.

The existing current-unique inventory and canonical representative remain the work
scope. An invalidated archive receipt does not create an archive job. Accepted and
Iran-guarded identities retain periodic safety checks, bounded workers and expiry
without stale fallback. A first incomplete-profile assessment gets the existing
short deep retry; subsequent uncertainty backs off within the existing budget.

## Deployment evidence

The design was reviewed through the OpenAI API using `gpt-6-astra`, maximum
reasoning and pro mode. Review is advisory; actual test, canary and deployment
receipts belong in the P65 report. No source credentials or raw configs were sent.
Do not claim 1000 configurations in 600 seconds, Android/carrier compatibility,
all-domain/QUIC coverage or future uptime from these finite checks.
