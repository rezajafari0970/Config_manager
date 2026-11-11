# P33 client compatibility contract

Target: official v2rayNG **2.3.15**, Lib40, bundled Xray **26.10.31**.
The additional forward-compatibility gate uses official Xray **26.11.9**.
Pins and the generated build matrix are in `docs/reports/P33-capability-matrix.csv`.
The branch name is historical; it does not identify the current client target.

## Shared boundaries

`collect` preserves source-response and extracted original bytes in the encrypted
content-addressed vault. Format drivers decode/parse without changing the raw
record. `repair` owns bounded structural candidates; `compat` owns protocol,
transport and security capability checks and narrowly qualified adaptations.
`convert` and `smart` build normal, balanced, Smart, Splash and Ads candidates.
The same supervisor gate validates final bytes on both pinned cores, checks the
source-derived Android Custom Config transformations, and runs those final bytes
on **26.10.31** in the existing isolated sandbox. Publication requires exact
candidate/runtime SHA equality and fresh HTTPS/DNS/UDP evidence where supported.
No successful build alone is a healthy output; an old-core pass cannot qualify.
Independent residential diagnosis retains its own checksum-pinned historical
supervisor baseline; publication upgrades cannot silently retarget that preflight.
Positive/negative live self-tests and a regression test cover this separation.

Schema20 records current raw/candidate/runtime hashes, target and driver versions,
changes, attempts and transport checks separately. Receipts are bounded metadata,
not a new historical sample bank. Originals remain immutable. Migration updates
persisted admission predicates, fences old leases and caches, and schedules new
proofs for current inputs and independently retained eligible records. Source
controls, scheduling, retention policy and durable rotation are preserved.

The registries are compiled, reviewed drivers. No uploaded plugin executes on the
host. Collection and worker pools retain separate bounded concurrency, deadlines,
lease recovery and resource namespaces. New empty source insertion does not
rescan all conversion/Smart views; edits and arriving collector data still reconcile
atomically. The 1,003-source bulk/CSRF regression passes without raising its
request deadline. Payloads only cross the protected Unix
queue. Untrusted cores run without root, in a restricted filesystem/cgroup/netns.

## Legacy plaintext VLESS TCP driver

`legacy-vless-tcp-loopback.v1` applies only to VLESS with no transport TLS/REALITY,
no VLESS encryption, TCP/RAW, optional none/HTTP header, and no unsupported flow,
stream override/finalmask or proxySettings. It preserves credentials, HTTP headers
and actual endpoint. The local VLESS leg dials a unique loopback port; a bounded
`dokodemo-door` leg targets only the actual endpoint. Exact scoped allow followed
by relay deny precedes normal routing. Outbound socket/carrier options move to the
remote transport leg, including Ads dialer chaining. Port/tag allocation is per
complete profile, reserves existing listeners, and rejects ambiguous graphs.

**This is legacy compatibility with NO ADDED ENCRYPTION.** It uses the core's
loopback exception. It never upgrades security and never downgrades encrypted
inputs. It is retained as Custom JSON; exporting only a VLESS URI would lose the
relay. Other plaintext transports remain `legacy_vless_transport_requires_driver`
until separately qualified. TLS/REALITY keys and server capabilities are never
invented. Unsupported inputs remain stored with explicit diagnostics.

`Logical` recognizes complete source relay graphs before composition; `Wrap`
rebuilds their infrastructure without collisions. Unbounded/shared routes,
missing deny, wrong listener, carrier extensions and ambiguous ownership fail
closed. Repeated application is deterministic. Number precision and Smart root
field ordering are preserved. No post-validation outbound shuffle changes the
published bytes; durable profile rotation still works.

## Import and publication

Single output is one object. Native group subscriptions are **JSON arrays** at
`configs.json`; existing `configs.jsonl` remains available for integrations. This
applies to global, per-source, Smart, Splash, Ads and classified export surfaces.
Original/raw downloads are explicitly unqualified source data, not healthy client
outputs. Read-time ownership, proof, version and expiry gates still apply.

The pinned Android importer tries an array before falling back to an object.
Its first Gson BEGIN_ARRAY exception is not a final import failure. The Custom
Config builder preserves relay/routing, changes stats/policy.system according to
preferences and may add a TUN inbound. Four source-derived speed/TUN variants are
core-tested. Package-to-UID routing needs a real Android device and is not certified.

**RealPing caveat:** Lib40 MeasureOutboundDelay removes all inbounds, routing,
DNS and observers. A loopback-relay default can therefore fail the app's delay
button even when full VPN execution works. This repository does not patch the
Android APK and does not claim to fix that distinct app limitation.

## What the evidence means

The release matrix tests 184 protocol/transport/security builds across both cores;
132 are accepted as constructed. Acceptance is not a useful-network or universal
combination claim. RAW/TCP aliases, TLS/REALITY constraints, deprecated accepted
WS/gRPC/HTTPUpgrade and removed legacy HTTP/QUIC transports remain distinct.
Hysteria2 is native TLS/QUIC; Hysteria1 and TUIC require another core/client, not a
renamed Xray protocol. WireGuard has native encryption and transport.

Closed synthetic traffic fixtures test TCP, UDP and UDP443 for VLESS TLS
TCP/WS/XHTTP/gRPC/HTTPUpgrade, VMess TCP and WS/TLS, Trojan TLS, Shadowsocks AEAD,
Hysteria2 and WireGuard, plus legacy VLESS TCP/HTTP relay, on both core versions.
A fixture certificate pin is not public-PKI proof. Separate existing-project live
canaries have valid-public-certificate HTTPS, Cloudflare/Google DoH, DNS TCP/UDP,
QUIC Version Negotiation and full HTTP/3 DoH receipts. VN alone is never HTTP/3.

Production baseline requires HTTPS, DNS TCP/UDP, both DoH resolvers and QUIC VN
where the transport supports UDP; HTTP-only inputs explicitly record UDP unsupported.
HTTP/3 is a separate optional bounded check using pinned aioquic1.3.0. Failure or
absence is recorded; it is not silently promoted to passed. Smart membership also
retains the existing Instagram check; a balanced profile pass does not prove every
internal member or every possible selected path.

The supplied TC81 endpoint was currently unreachable from this server during this
run. The relay built and ran, but that endpoint is not certified healthy here.
No Android APK/device, MCI/Irancell/RighTel, long-term stability, IPv6-only runtime,
throughput or zero-loss guarantee is claimed. Unsupported/ambiguous extensions
remain explicit instead of silently rewriting their semantics.

## Assets and DNS

Final JSON is never stripped or rewritten for the test. Endpoint resolution is
pinned only in the sandbox OS environment; app DNS stays in its configured path.
The sandbox has no general direct-DNS exception. Explicit pre-existing local-DoH
bootstrap is limited to fixed resolver IP/443, not arbitrary application traffic.
Smart keeps the requested dual tunneled bootstrap/main leastLoad layout.
Generated sockets use ForceIPv4; explicit normal-output IPv6 family preferences
use the corresponding Force strategy. XHTTP download sockets are covered too.
DNS queryStrategy remains UseIPv4. This closes Xray DialSystem's system-resolver
fallback after internal DNS failure. The closed startup/cold-failure fixture
asserts zero system-DNS attempts, rather than relying on its firewall to hide them.
The inherited Ads `geosite:google@ads` dependency remains deliberate for category
coverage and is checked against installed core assets; Android asset availability
still needs device verification. No blanket UDP443 block is added.

## Reproduction and evidence

Run `scripts/check.sh`, client compatibility integration, capability matrix,
relay and protocol traffic fixtures, all four queue SIGKILL tests, browser/restart
checks and the live-DB migration rehearsal. Queue tests use explicitly synthetic
connection receipts and documentation addresses; actual traffic evidence is a
separate test. Private source URLs, credentials, DB and raw canaries never enter Git.
Driver changes or target upgrades invalidate prior publication evidence. Failed
rules do not continue publishing new outputs. A new release requires source pins,
requalification, matched stopped-service backup and separately verified runtime.

Official source:
- https://github.com/2dust/v2rayNG/releases/tag/2.3.15
- https://github.com/2dust/v2rayNG/tree/57907cf354d9769200ed4717b4430d605abe0952
- https://github.com/2dust/AndroidLibXrayLite/tree/2ffe5f794a32ba15b122e177b5acffc6f11c4959
- https://github.com/XTLS/Xray-core/releases/tag/v26.10.31
- https://github.com/XTLS/Xray-core/releases/tag/v26.11.9
