# Project Brain

Read this file → PROJECT_STATE.json → HANDOFF.md → current checkpoint → relevant source/tests. Current user instruction supersedes P2 authentication and repair-in-collection plans.

## Active contract

An independent Go daemon collects source URLs continuously, without changing config bytes. Passwordless Persian Input / Output / Setting manages bulk URLs, independent randomized seconds/minutes requests, source state, raw downloads and fixed retention (default 2h). A nonempty accepted generation atomically replaces one source's membership; empty/error retains it only until fixed expiry. Repeated fetch/304 cannot renew expiry. Global identity dedup and per-source original representations are separate. Stale jobs cannot revive deleted/edited sources.

Raw Output is not a healthy output. Existing repair CLI and pinned V2RayNG 2.3.6 / Lib40 / Xray26.7.28 compatibility work remain separate. Any future repair creates a distinct auditable candidate, and any healthy export requires core/import/network evidence. No arbitrary raw JSON is executed by the collector.

| Module | Location | State |
|---|---|---|
| Independent collector and format drivers | internal/collect | active; bounded jobs, leases, retries, safe HTTP egress |
| Encrypted immutable blobs | internal/blob | streaming encryption, SHA identity, pins, GC |
| Source/settings/output repository | internal/panel/repository.go | SQLite transactions, revisions, fixed TTL, memberships |
| Input / Output / Setting | internal/panel/*_http.go, web/* | separate route and frontend modules |
| Daemon and supervisor | cmd/config-panel, deploy | shutdown, single DB owner, watchdog/restart |
| Target and initial CLI repair | internal/target, internal/repair | earlier controlled rule, separate from collection |
| Core / Android acceptance | scripts/core-smoke.py, android-preflight.py | controlled core fixtures; Android not_run |

Evidence: compatibility/target.lock.json, docs/compatibility/, docs/checkpoints/. Private operational receipts/toolchains/fixtures are ignored under .local/. Deployment and source publication require independent readback.
