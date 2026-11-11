# P35 shared templates and native delay — 2026-11-11

Parent: a9fc642 (P34 integrated; identical runtime source to da05ca3). Worktree work/p35-template-realping-20261111.
P34 acquisition recovery is integrated without changing its behavior.
Status: deployed and runtime verified. Source publication is tracked in .local/p35-published.json.

The user authorizes the proven outbound-only fixed redirect for legacy plaintext
VLESS, requests one shared normal/Smart pipeline, all unique input build checks,
and at most three live representatives per protocol/transport/security family.
Sample success is never transferred to untested endpoints. Existing per-config
production health, ownership, deletion and TTL gates remain independent.

Normal template v7 and Smart v11 / Ads v4 replace P33's listener dependency.
No VPN core or remote VPN server software is changed. The official Xray freedom redirect
keeps the actual endpoint, while VLESS uses the loopback exception with a dedicated
dialerProxy. This adds NO encryption. No relay inbound or exposed port exists.
The first outbound stays a real protocol. Helpers cannot join a balancer or
observer and cannot be shared by user routes. Old bounded listener graphs migrate
through Logical; repeated composition is idempotent. Original source bytes stay
immutable. Credentials, explicit HTTP Host/path, security and flow are retained.
Implicit HTTP authority is made explicit from the original endpoint, according
to the pinned official transport dialers, before its logical address changes.

Qualified stream families: RAW/TCP (optional HTTP header), WS, gRPC, XHTTP,
HTTPUpgrade, mKCP; encrypted variants keep their native settings. VLESS native
encryption is not downgraded. Plaintext finalmask, explicit stream destination,
proxySettings and separate XHTTP download legs remain explicit driver limitations.
Removed H2/legacy QUIC, HY1 and absent plugins are not renamed into another protocol.

The sandbox preflight independently recognizes fixed redirects and examines real
public targets by owner tag, so multiple logical 127.0.0.1:443 members never
collapse into one allowlist entry. The exact original final bytes execute in the
sandbox; no direct shortcut is permitted as health proof. Native group ping uses
the first outbound, not the Smart pool. Actual Android APK/carriers remain not_run.

Schema21 upgrades persisted view predicates and fences old leases before serving.
It preserves source definitions/settings, immutable blobs, retention, queues and
rotation state, and never rewinds the external residential deletion journal.
Old binaries cannot open schema21; use fix-forward or a stopped, matched recovery.

The Output UI displays each candidate's family and marks stale-version proofs as
pending. /api/conversion/families summarizes current identity-level build and
connection results independently. scripts/template-audit.py performs bounded,
deduplicated all-input builds on both pinned cores plus up to three live family
representatives and whole-profile Smart checks. It keeps raw data only in memory
and emits bounded metadata, not credentials, addresses or subscription tokens.
Automatic ingestion/conversion/retest/lease recovery continues through the existing
independent production services. No uploaded executable plugins are run.

Qualification receipts live privately under .local/p35-*; public sanitized reports
belong under docs/reports. Tests distinguish synthetic forwarding, real Internet
HTTPS/DNS/UDP/QUIC/HTTP3, app source contract and native delay from APK testing.
A one-time XHTTP TLS fixture failure and corrupt system race toolchain are retained
as initial failed receipts; final results must identify their separate retests.

## Final evidence

Installed p35-57a1e4adb432b7dcca5c; consistent stopped backup at
/var/backups/config-repair/pre-p35-20261111-024713. All six services and five fresh
worker versions verified. Sources/settings/rotations preserved, old caches fenced.
Final source manifest contains255 exact file hashes and matches deployed artifacts.
Full release check exit0, traffic suite exit0, each final normal/Smart/Splash/Ads
worker exit0, browser restart and live mobile/desktop checks passed. Final workers
use the deployed worker SHA; worker health fixtures are synthetic. Earlier single
race timeout reused one one-second context across SQL work; its second operation
now gets an independent bounded context. No production timeout was enlarged.

The 721-identity snapshot had718 accessible inputs;686 were prepared and accepted,
32 unresolved and3 unavailable.690 distinct final candidates passed both cores.
Normal complete Internet proof28/46; native delay27/46. Smart17/18 built,
8 complete,3 partial,6 failed,1 not run. Final Host alias correction preserved
identical bytes for476 retained inputs; rotating/unavailable records are not
silently requalified. Controlled protocol matrix:232 cases,172 pass,
42 explicit preparation limits,18 expected core rejects. Controlled packet tests
cover HY2/WireGuard and all six SS ciphers, independently of live source inventory.

Fresh exact served JSON canaries in all four scopes passed HTTPS/DoH/DNS TCP+UDP,
UDP443 and HTTP3 on bundled26.10.31, and built on26.10.31/26.11.9. Group exports
returned200 arrays in follow-up; initial Ads warmup503 is retained in the receipt.
Native VLESS TCP HTTP None canary:7ms delay,54ms total call, no local listener.
This is server-side latency; APK/MCI/Irancell remain not_run. Official native
library removes DNS/routing, so strict sandbox denies its attempted direct DNS
for some domain-only probes; successful full sessions are reported separately.

Reports:docs/reports/P35-validation.json, P35-family-results.csv,
P35-capability-matrix.csv, P35-report-fa.md. No raw endpoints, credentials or
subscription tokens are published in these reports. Ongoing automatic per-config
production checks remain stricter than family sampling and control publication.
