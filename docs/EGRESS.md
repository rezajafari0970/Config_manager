# Ads egress investigation, version 2

The previous Residential subsystem was retired at the user's explicit request.
The replacement owns internal/egress, internal/panel/egress_*, scripts/egress-*,
cmd/config-egress-worker and config-repair-egress.service. Shared input collection,
conversion, encrypted blob storage and the sandbox are reused. Schema23 removes
the legacy tables and seeds only current memberships; schema22 acquisition
admission and all ordinary/Smart/Splash/Ads behavior remain.

## Measurement and its limits

For each acquired raw hash, preserve the exact original blob. Prepare raw and
common-template variants without inventing credentials, destinations or TLS
settings. Use pinned official Xray26.11.9,26.7.28,26.5.9 in compatibility order;
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

Pair every wave with separate IPv4/IPv6 HTTPS Cloudflare trace and ipify controls in the same
variant. Never assign control IPs to Ads. Three independent waves for all three
Ads names and both variants are required. Distinct output is reported only for
disjoint observations against a stable control IP; pooled/rotating controls
remain inconclusive. Screening can stop early but cannot produce eligibility.

Proxycheck v3 is the exact-IP network-class driver:
https://proxycheck.io/api/
https://proxycheck.io/api/querying/
Only its explicit Residential network.type qualifies all observed Ads IPs.
ASN, country, ISP names and absence of Hosting do not substitute. Wireless,
Business, mixed and missing labels do not qualify. Store source, time, digest,
ASN and country as provider-reported facts. Classifications are probabilistic;
country and Residential are separate from route observation.

The free driver uses HTTPS with certificate verification, rejects redirects,
sends only public IPs, uses a one-hour cache, reserves its daily budget before
networking and caps unregistered use at90 unique queries/day. Missing/quota
results are UNKNOWN. It is a replaceable driver; no AI model decides network
facts and no OpenAI key/config secret is sent externally.

## Lifecycle and concurrency

Every membership insertion enqueues its raw hash, deduplicated across links.
Two independent root supervisors slots own bounded netns/cgroups, with nobody
running untrusted cores and only public actual upstream endpoints allowed.
Random claims are revision-fenced; renew every20s; expired120s leases recover.
Worker death triggers parent-watch cleanup and scoped startup cleanup.
Accepted members get priority rechecks every120s and are exempt from ordinary
retention. Their evidence lasts at most600s and never exceeds classifier expiry.
Uncertainty immediately removes them from publication but retains them for retry.
Three credible negative observations spaced at least30s, outside the shared
outage guard, purge the identity from all active outputs. Generic permanent
input exclusions prevent re-import; manual deletion is also global.
Provider outage, ambiguous response and unsupported driver are not negative votes.

## Outputs and verification

/egress is the new page, with accepted members by default and an optional
all-input diagnostics view. /residential redirects; old APIs/subscriptions return410.
Four stable authenticated URLs: JSON array, JSONL, rotating single JSON, raw text.
Exports read immutable blobs and recheck the whole eligibility generation before
sending. Rotation persists; a competing request can receive409 and retry.
Empty/stale eligible pools return503, never a fallback to ordinary servers.

Run scripts/check.sh, scripts/egress-worker-test.py and the private migration
rehearsal test. Synthetic policy fixtures never count as live Residential proof.
Runtime validation receipts and secrets stay private under .local or /tmp.
Hold the deployment lock and create a stopped matching DB/blob/key/binary backup.
Never open schema23 with an older binary; use fix-forward or an explicitly
coordinated matched rollback. Historical code remains in Git history only.
