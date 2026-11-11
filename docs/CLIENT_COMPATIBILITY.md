# P40 client compatibility contract

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
The independent P39 egress pipeline retains its own evidence gates and reconciles
convert.Version changes without deleting source records. Never restore the retired
Residential implementation or substitute ordinary health for Ads egress proof.

Schema24 records current raw/candidate/runtime hashes, target and driver versions,
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

## Plaintext VLESS stream compatibility

`legacy-vless-stream-redirect.v2` covers RAW/TCP with optional HTTP header,
WebSocket, gRPC, XHTTP, HTTPUpgrade and mKCP when supported by the pinned cores.
The original protocol/credentials/endpoint/transport are retained. A dedicated
Freedom outbound redirects the VLESS dialer's logical loopback destination to
only its original public endpoint. No listening socket or relay inbound exists.
Original remote socket policy is on that fixed helper. Implicit HTTP Host/authority
is materialized from the original endpoint before changing its logical address.
This is **NO ADDED ENCRYPTION** compatibility, not TLS/REALITY or a security upgrade.
Do not export only the VLESS URI: the helper is part of the Custom JSON graph.

`Logical` validates exclusive ownership and unwraps recognized fixed-redirect or
old bounded-listener graphs before normal/Smart/Ads composition. `Wrap` rebuilds
collision-free fixed helpers. Shared routes/pool selectors, ambiguous ownership,
unsupported flow/proxySettings/finalmask/stream destination or separate XHTTP
legs remain explicit driver limitations. Repeated composition is deterministic.
Native VLESS encryption is never replaced with `none`.

Known VMess `skip-cert-verify` maps to the same certificate policy as `insecure`;
conflicting/invalid values fail explicitly. Display-only `name` and `test_name`
are accepted. Unknown transport/plugin fields are not silently discarded. The
existing strict-certificate alternative still requires real connection proof.

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

**Native delay:** official Lib40 MeasureOutboundDelay removes inbounds, routing,
DNS and observers but keeps outbounds. Fixed outbound redirects therefore survive
its filtering, unlike the former listener relay. The source-compatible negative
controls and unmodified official-library Internet tests are recorded separately.
A Smart profile's native delay measures its first outbound, not the leastLoad
pool; the full running-profile test exercises the balancer. JSON cannot change
APK scheduling, built-in retry count/timeouts, or guarantee latency on every ISP.

## What the evidence means

The P35 matrix applies the common template before protocol/transport/security
build checks on both cores; see the P35 reports for accepted and rejected rows. Acceptance is not a useful-network or universal
combination claim. RAW/TCP aliases, TLS/REALITY constraints, deprecated accepted
WS/gRPC/HTTPUpgrade and removed legacy HTTP/QUIC transports remain distinct.
Hysteria2 is native TLS/QUIC; Hysteria1 and TUIC require another core/client, not a
renamed Xray protocol. WireGuard has native encryption and transport.

Closed synthetic traffic fixtures test TCP, UDP and UDP443 for VLESS TLS
TCP/WS/XHTTP/gRPC/HTTPUpgrade, VMess TCP and WS/TLS, Trojan TLS, Shadowsocks AEAD,
Hysteria2 and WireGuard, plus plaintext VLESS stream redirects, on both core versions. Six Shadowsocks
ciphers (three AEAD and three 2022 methods) have separate TCP/UDP/UDP443 fixtures.
A fixture certificate pin is not public-PKI proof. Separate existing-project live
canaries have valid-public-certificate HTTPS, Cloudflare/Google DoH, DNS TCP/UDP,
QUIC Version Negotiation and full HTTP/3 DoH receipts. VN alone is never HTTP/3.

Production baseline requires HTTPS, DNS TCP/UDP, both DoH resolvers and QUIC VN
where the transport supports UDP; HTTP-only inputs explicitly record UDP unsupported.
HTTP/3 is a separate optional bounded check using pinned aioquic1.3.0. Failure or
absence is recorded; it is not silently promoted to passed. Smart membership also
retains the existing Instagram check; a balanced profile pass does not prove every
internal member or every possible selected path.

Live evidence is snapshot-specific and separate from controlled fixtures.
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

## Current diagnostics and repeatable audit

Normal v7, Smart v11 and Ads v4 share the compatibility boundary. Schema21 fences
prior final-byte proofs before serving and queues fresh validation. P34 acquisition
recovery and blob-reference indexes are preserved. Current family metadata appears
on Output rows and `/api/conversion/families`; build and connection counts differ.
`scripts/template-audit.py` validates all accessible unique candidates, samples at
most three per family for traffic, and builds/tests representative Smart profiles.
Concurrent expiry/deletion can make raw records unavailable; these are recorded as
snapshot-unavailable, not malformed or core-rejected. Official native delay tests
are optional and sandboxed. Audits never grant health/publication to other records.

In the strict native-delay sandbox, some domain endpoints attempt UDP53 after
Lib40 discards DNS/routing. The firewall denies that bootstrap; the native test
is blocked even when the exact full profile passed. No direct-DNS exception was
added. This is not an observed Android APK failure or proof of universal DNS
behavior on devices. IP endpoints and full-profile tunnel DNS tests are separate.


## P40 parser repairs and audit

The versioned share-compat.v2 decoder strips only the URI fragment before Base64,
supports percent-encoded payload bytes without changing '+', recognizes explicit
VMess security/scy aliases with conflict checks, and accepts known exact-empty
placeholder strings. Empty strings do not authorize dropping populated extensions.
Numeric port spelling preserves digits and normalizes Persian/Arabic forms; missing
ports are not inferred. Bounded realityVersion metadata never substitutes for a key.
endpoint-ipv6-literal.v1 unwraps complete IPv6 literals in protocol address fields
while preserving Host/SNI, credentials, ports and exact raw storage.

Read P40-validation.json for the finite all-input four-core audit and bounded real
network representatives. Original variant rejection and final input rejection are
separate: strict TLS alternatives still require valid certificate connection proof.
The audit itself cannot mark a production config healthy. Schema24 fences stale
leases and final-byte evidence in normal/Smart/Splash/Ads; future inputs take the
same common preparation path. Actual Android APK/carrier execution remains not_run.
