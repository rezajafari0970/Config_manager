# P33 deployed client compatibility — 2026-11-11

Parent: d2d2f5c2d6dd8538466fcd5a1fd7abc379a1bb47.
Branch: go/v2rayng-2.3.6; main remains untouched.
Release: /opt/config-repair/releases/p33-abfd54f0aee4660e4146-r1; schema20; 33 source controls preserved.
Panel SHA256: abfd54f0aee4660e41467e6bf5f02cda38982ec81a825eea6595ca76681c5b9f
Worker SHA256: 34b27143296362529b194968a8910cb37386210d3e31a4499426dcc62f4fca8b

Read docs/CLIENT_COMPATIBILITY.md, docs/reports/P33-report-fa.md,
P33-source-pins.json, P33-validation.json and P33-capability-matrix.csv.
The private stopped-service backup is named in .local/p33-deployed.json.
Its encrypted vault uses immutable hardlinks, while DB/key/binaries are copies.
Never edit ciphertext in place or rewind the independent residential revocation ledger.

Target: official v2rayNG2.3.15, Lib40, bundled Xray26.10.31.
Both26.10.31 and26.11.9 validate exact final output; actual connection runs bundled26.10.31.
Normal, Smart, Splash, Ads, per-source and classified client exports share current
evidence gates. Raw stays exact. Old caches/leases are fenced before serving.
Native group subscriptions are arrays; legacy JSONL stays available.
Published profile bytes never change after validation; profile rotation persists.

Qualified legacy VLESS TCP/RAW HTTP relay preserves endpoint/credentials and adds
NO encryption. Tags/ports are allocated per complete profile, routing is fixed
destination allow then deny, and Ads carrier options stay on the remote leg.
Other protocols and transports use their own capability/driver rules.
Socket DNS forced strategies prevent fallback to the system resolver, including
XHTTP download legs; global DNS queryStrategy remains UseIPv4.
New empty source insertion no longer rescans all conversion/Smart views. Edits and
arriving data still reconcile atomically; bulk1003 regression passes.

Validation: whole-project go vet/race/build; Python/JS tests;184 build matrix rows;
20 protocol traffic plus2 WireGuard cases and relay fixtures on both cores;
real normal/REALITY/Smart HTTPS/DoH/DNS TCP+UDP/QUIC VN and full HTTP3;
zero system-DNS fallback attempts in startup/cold failure;
all four queue SIGKILL recoveries,55 Smart/Splash/Ads profiles each;
browser/restart, live-DB migration rehearsal and deployed browser/API checks.
Queue health receipts are explicitly synthetic; they never certify network traffic.

Limits: actual Android APK, package-to-UID rules, carrier and long-term tests are
not run. Lib40 RealPing strips relay inbounds/routing/DNS and may fail while full
VPN works; the official source-built helper reproduced this limitation.
TC81 was unreachable here and is not healthy. Ads keeps the deliberate
geosite:google@ads dependency; Android asset availability needs device verification.
Build acceptance is not connectivity; QUIC VN alone is not HTTP3. Individual
Smart paths and all possible source combinations are not universally certified.

Runtime evidence: .local/p33-deployed.json, p33-live-check.json,
p33-live-browser.json, p33-tested-runtime.json.
Publication readback and canonical runtime hash receipt: .local/p33-published.json.
