> Current v6/schema32 behavior: see [ADS_EXCLUSIVE.md](ADS_EXCLUSIVE.md). The historical policy below is superseded where it conflicts with Ads-only health, independent controls, preserved history and multi-attempt IR evidence.

# DoubleClick Iran guard — P56

`doubleclick-iran.v1` is a deny policy independent of the Residential split-route
classifier, payload health and the ANY selected-country filter. A TW+IR member
is withheld even when TW is selected. TW+MX+CY retains the existing ANY behavior.

## Evidence and state

The existing pinned runtime probes exact raw/template configurations. A signal
must belong to a validated variant/core and a fresh, hash-bound job report; its
TLS-authenticated same-stream peer observation must concern one of the three
explicit DoubleClick domains. The existing country driver registry supplies
fresh provider facts: at least two agreeing independent lineages and a strict
majority. Mirrors cannot vote twice. The country consensus validator is shared
with publication, without requiring a complete split-route result for a deny.
Resource-pressure receipts, broken TLS, generic control IPs, unknown or expired
GeoIP data and incomplete source attribution cannot create an Iran signal.

First valid IR observation atomically creates a sticky identity quarantine.
Ordinary, Smart and all four Residential exports use the same input/identity
veto; source response caches are invalidated and rotation revalidates eligibility
under the writer lock. Existing clients' downloaded configurations cannot be
revoked by a subscription server; clients must refresh their subscriptions.

Two positive rounds are required for automatic global revocation. The next
round must start at least60seconds after the previous round finished. Samples,
waves, duplicate acknowledgments and concurrent renamed aliases do not count
as independent rounds. Two early follow-ups are scheduled60seconds apart; after
those, unresolved quarantines remain on the bounded ten-minute monitoring cycle.
A clean or failed test never erases prior valid IR. Confirmed identities remain
blocked until an explicit operator policy change; there is no automatic reset.

`egress_iran_guard` stores minimal first/latest provider evidence, positive-round
count and timestamps, without plaintext exit IPs or configuration credentials.
It survives cleanup and restart. `input_exclusions` uses the collector's existing
semantic fingerprints: URI fragments, VMess ps and JSON remarks/key order do not
bypass revocation. It does not blacklist entire servers or sources, and does not
claim to recognize every semantically equivalent rewrite across formats.

## Scheduling and recovery

Accepted inputs and quarantines use the existing durable queue, bounded workers,
leases, resource governor, watchdog and reboot-enabled service. Health settings
are preserved. The Iran cycle caps the effective interval at ten minutes;
completion time is reserved using the previous job duration (60–240seconds).
Publication proof is never extended beyond ten minutes after the measured
report finished. Capacity exhaustion is exposed as `overdue_retained`, not
hidden by prolonging proof. Unknown inputs remain retained.

Schema31 creates the policy start marker and identity guard. Historical IR
country observations from the current24hour window cause quarantine only;
they are not counted as new independent positive rounds. Accepted inputs are
rescheduled. Pre-guard route proofs are not published until checked by the new
policy. Country selection/revision, source settings and the health interval
remain unchanged. Ordinary retention still cannot erase retained accepted inputs.

## Operational limits

This is finite TCP evidence for three DoubleClick domains and observed IP
families, not proof of every AdMob request, IPv6-only route or UDP/QUIC session.
GeoIP remains an estimate. A route can change between probes. Guard activation
cannot recall already downloaded client configs or stop an existing connection.
No model or external AI call is required in the runtime decision path.

The panel exposes quarantine identities, permanent blocked identities and
retained jobs overdue ten minutes. `X-Ads-Iran-Guard` identifies the export
policy. Counts of identities and raw representations are deliberately distinct.

Deploy only after `scripts/check.sh`, private migration and crash/browser checks.
Hold the shared deployment lock and take a stopped, matched DB/blob/key/release
backup. All six executable hashes must match. Never run pre31 binaries on
schema31 or restore a DB that silently drops later revocations; fix forward.
