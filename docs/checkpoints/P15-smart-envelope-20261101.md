# P15 — Exact ordered Smart envelope, complete outbound copying

Date:2026-11-01. Parent:478491b43432b5855a927f7ee7dd21485411bef6. Branch:go/v2rayng-2.3.6. Schema9; normal template-v4 unchanged.

## Contract and implementation

The user requires all outbounds copied and everything else equal to the supplied Smart template, with logging fully off. Earlier P14 global merging violated that exact shape by importing scoped DNS, internal routes/balancers, observers and fallbackTag, and map serialization alphabetized the envelope.

Smart v2 now serializes the constant envelope with only outbounds replaced. Explicit loglevel/access/error none and dnsLog false are supported by pinned Xray infra/conf/log.go at5ca6f4b7d4dc20a881d4330e498892697627ec0c. Fixed global fields/ordering are independently golden-tested. LeastLoad expected remains3 even for one/two entries; pinned core accepts both.

Copy every input outbound with all settings, extension values, arrays and numeric precision. Namespace only tags/references; do not prune unused helpers or inject keepalive. Proxy tags are sequential proxy-01 onward; copied helpers have member-specific tags outside the selector. Template direct/block remain last. Ten ordinary inputs produce42outbounds by design, not a missing-copy error.

Custom balanced pools flatten their complete selected real nodes into the fixed global leastLoad. They retain outbound values, not their old source-global selection policy. Active loopback dependencies needing custom routes fail explicitly with a Persian explanation; otherwise a fixed routing envelope would silently break their graph. Ordinary output keeps its prior independent conversion contract. Original encrypted bytes are unchanged.

Smart version change excludes old cache and revokes old leases before capacity counting. Count settings, source/member fairness and independent durable rotations survive. Export links/tokens, inline text and compact JSONL remain unchanged.

## Validation

- Full modules/vet/race/build and11Python sandbox checks passed.
- Exact envelope/field order, deterministic serialization, complete outbound reverse-reference equality, unknown fields/nulls/large numbers, balanced pools/helpers and missing/cyclic dependencies covered.
- Pinned-core1/2/10/50profiles passed.
- Controlled leastLoad dead-first/live-second fixture forwarded TCP/UDP/UDP443 with shipped logging disabled.
- Browser mobile/desktop/restart/full/single/DOM refresh checks passed. A test synchronization race was fixed by waiting for the asynchronously loaded Smart link.
- Real isolated sandbox worker accepted59profiles with50members, four sources and four current accepted canaries; SIGKILL recovery, two-slot bound and unique first-position round passed.
- Production recomposition from encrypted accepted inputs matched85cached profiles byte-for-byte:850input selections,3400copied outbounds plus170template helpers. No raw values were logged or committed.
- Android APK/TUN, live QUIC/HTTP3 handshake and full composite connectivity are not_run. Core acceptance is separate from endpoint reachability.

## Deployment and rollback

Release:/opt/config-repair/releases/p15-b003f05d4d978a956f0e
Panel SHA256:b003f05d4d978a956f0e4506277efa4992cebea32c88c42bfb3d1bf84afbf884
Worker SHA256:977d48d2ad57d55caea16683f0e90dbe0bdd0dad4c367c506b5ce135604d5d0b
Matching stopped backup:/var/backups/config-repair/pre-smart-envelope-20261101-062348
Previous release:/opt/config-repair/releases/p14-3ab4b9d7c8d076f24092

All10actual sources/settings and normal/Smart rotations preserved; no schema migration. All three services active/enabled with runtime hashes checked. Original normal/source outputs untouched. Old Smart cache rebuilds under v2; rapidly changing sources can invalidate profiles normally. Restore matching prior binaries for same-schema rollback, preserving current user data and rebuilding versioned cache.

Private receipts:p15-deployed.json, p15-audit.json, p15-public.json, p15-published.json and p15-check/browser/worker logs. Publish exact tested source and independently read back every changed file/tree and runtime hash.
