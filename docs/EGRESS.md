# Ads egress investigation, version 4

The previous Residential subsystem was retired at the user's explicit request.
The replacement owns internal/egress, internal/panel/egress_*, scripts/egress-*,
cmd/config-egress-worker and config-repair-egress.service. Shared input collection,
conversion, encrypted blob storage and the sandbox are reused. Schema23 removed
the legacy tables and seeds only current memberships; schema22 acquisition
admission and all ordinary/Smart/Splash/Ads behavior remain. This release incorporates P42/schema25 and its strict TLS candidate generation.

## Measurement and its limits

For each acquired raw hash, preserve the exact original blob. Prepare raw and
common-template variants without inventing credentials, destinations or TLS
settings. Use pinned official Xray26.11.9,26.10.31,26.7.28,26.5.9 in compatibility order;
TUIC/Hysteria variants use pinned sing-box1.14.4. Unsupported, IPv6-only ingress
and ambiguous multipath remain visible UNKNOWN/UNSUPPORTED, never Datacenter.

In an endpoint-restricted namespace, SOCKS carries the actual destination domain.
TLS validates that Ads domain's certificate and SNI. GET /robots.txt completes
without following redirects. On the same still-open TLS stream, request the
fixed www.google.com /sorry/index diagnostic with a random nonce. Record a
returned IP only when it is global and the response contains the nonce. Record
certificate hash, response hash, stream identifier, HTTP statuses and whether
the additional authority is covered by the certificate. Do not reconnect.

This is an authenticated Ads *peer-reported* observation. The error page is an
undocumented Google behavior, not an attestation API. A shared Google front end
or internal reverse proxy can affect its semantics. It does NOT prove every real
Google Ads request's source, every category domain or a household connection with
100% certainty. The UI explicitly scopes the finding to three DoubleClick names,
observed IP family, TCP, raw and template. UDP/QUIC and Android/carriers are not
qualified. The MO result from an old experiment is not carried forward.

Pair every wave with authenticated IPv4 Cloudflare (www.cloudflare.com) and ipify controls in the same variant. Start the two explicit IPv6 controls concurrently and retain them when an Ads observation uses IPv6; never infer an IPv6 source from IPv4 controls. Never assign control IPs to Ads. Three independent waves for all three
Ads names and both variants are required. Distinct output is reported only for
disjoint observations against a stable control IP; pooled/rotating controls
remain inconclusive. A bounded first assessment can defer network failures to a deeper retry; it cannot produce eligibility. Independent TLS connections for the three waves start with150ms staggering, using one immutable verified SSL context per session. The same-stream Ads diagnostic and raw/template hash gates remain unchanged.

Proxycheck v3 is the exact-IP network-class driver:
https://proxycheck.io/api/
https://proxycheck.io/api/querying/
Only its explicit Residential network.type qualifies all observed Ads IPs. A family-aware prefilter spends lookup quota only on observed distinct Ads sources, never on stable overlap.
ASN, country, ISP names and absence of Hosting do not substitute. Wireless,
Business, mixed and missing labels do not qualify. Store source, time, digest,
ASN and country as provider-reported facts. Classifications are probabilistic;
country and Residential are separate from route observation.

The free driver uses HTTPS with certificate verification, rejects redirects,
sends only public IPs, uses a one-hour cache, reserves its daily budget before
networking and caps unregistered use at100 unique queries/day. Missing/quota
results are UNKNOWN. It is a replaceable driver; no AI model decides network
facts and no OpenAI key/config secret is sent externally.

## Lifecycle and concurrency

Every membership insertion enqueues its raw hash, deduplicated across links.
Up to31 independent slots in one root supervisor own bounded netns/cgroups, with nobody
running untrusted cores and only public actual upstream endpoints allowed.
One GiB aggregate memory and two CPU cores bound sandbox cores per scope; the supervisor unit has a separate2GiB limit. Admission falls to12/4/0 slots below1.5GiB/1GiB/512MiB host MemAvailable. Slots remain in disjoint32-address blocks. Random claims are revision-fenced; renew every20s; expired120s leases recover.
Each slot retains one trusted helper and an empty network namespace between requests. Fresh unprivileged cores are killed, the namespace denies egress, and private files are removed before acknowledgement. The helper uses network-only nsenter to avoid repeated mount-namespace overhead; shared drivers retain their previous default. Core digests are cached only while file identity, size and modification time match. Worker death triggers parent-watch cleanup and scoped startup cleanup.
Accepted members near proof expiry have priority; other accepted rechecks yield to first assessments so new links are not starved. Accepted members get rechecks every120s and are exempt from ordinary
retention. Their evidence lasts at most600s and never exceeds classifier expiry.
Uncertainty immediately removes them from publication but retains them for retry.
Three credible negative observations spaced at least30s, outside the shared
outage guard, purge the identity from all active outputs. Generic permanent
input exclusions prevent re-import; manual deletion is also global.
Provider outage, ambiguous response and unsupported driver are not negative votes.

## Outputs and verification

/egress is the new page, with accepted members and distinctly labelled ADS_SPLIT candidates by default and an optional
all-input diagnostics view. /residential redirects; old APIs/subscriptions return410.
ADS_SPLIT means a repeated different-source observation with missing/mixed network classification, or incomplete full coverage but at least two paired independent observations per required domain and variant. Incomplete coverage is labelled in its reason and never qualifies exports. It never qualifies the four Residential exports. NO_SPLIT means complete stable overlap; it cannot qualify even when a provider labels those IPs Residential. Partial evidence retains observed IPs but cannot qualify. A recognized fixed-destination compatibility redirect and DNS dialed through the sole proxy are not treated as extra egress paths; ambiguous chains and direct destination rules remain unqualified.

Four stable authenticated URLs: JSON array, JSONL, rotating single JSON, raw text.
Exports read immutable blobs and recheck the whole eligibility generation before
sending. Rotation persists; a competing request can receive409 and retry.
Empty/stale eligible pools return503, never a fallback to ordinary servers.

Run scripts/check.sh, scripts/egress-worker-test.py and the private migration
rehearsal test. Synthetic policy fixtures never count as live Residential proof.
Runtime validation receipts and secrets stay private under .local or /tmp.
Hold the deployment lock and create a stopped matching DB/blob/key/binary backup.
Never open schema25 with an older binary; use fix-forward or an explicitly
coordinated matched rollback. Historical code remains in Git history only.

## Performance qualification

The target is1000 initial assessments in300s, not a guarantee of1000 definitive Residential decisions. The UI reports observed five-minute throughput and state counts. Slow or unresponsive remote servers, unsupported transports, missing Ads diagnostics and provider quota remain explicit. No elapsed deadline is converted into a successful classification.

Use scripts/egress-throughput-test.py for an isolated queue populated with up to2000 distinct retained real source hashes. Its receipt separates assessed/state/reason/sample outcomes and records the300s checkpoint, maximum active jobs and minimum available host memory. It never mutates production memberships, evidence or outputs. The worker kill/recovery fixture keeps an explicit two-worker count; production defaults to two slots per visible CPU, bounded at31 (eight on the measured four-CPU server). The31-slot stress test increased CPU contention and deadline failures; configured capacity is not a throughput guarantee.

Classifier reservations and exact-IP single-flight leases use short atomic file locks. HTTPS runs outside the lock with at most four parallel fetches per job. The free100/day cap survives crashes, and exhausted quota is displayed. Increasing worker count does not increase provider quota. TLS, server credentials and the certificate pin are preserved. Runtime AI inference is not used as network evidence.


## P43: source DNS bypass and overloaded session correction

Alan splash's immutable raw JSON contained port53 and fixed public resolver-IP
Direct routes. The shared all-traffic proxy-health preflight rejected it before
any network measurement; 1.1.1.1 was also an invalid proxy control for that raw
routing. The dedicated egress-preflight module now ignores only enumerated
resolver-only exceptions during inspection, executes exact original bytes, and
keeps the endpoint-only namespace firewall unchanged. Source direct DNS remains
blocked in this lab. This proves only the named HTTPS scope, not DNS leak freedom.
Broad Direct rules, arbitrary resolver addresses/subnets, unknown extensions,
Direct defaults and ambiguous paths remain unqualified. Shared ordinary/Smart
preflight is unchanged. Controls now use www.cloudflare.com and ipify.

XHTTP and deep rechecks run three sequential waves (destinations within each
wave remain parallel). Deep per-connection timeout is8s. This reduced connection
burst failures on the measured source. Current inputs get a first assessment and
one deep retry before retained historical backlog; repeated retries then yield.
Manual rechecks have explicit priority, accepted proof-expiry priority remains.
No arbitrary input gets a Residential override.

Partial route evidence now shows fresh observed type counts and Mixed /
ResidentialObserved without qualifying exports. Source names, exact driver
failure codes, control IPs and provider facts are visible in the panel. Missing
classifier reports no longer erase the latest known quota status.

Optional server-only Proxycheck configuration is /etc/config-repair/egress-intel.json
with api_key and daily_limit. The file must be regular, root-owned and mode0600.
Without a key the limit is100/day; a key defaults to1000/day, with an explicitly
configured account budget permitted. No account is registered or purchased.
The durable daily usage counter is not reset on configuration changes. Keys are
never emitted in status, logs, receipts or subscriptions. The current server has
no registered IP-classification key. Provider quotas still bound coverage.

Official API limits and network types: https://proxycheck.io/api/ .

Partial distinct-route observations remain visible in the default page, explicitly
unqualified. A first assessment with incomplete controls or Ads echoes schedules
a deep retry after30s instead of300s. This is scheduling, not promotion to a
definitive classification. A real fast Alan test (17.176s, six Ads echoes)
reproduced this condition and the queue/UI contract has a regression test.

The Cloudflare IPv4 control uses a bounded, cached A lookup of the fixed
www.cloudflare.com name, a SOCKS IPv4 destination and verified TLS/Host for that
same name. The IP is never added to the namespace's direct firewall allowlist.
This fixes a reproduced dual-stack mismatch: the former domain-based request
returned IPv6 while ipify returned IPv4. Explicit IPv6 controls remain separate.
Resolver failure is an unknown control, never an IPv4 inference from IPv6.
