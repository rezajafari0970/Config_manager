# P55 Residential Ads operations

This release hardens the existing finite Ads split detector. It preserves route
ads-egress.v5, health v1, country v2, exact raw bytes, pinned cores, all normal and
Smart templates, and ANY selected-country publication. It does not redefine
Residential as a household ISP or infer missing evidence with an AI model.

## Independent modules

| Module | Responsibility |
| --- | --- |
| worker/resources.go | Memory, CPU PSI and I/O PSI admission governor |
| worker/lanes.go | First assessments and bounded follow-up share at every positive capacity |
| worker/watchdog.go | Optional systemd readiness and coordinator watchdog driver |
| panel/egress_ingestion.go | Migration-time recovery of committed staged inputs |
| panel/egress_operations.go | Schema30, durable operational counters and retry policy |
| panel/egress_snapshot.go | One timestamp for all eligibility views in a read snapshot |
| panel/web/egress-evidence.js | Bounded on-demand evidence loading independent of summary polling |
| existing protocol, country and health registries | Isolated versioned drivers with authoritative validation |

Here worker means cmd/config-egress-worker and panel means internal/panel.
No model or external AI service is a runtime dependency.

## Scheduling and recovery

Configured concurrency remains a ceiling, never a promise to consume every slot.
The governor samples every5seconds. Low available RAM and CPU/I/O pressure reduce
new admissions; extreme memory/I/O pressure temporarily pauses admissions.
Already claimed jobs retain their bounded deadlines and lease fencing. Recovery
adds at most one slot per10seconds. Private network/cgroup isolation remains.

At least one follow-up slot is reserved when capacity is2or more; a lone slot
rotates every fifth successful claim. Due follow-ups are ordered fairly, with
urgent accepted proofs and manual requests taking precedence. Empty lanes steal.
Ordinary repeated inconclusive checks back off from5minutes to1hour; unsupported
drivers back off from1hour to6hours. First deep retries and credible unhealthy
rounds remain prompt. Accepted health checks retain the panel-selected interval.
Backoff changes scheduling, not the classification or raw retention.

The service uses Type=notify, WatchdogSec=90, WatchdogSignal=SIGKILL, Restart=always
and RestartSec=10. The coordinator emits watchdog messages while progressing.
A hung coordinator is killed with its service group; startup cleans the dedicated
sandbox scope, and expired database leases are recoverable. Driver exchanges,
renewals and subprocess cleanup have bounded deadlines. This is recovery from
covered faults, not guaranteed uninterrupted service under arbitrary host failure.
The private fault test uses a shorter10second watchdog; its measured recovery
must not be advertised as the production90second watchdog deadline.

Completion and renewal acquire the database writer reservation before checking
lease expiry. Completion re-evaluates evidence freshness against that clock.
Replacement tokens and revisions continue to reject stale writers. Missing or
corrupt raw storage produces input_unavailable/UNKNOWN and remains referenced.
A new sandbox OOM event produces local_resource_pressure/UNKNOWN; it cannot count
as evidence that a remote configuration is unhealthy.

## Durable ingestion and statistics

Schema30 also backfills committed staged rows before source recovery can discard
old staging. The pre-v28 migration uses the same backfill. Exclusions and permanent
deletions win; only ordinary retention/deletion reasons may be revived.
Previously garbage-collected bytes cannot be reconstructed by this migration.

Operational counters are updated in the same transaction as the associated
queue effect. Duplicate or expired completion receipts cannot increment them.
Counters survive process restarts and24hour event cleanup:
registered new hashes, committed completions, unique first assessments,
definitive completions, recovered leases and confirmed unhealthy removals.
The baseline and counters' start timestamp are explicit. Earlier discarded
historical events are not fabricated as lifetime totals.

Rolling5/10minute throughput counts actual completion events, with distinct
configuration hashes and definitive results reported separately. State totals,
country counts and the displayed entries share one database snapshot and one
eligibility timestamp. Errors are returned instead of silently becoming zero.

GET /api/egress?brief=1 omits heavy report evidence from periodic summaries.
The existing default API retains full reports. A validated hash query selects
one diagnostic row for on-demand evidence. The browser permits two evidence
requests concurrently, a bounded64request queue and a32receipt cache. Responses
are bound to hash/tested/revision and cannot overwrite a newer row. Open details
survive summary refreshes. Stale worker heartbeats do not display old active work.

## Publication and retention

All four outputs use the same fresh route+payload+country eligibility and ANY
country intersection. Empty selection publishes nothing. A TW+MX+CY member
qualifies with TW or CY selected even if MX is not selected; it appears once.
Filtering selects whole configs and does not force their actual future exit.
The X-Country-Policy header now describes the same ANY policy.

Accepted and unresolved raw inputs remain exempt from ordinary retention.
Existing three-round credible unhealthy removal and host outage circuit remain.
Unknown routes, unavailable country providers, host pressure and missing storage
do not become fabricated negative verdicts or global deletion. Fresh evidence is
still required to publish. Finite storage can pause acquisition safely; it cannot
provide an unlimited archive or guarantee that an unavailable upstream returns.

## Validation and deployment

Run scripts/check.sh using the pinned toolchain. It verifies modules, formatting,
vet, the full Go race suite, six binaries and Python/JS contracts. Migration,
writer-contention, fair lanes, exact-once counters, expiry snapshots, raw
references, local pressure, brief reports and ANY outputs have regression tests.

Isolated integration scripts:
- scripts/egress-worker-test.py: real worker, private fixture, crash recovery and browser.
- scripts/egress-watchdog-test.py: SIGSTOP against its own transient unit.
- scripts/egress-throughput-test.py: unique retained real inputs, bounded workers,
  optional trusted country manifest, and explicit conclusive/unknown counts.

Run sandbox integration scripts sequentially; they share the test-only cgt scope.
The authenticated blob audit uses EGRESS_BLOB_AUDIT_CONFIG with the trusted local
panel configuration; it reports aggregate integrity results, not credentials.
A1000-input assessment benchmark includes UNKNOWN and UNSUPPORTED outcomes and
must never be described as1000 certain residential classifications.

Hold /var/lock/config-repair-deploy.lock. Verify canonical HEAD and clean worktree.
Stop the six matched services, checkpoint SQLite, and save a matched DB/blob/key/
release and egress-unit backup. Activate matching six binaries, install the
watchdog unit, daemon-reload, migrate29to30 and compare source/settings/country/
health controls. Verify all service executable hashes, API, coverage and outputs.
Do not run a pre30 application on schema30 or roll back only one component.
