# Project Brain

Read this file → PROJECT_STATE.json → HANDOFF.md → current checkpoint → relevant source/tests. Current user instruction supersedes P2 authentication and repair-in-collection plans. P3b focuses on acquisition/settings, batch management and honest diagnostics.

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


## P3b invariants

- Source revisions/cancellations isolate bulk management; no old completion can revive deleted sources.
- Expiring config controls isolate disabled/deleted identities from concurrent fetch. Raw bytes are immutable.
- Inspection plugins produce metadata only. Empty, malformed, unrecognized and transport failure are distinct.
- Per-source timeout overrides default in seconds, including actual headers/body wait. Expose effective parallel capacity.
- Live acceptance must preserve current source URLs/settings; destructive scenarios run against synthetic/isolated databases.

## P4a active authorization

The user now requests structural repairs. Collection retains immutable originals and automatically produces separate CandidateDriver results; candidates never replace raw identities or become healthy exports. Schema 4 keys repair metadata/ciphertext by exact raw SHA and links lifetime/access to the original. Strict ambiguity rejection, version-pinned schema rules, unchanged-byte/idempotence regression tests, and separate structural/core/Android/network evidence are mandatory. Input and Output display repair counts and reasons; raw source quality remains visible even when a candidate fixes it.

## P5a temporary test laboratory

The user accepts any network vantage. `internal/probe` / `cmd/config-probe-plan` prepare separate pinned-target candidates. `scripts/live-probe.py` provides a temporary privileged, namespace/cgroup/egress-isolated Linux validation and HTTPS test driver. `.local/probe-bank/evidence.db` persists hashes, diagnosis, expiry and separate stage evidence; no raw secrets. `compatibility/probe-knowledge.json` records evidence-backed repair strategies. Actual Android acceptance remains not_run. This is not the permanent collector probe worker, and the panel daemon/schema remain at P4a. Read docs/probe-lab.md and the P5a checkpoint before promoting or extending it.
