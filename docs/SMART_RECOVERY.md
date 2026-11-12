> Current policy: Smart v15 / Ads v8 / schema27. See [SMART_CARRIER_HEALTH.md](SMART_CARRIER_HEALTH.md). The v14 timing and tolerance below are historical; the admission and preservation rules remain.

# Smart v14: observed recovery and payload qualification

The fast primary remains leastLoad, expected 3, maxRTT 2s, tolerance 0.15.
A successful but slower path can be excluded by that predicate. Only when the
primary pool has no eligible member does its fallback outbound redispatch once
through a scoped loopback tag into another leastLoad pool (expected 1, maxRTT 5s,
tolerance 0.5). The secondary fallback is blackhole. It cannot select unobserved
or dead paths. This is NOT retry of a failed request, stream migration, permanent
session affinity, or a guarantee that every destination works.

Main and endpoint-bootstrap recovery have separate tags and selectors. Bootstrap
uses exactly the same independent-IP membership as its strict pool. Helpers are
outside proxy selectors. Fixed-width entry tags avoid prefix collisions.
Balanced Ads exits have a separate recovery scope; internal recovery/DNS/chain
rules precede application domain routing. Native filtered ping keeps a real
independent proxy first and does not measure the complete Smart routing policy.

## Admission and exact final output

Raw inputs, source controls, ordinary template v9, Ads classifier v5, credentials
and TLS validation remain. Smart v14 / Smart Ads v7 and schema 26 fence old cached
Smart bytes and short-request-only membership receipts.

The shared isolated driver runs two general HTTPS checks, Instagram HTTPS,
DNS TCP/UDP, both DoH resolvers, QUIC VN and optional full HTTP/3. It now also
requires complete 2 MiB download, complete 1 MiB upload with HTTPS acknowledgement,
and a post-load HTTPS check for Smart admission/publication. Curl timeout with
HTTP 200 and a partial payload is failure. Upload evidence is exact submitted byte
count plus verified HTTPS response, not a server-side digest. Performance proof
is finite and belongs to the exact candidate hash, core, time and test network.
A failed transfer retains the original configuration and its diagnostic.

Payload tests are sequential per candidate, capped at 20–45 seconds per transfer
and 10 seconds for the post-load request. Existing worker concurrency caps and
namespace/cgroup limits remain; the supervised candidate deadline includes the
new stage. There is no endpoint flood, automatic TLS bypass or direct fallback.
A speed-test endpoint outage can withhold qualification; it does not prove the
server is unusable. A pool result does not qualify every member; member admission
is independently gated by current conversion evidence.

The production quality receipt uses payload_v1. Revalidation naturally restores
eligibility after a transient failure. Migration only makes current inputs due
and fences live leases; archived results, raw storage and deletion fences remain.
Old ordinary HTTPS receipts are not rewritten into payload successes.

## Preservation and bootstrap inspection

Smart socket defaults fill absent keepalive values instead of replacing explicit
ones. Address-family intent is preserved; Use* is converted to the corresponding
Force* to prevent fallback to system DNS. Absent/AsIs retains the established
ForceIPv4 Smart policy. IPv6-only device/carrier operation remains a separate test.
XHTTP inspection follows official precedence: xhttpSettings over splithttpSettings;
extra replaces outer download settings. Only effective endpoint/dependency fields
contribute bootstrap domains; SNI and HTTP Host are not dial destinations.
Composition accepts exactly one bounded JSON object and rejects trailing data.

## Reproducible checks

Run scripts/check.sh with the pinned Go toolchain, then run internal/smart tests
with XRAY_TEST_BINARY set separately to each official pinned core.
scripts/payload-quality-test.py covers partial payloads, TLS failures, timeouts,
wrong versions/counts and post-load failure. Panel tests prevent a worker code
alone from asserting payload proof and fence old cache/leases across migration.

scripts/smart-recovery-test.py CORE GENERATED_SYNTHETIC_PROFILE runs real cores in
a closed namespace: baseline slow path rejection, recovery, fast path, DNS A/TXT
over TCP/UDP and observed all-down blocking. Build the synthetic profile through
config-probe-plan --smart. Its proxy endpoint is loopback:24456 with fixture UUID
11111111-1111-4111-8111-111111111111; the harness supplies only loopback peers.
The relay adds 1150ms in each direction. The baseline fixture uses 10s/1
observations; generated candidates retain their supplied production timing.
This is a controlled transport test, not an Iranian carrier result.

scripts/smart-bootstrap-test.py separately verifies real TLS/HTTP2 DoH through
IP-only bootstrap, a domain member, no system DNS attempt and cold all-down
blocking without host pre-seeding for the synthetic endpoint.
Queue tests use clearly labeled synthetic receipts; they do not certify traffic.
The real traffic lane executes the exact exported bytes and records independent
payload and transport results.

## Operational limits and recovery

The main 15s × 3 observation sweep is retained. A network change can leave stale
observations until remeasurement. Existing TCP streams cannot migrate between
servers. The 5s recovery ceiling and 2500ms DNS deadline remain bounded tradeoffs,
not measured universal optima. Do not globally tune DNS deadlines, expected,
MTU, Mux or fragmentation from a short ping or one app.

Before activation, hold /var/lock/config-repair-deploy.lock and take a matching
stopped DB/blob/key/release backup. Schema 26 must not be opened by pre26 binaries.
Recovery prefers a forward fix; restoring a matched snapshot requires stopping
writers and preserving newer external deletion evidence and user changes.
Do not rewind a live database merely to switch the executable.
