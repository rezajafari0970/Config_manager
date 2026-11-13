# Residential accounting and renewal

`internal/panel/egress_accounting.go` owns the presentation accounting contract.
It uses the same committed read snapshot and frozen eligibility clock as country
counts and entries. Counts are computed from authoritative rows, not incremented
in browser memory and not reconstructed from the rolling event log.

## Units and reconciliation

* A **raw version** is one immutable SHA256 input. Different labels/serialization
  may produce separate raw versions while preserving the same collector identity.
* An **identity** is the existing collector fingerprint, with its existing
  normalization scope. It is not an assertion that all formats are equivalent.
* A **distinct template** is a candidate content hash. Multiple raw versions may
  emit identical template bytes. Export rows remain unchanged and are not silently
  deduplicated or deleted to make a smaller counter.
* A **completed attempt** is one committed completion. Repeated successful probes
  increase attempts but not distinct assessed/conclusive inputs in that window.

`live_versions = current_live_versions + archive_versions`.
`live_versions = awaiting_first + assessed_versions`.
`live_versions = queue.paused + queue.running + queue.due + queue.scheduled`.
These four queue states are mutually exclusive. A running first assessment still
belongs to the separate `awaiting_first` inventory facet until it commits.
This facet uses `tested=0` in the current lifecycle; revival may reset that field
without erasing its older lifetime first-assessment counter.
An expired lease is recoverable; it is never counted as an active running lease.

Current source versions and identities are counted independently of the durable
investigation archive. Staging is shown separately and may overlap memberships.
Coverage checks include both memberships and staging, excluding intentionally
suppressed identities. They measure registration, not blob integrity; the latter
requires the independent authenticated full-inventory audit.

`retained = eligible + awaiting_proof` for accepted live inputs.
`eligible = country_unknown + country_filtered + published`.
`published` is the count of eligible raw-version rows after the saved country
policy. The API also returns their distinct candidate/identity counts. Country
totals overlap for multicountry inputs and must not be added into a config total.
The existing ANY country intersection and stronger sticky Iran veto remain.

Stored job state is the last classification, not proof of current eligibility.
An expired health receipt is explicitly labelled expired in diagnostics. The
summary displays fresh eligible and publishable counts and a separate count of
retained accepted inputs awaiting proof. `accounting.at`/`now` are server times;
worker freshness does not depend on the handset clock.

Lifetime operational counters retain their original baseline/start timestamp and
transactional updates. No counter is reset. The rolling throughput API now adds
`unique_definitive_5m` and `unique_definitive_10m`; old attempt counters remain for
compatibility. A conclusive result in a rolling window is historical throughput,
not a guarantee that the input still qualifies for publication now.

## Renewal and database work

Accepted probes already calculate an advance start time from measured duration.
Both claim lanes now honor that `next_run`, instead of restricting most slots to
the final minute before expiry. Existing lease fencing, bounded admission,
inconclusive backoff, country/health validation and ten-minute Iran expiry remain.
Due accepted work has priority; no extra concurrent probes or resource ceilings
are introduced. This cannot guarantee deadlines if required work exceeds capacity.

The additive `egress_schedule_cover` index keeps claim/counter metadata separate
from large report pages. Schema31 and evidence epochs are unchanged. The index
is installed idempotently at startup; no input or proof reset is needed.

Bounded blob sweeps resume after the last attempted directory entry and wrap
once, so repeated cleanup deadlines or a failing reference query cannot starve
the tail. Progress is in memory; restart safely rechecks from the beginning.
Sweeps are serialized independently of publication pins. Current writer-database
reference checks and pin locking remain authoritative before every deletion;
query failures preserve the affected file. A cursor is not deletion authority.

Run `scripts/check.sh`, the private worker/browser integration, the read-only
authenticated blob audit, and live API/output reconciliation. Deployment retains
the shared lock and matched stopped DB/blob/key/release backup. Verify the actual
six binaries, preserved controls, and runtime behavior after activation. Do not
turn a finite observation window into a zero-bug or permanent-uptime claim.
