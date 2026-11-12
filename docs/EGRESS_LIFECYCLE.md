# Residential Ads lifecycle (P55 operations; P53 health contract)

Residential is the existing project label for finite repeated Google Ads split
observations. It does not prove household ISP service or every possible route.
Route v5, strict TLS, raw bytes, pinned core fallback and ordinary/Smart templates
are preserved. Health, route and country are independently validated facts.

| Module | Responsibility |
| --- | --- |
| `egress-health.py` | Two bounded independent payload targets through the isolated SOCKS runtime |
| `internal/egress/health.go` | Bind complete transfers to the exact raw/template/runtime/core report |
| `egress_lifecycle.go` | Durable admission, schema28, retention and editable interval policy |
| `egress_store.go` | Lease/revision validation, scheduling, health persistence and confirmed deletion |
| `egress-health.js` | Independent interval editor with revision conflict protection |

Default health interval is15minutes; POST `/api/egress/lifecycle` accepts revision
and health_minutes between1and60. Shortening advances due work immediately;
lengthening does not fabricate longer evidence. The next successful check grants
interval+2minutes for bounded scheduling grace. Network/queue delays can exceed
this grace, in which case publication stops while stored data is retained.
A scheduler cannot guarantee an exact minute under arbitrary outages or load.

A fresh split candidate must pass actual64KiB download AND64KiB upload with TLS
verified, curl exit0, HTTP200 and exact complete byte counts in both raw and
published-template runtimes. Cloudflare speed and HTTPBingo are independent target
plugins; one fully successful pair per variant suffices. An incomplete/timeout
HTTP200 transfer is failure. A successfully started core or ping is insufficient.
This finite small transfer is health evidence, not sustained throughput or an
Android/app performance measurement. The separate Smart payload_v1 contract is
unchanged.

Previously accepted members always receive payload checks in the same runtime as
the route observation. New inputs incur payload cost only when an Ads split remains
possible. Route/country sampling is repeated with the health cycle. A stale,
ambiguous or missing result never becomes a conclusive negative classification.

Automatic global deletion requires three spaced credible unhealthy rounds, at
least30seconds apart. Each round requires BOTH independent targets failing in
BOTH exact runtimes while BOTH complete payload pairs work in a host outage
control. The host control never supplies VPN health or Ads country evidence.
Correlated failures across three members open a bounded five-minute circuit.
Repeated failures do not extend its deadline. A five-minute cooldown after that
window allows fresh confirmed negatives to complete revocation before rearming.
An unknown round resets the destructive streak. A healthy route change withdraws
Residential publication but does not globally delete the config. Manual deletion
still wins. Metadata/country-service failures never count as unhealthy.

The queue protects all successfully staged immutable inputs from ordinary age
cleanup. Metadata is deduplicated per raw hash; reports are bounded/latest-only;
country history and events are bounded to24hours. Blob GC retains referenced raw
and candidates, and deletes unreferenced material using existing fences. Disk
admission guards pause collection under pressure. Finite disk is not an unlimited
archive guarantee, and malformed/truncated upstream content cannot be classified
as an invented valid config. Unsupported/unknown cases remain visible and retry.

All four Residential exports use the same route+payload+country allowlist view,
with final writer-transaction eligibility revalidation. Counts come from a single
read snapshot; saving country selection returns its count atomically. At least one country
observed for a multi-country member during24hours must be selected (ANY). Filters do not
erase stored healthy members. Neither GeoIP nor finite probes guarantee100% country
accuracy, future routes or1000 conclusive classifications in a fixed time budget.

P55 adds schema30 durable counters, fair resource-aware scheduling, watchdog recovery,
upgrade-time staged-input recovery and execution-time lease fencing. See
EGRESS_OPERATIONS.md. Accepted health and publication contracts above remain.
