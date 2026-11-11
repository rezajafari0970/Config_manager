# Ads split-egress assessment, version 5

The user's latest definition supersedes the former ISP classification gate:
**Residential in this project means repeated dedicated Ads egress.** It does
not assert a household/ISP network type. A complete stable shared egress is
NO_SPLIT; missing, mixed or ambiguous evidence remains UNKNOWN. Only RESIDENTIAL
with fresh evidence qualifies the four exports. There is no external IP-type,
ASN, country, provider-quota or AI dependency. The old classifier is removed.

## Evidence contract

Every unique acquired raw hash is queued across all source links. Preserve raw
bytes and credentials, construct the common template, then test both exact
variants. A recognized compatibility redirect and DNS dialer through the sole
proxy are allowed; ambiguous multiple paths and general Direct routes remain
unqualified. Resolver-only Direct exceptions are outside this finite HTTPS
scope and remain blocked by the sandbox firewall. No DNS-leak claim is made.

Pinned official Xray 26.11.9, 26.10.31, 26.7.28, 26.5.9 support compatibility
fallback without weakening TLS or inventing fields. Native TUIC/Hysteria use
pinned sing-box 1.14.4. Runtime startup validates the actual configuration; a
redundant separate `-test` launch is no longer required. The core hash, variant
hash and runtime hash are bound to each report. Xray runtime bytes must equal
the tested variant bytes. Unsupported or IPv6-only entries remain explicit.

SOCKS connects to googleads.g.doubleclick.net, pubads.g.doubleclick.net and
securepubads.g.doubleclick.net. Strict TLS authenticates the actual Ads SNI.
GET /robots.txt completes without redirect following; on the same open TLS
stream, GET www.google.com /sorry/index with a random nonce obtains the
peer-reported source. No reconnect, ad click or impression is generated.
Require a global source, echoed nonce, certificate/body digests and unique
connection ID. Record whether the second HTTP authority matches the TLS cert.
This undocumented diagnostic is an authenticated peer report, not a signed
Google API or full network topology. Shared frontend internals can affect its
meaning. Results cover these three domains, TCP and observed address family;
they do not establish all Google Ads traffic, UDP/QUIC or future behavior.

Pair every wave and variant with two authenticated controls: www.cloudflare.com
and api.ipify.org. Cloudflare's bounded cached A resolution only chooses the
SOCKS destination; its IP is NEVER a firewall Direct exception. Explicit IPv6
controls run concurrently and are retained when needed. Never compare IPv4
Ads with IPv6 controls. Each domain in each variant needs at least two valid
independent paired observations. All observed Ads sources must be disjoint
from the stable control source per family; any overlap or rotating controls
prevents acceptance. Even an observed overlap in an otherwise unpaired wave
cannot be silently discarded to manufacture a split.

A fully observed shared raw egress is sufficient for early rejection; its
template need not run. Go independently verifies the raw evidence, and this
shortcut can never qualify an output. A running core with network timeouts does
not repeat the same timeouts on four versions; build/start incompatibilities
still use the pinned fallback matrix. Fast jobs use two waves; deeper retries use three sequential waves with8s
connection deadlines. XHTTP always uses sequential waves to avoid connection
bursts. Destinations within a wave run in parallel. Ordinary fast deadlines
remain3s. Partial failures defer to deeper retries and are never NO_SPLIT by
elapsed time alone. A repeated split can qualify in the fast pass.

Source addresses are transient helper memory only. At report completion each
is replaced with HMAC-SHA256 using a fresh random job-local key. Equality
within the job is retained; the key and plaintext addresses are discarded.
No IP, ASN or country is persisted in new measurement reports or displayed.
Old policy reports are fenced and cleared on reconciliation, not relabelled.

## Isolation, scheduling and lifecycle

The feature owns internal/egress, internal/panel/egress_*, scripts/egress-*,
cmd/config-egress-worker and its service. Shared collection/storage remains.
Every worker owns a persistent trusted helper and namespace but launches fresh
unprivileged core processes. Endpoint-only firewall, checksum pins, bounded
cgroups and disjoint cge/cgt namespaces remain. After each job kill cores,
deny egress, verify empty cgroup and erase private runtime files before ack.
Parent death and startup cleanup recover only this feature's own resources.
Capacity is bounded at31 and reduced under memory pressure. Leases renew every
20s, expire after120s and are revision-fenced. Manual deletion always wins.

Accepted proof nearing expiry has priority. Current inputs receive initial
assessment and one deep retry before historical backlog. Partial first passes
retry after30s. Accepted entries recheck after120s and are exempt from ordinary
age-based retention; publication needs evidence younger than600s. Uncertainty
quarantines and withdraws exports without calling it a route failure. Three
spaced credible negatives, outside the outage guard, revoke globally and fence
re-import. Source settings, ordinary/Smart gates and deletion journal remain.

## Panel and exports

/egress shows accepted split-route configurations by default. All-input
checkbox exposes pending, incomplete and unsupported cases with source names
and diagnostic evidence. The UI explains the project-specific Residential
label and distinguishes five-minute assessments from conclusive route results.

Four stable links: JSON array, JSONL, rotating single JSON and immutable raw
text. Publication checks current eligibility and generation, and rotation is
persistent. Empty/stale pools return503; concurrent rotation may return409.
No ordinary-server fallback is substituted for an empty accepted pool.

## Verification and performance

Run scripts/check.sh, the isolated crash/recovery fixture, private migration
rehearsal and scripts/egress-throughput-test.py on1000 distinct real retained
inputs. Test harness never changes production evidence or sources. Report the
actual300s count and states; UNKNOWN/UNSUPPORTED are not conclusive decisions.
The target1000 in300s is measured, not asserted from configured concurrency.
Preserve strict TLS, raw data, server pins and stage-specific error evidence.

Deployment holds /var/lock/config-repair-deploy.lock and captures a matching
stopped DB/blob/key/release backup. Keep schema25 protections and never rewind
the authenticated deletion journal. Historical P39–P43 reports describe older
policies and do not override the user's version5 definition.
