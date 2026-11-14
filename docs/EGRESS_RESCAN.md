# Durable Residential rescan / schema33

A user-requested reset starts one finite cohort of nondeleted, unpaused raw SHA256
inputs. It includes retained accepted inputs and the durable investigation archive.
Manual identity exclusions and confirmed Iran blocks are excluded; quarantined Iran
identities remain quarantined and keep the independent spaced-confirmation policy.
Newly ingested inputs use the normal queue and are not silently added to this cohort.

The public CSRF-protected POST /api/egress/rescan accepts the last run id as
`revision`. A stale revision rejects a duplicate or competing reset. The reset is
one writer transaction: create run and members, expire route/health/country receipt
validity, fence leases by increasing revisions, clear the current tested clock,
schedule all cohort members, and reset bounded retry progression. It does not clear
raw data, prior diagnostic reports, lifetime attempts, first_completed, operational
counters, source controls, country choices, manual exclusions or Iran evidence.
Schema33 also fences already excluded alternate raw versions and applies the same
identity fence on subsequent exclusion inserts/updates. The directly revoked member
keeps its own reason and health-deletion accounting; siblings are identity_excluded.
Reports and lifetime attempt counters remain historical.
Migration itself never initiates a reset. The operator/user explicitly starts it.

Accepted inputs retain their deep raw and template check and existing queue priority.
The first assessment of other members uses the normal fast profile despite historical
attempt counters; inconclusive results retain the existing deep follow-up. Completion
counts once per raw hash, after current lease/revision validation, in the same
transaction as the result and Iran guard. A revoked pending sibling is excluded,
not counted as tested. A measured member that triggers an Iran revocation is counted
as completed. A crash cannot commit half a result or reset; leases recover as before.

GET /api/egress exposes rescan separately from lifetime and rolling throughput:
total = pending + completed + excluded. Previously accepted coverage, conclusive
first results, paused pending members, elapsed time, the first 1000 completions'
elapsed time and completions by 600 seconds are explicit. UNKNOWN and UNSUPPORTED
count as assessed, never as conclusive. The cohort and its receipts survive restarts
and rolling event cleanup. A later explicit reset supersedes the current cohort;
previous cohort receipts remain historical.

## Bounded throughput changes

For retained members, general 64KiB upload/download diagnostics and route observations
run concurrently in the same exact runtime. Each child keeps its deadline and strict
TLS. Both are joined before the core stops or its sandbox moves to another variant.
Route observations still have their 12-connection bound; the independent diagnostic
uses at most four additional transfers. Its exception cannot discard route/IR proof.
All required source observations, paired waves, raw/template validation, country
consensus and Iran gates remain. The auxiliary payload is diagnostic, not route proof.

A healthy Ads exchange with incomplete route visibility renews on the same measured
advance schedule used for accepted publication, capped by the existing ten-minute
Iran interval. It no longer triggers a futile 30/60/120-second retry ramp. Unknown
health and credible unhealthy evidence retain bounded early retries; sticky Iran
follow-ups retain their own stronger schedule.

Default maximum slots is min(31, 4 * CPU count), 16 on this host. Existing memory,
CPU and I/O admission reduction and gradual recovery remain authoritative. Sandbox
memory/CPU/process ceilings, systemd watchdog and restart remain unchanged. This is
a concurrency ceiling, not a capacity guarantee; an actual run must measure the
1000/600-second target. The isolated benchmark includes real previously accepted
inputs first, then current inputs, then archive, with full country drivers. Live
measurement of the explicit cohort is the production result.

## Limits

This change does not add an attributable Telegram/Instagram source observer.
Their route can still remain UNKNOWN under the strict v6 criterion, and zero
publishable results remains possible. Reachability is not source evidence.
Finite checks do not certify all domains, QUIC, DNS, IPv6, future routes or uptime.
Never describe 1000 first assessments as 1000 complete positive classifications.

Deployment requires the shared lock, a stopped matched DB/blob/key/release/units
backup, schema rehearsal, race/vet/build/Python/JS checks, worker/crash/browser
validation, live accounting reconciliation and six binary checks. Do not run a
schema32 binary on schema33 or rewind the external deletion journal.
