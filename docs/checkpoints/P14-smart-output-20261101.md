# P14 — Independent Smart Output

## Behavior

Smart Output is an additional module at /smart, linked from Output. Setting controls enabled state and member count (default10,50 tested). It combines existing accepted candidates in the requested @shenvpn-smart-parallel-v6b template; sample endpoints/credentials are never added.

One cached profile per canonical input places it first once. Followers are shuffled and chosen with durable per-source/member least-use counters. Each eligible source is covered when slots allow; otherwise sources rotate. Tiny sources must reuse members to keep their per-profile presence, while first-position rounds are independent. Shared canonical/identical configs use a single slot. Profiles may be shorter than requested until enough distinct inputs are available.

Nested balancer/chain graphs retain renamed dependencies, internal routing and pinned cost matching. A real-proxy fallback avoids loopback recursion and never falls back to direct. The base keeps UDP enabled and tcp,udp routing; leastLoad and burstObservatory match the supplied policy. This is connection balancing, not bandwidth bonding.

## Cache and isolation

Schema9 adds only Smart settings, heads/members, usage, heartbeat and rotation. Exact source-specific acceptance/proof, disabled controls and shortest member TTL gate every export. Replaced/expired inputs disappear immediately; capacity/source growth rebuilds underfilled profiles. Ordinary/source URLs, settings and turns remain intact.

The independent config-repair-smart.service has2slots, a separate lock, Unix routes and csm sandbox resources. It validates each exact composite with pinned Xray26.7.28. Test resources use cst; normal conversion remains crw/ctw. Invalid ownership revokes leases immediately before capacity counting; actual crashes recover on the120slease. No new permanent laboratory bank is created.

The full link prints compact JSONL inline. The single link shares one persistent Smart round across clients; HEAD/prefetch do not consume. New heads join the next snapshot, pending heads can return204 rather than repeat early, and source removal/expiry prunes unavailable heads. Failed HTTP delivery after atomic allocation cannot provide exactly-once delivery.

## Verification

- Full formatting/module/vet/race/build checks and11Python sandbox regressions passed.
- Exact pinned core accepted protocol/composite and10/50-member fixtures; original bytes, credentials, dependencies and substring/regexp auto-cost semantics preserved.
- Synthetic leastLoad test skipped a dead first endpoint and forwarded TCP/UDP/UDP443 through the observed second endpoint. Actual QUIC/HTTP3 and Android/TUN remain not_run.
- Isolated worker: 59profiles,50members/profile,4sources,4current accepted canaries, maximum2liveclaims, SIGKILL recovery and complete unique-head round passed. Composite connection testing is separate from member evidence.
- Synthetic browser passed mobile/desktop, inline full/single, count50 setting, restart/TTL and forced DOM replacement during JSON fetch.
- Public TLS browser passed 14single allocations including12parallel requests, zero duplicate heads, no download events/overflow/JS errors.
- Production audit at 2026-11-01T05:53:27.499939+00:00: 13ready profiles exported,10members each, zero duplicates; current available sources including Mixo covered. Counts are moving snapshots while cache builds and sources change.
- Isolated SQL performance with957heads: ready count0.034s; invalidation0.036s. Shared queries materialize current offers and indexed union coverage rather than repeated source scans.

## Deployment

Release: /opt/config-repair/releases/p14-3ab4b9d7c8d076f24092
Panel SHA256: 3ab4b9d7c8d076f24092268e0e3fcc37c98db04d08e54774f31ccaa39b3ff42c
Worker SHA256: 02b45d3936d719b163cb601eaea9bff1fafc9aa6a01a40d4f23b35b1fbab921c
Sandbox driver SHA256: c979512fb5ecd21e2cf97b0d73169aa338684b1cbdd40ab36a325b7b149199de
Adapter SHA256: 875088ecbd2f5f1c3b5b847b2d760014ef5e67ae0ec9d07d8677032298231c12
Smart adapter SHA256: 315174552cdac75a509eb3fa45079dacda57e8e6d60eb966cb60f2ba5b3d0190

Stopped-service migration backup: /var/backups/config-repair/pre-smart-20261101-054522
Latest same-schema backup: /var/backups/config-repair/pre-smart-revoke-20261101-055223
All10source definitions/revisions, settings and ordinary rotation fingerprints matched through migration8→9. Three services are active/enabled; current process hashes are independently checked. The follow-up fix also preserved Smart settings and rotations. Additive failure downgrade removes only new Smart objects before version8; never run the old binary directly against schema9. Final reproducible installer release remains a separate task.

Private receipts: smart-check.log, smart-browser.log, smart-worker-test.json, smart-normal-worker.log, smart-traffic.log, smart-bench.log, smart-live.json, smart-public.json, smart-deployed.json, smart-published.json. Retain failed attempt receipts; do not publish configs, source URLs, DB, keys or raw logs.
