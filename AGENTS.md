# Working contract

Read docs/PROJECT_BRAIN.md, PROJECT_STATE.json and HANDOFF.md first. Verify actual git HEAD, branch, status and any runtime receipt before continuing. Read local overrides if later introduced.

## Scope and invariants
- This is an independent Go project. Do not modify the DigitalOcean project, its database, services, proxy paths or credentials.
- Deployment target is the server; shared-hosting portability was explicitly removed from scope.
- Compatibility is pinned to V2RayNG 2.3.6 / Lib v40 / Xray 26.7.28, not latest/master.
- Read pinned upstream source before adding a version-sensitive rule. Preserve reference commits in evidence.
- Current user contract (P3) is immutable raw collection: no byte of collected artifacts may be repaired, rewritten or renamed. Existing CLI repair stays separate and opt-in. Never guess credentials or weaken TLS.
- A JSON conversion can only preserve recoverable information and supported protocol semantics.
- Syntax, Android import, core validation and actual connection are separate statuses. Never claim a later stage passed because an earlier one passed.
- An unverified or unsupported candidate cannot enter a healthy output.
- Raw Output is explicitly unverified. Keep prior membership on empty/error only until fixed retention expiry (default 2 hours from first insertion; repeated fetch/304 does not renew). Nonempty generation replaces membership atomically. Future healthy export requires separate evidence.
- Jobs carry source generation/revision; stale completion cannot reactivate deleted/edited source data.
- Store secrets only in protected runtime storage, never Git, diagnostics or fixtures. Fixtures use synthetic credentials and documentation addresses.

- Passwordless public management is explicitly requested. Retain CSRF/Origin protection and encrypted storage; do not reintroduce accounts without a new request.
- Input / Output / Setting are independent compiled panel modules. Bulk URLs, seconds/minutes intervals, and configurable fixed retention are product requirements.

## Workflow
- Work in this project only; keep the legacy main branch intact during Go development.
- Bounded parallel jobs, explicit deadlines, retry/backoff and durable recovery are required when implementing workers.
- No network/core execution of arbitrary input before sandboxing and egress policy exist. Current core smoke executes only synthetic test fixtures.
- Run scripts/check.sh and relevant integration tests. Record actual results and limitations in docs/checkpoints and PROJECT_STATE.json.
- Test modified behavior and negative cases. For repairs, include unchanged-byte, ambiguity, idempotence and regression tests.
- Commit/publish tested source; fetch/read back the exact published revision. Do not infer deployment from a commit.
- No force-push or default-branch replacement as a routine checkpoint operation.
- Android E2E requires an actual APK/device/emulator receipt. Missing infrastructure is a reported blocker, not a successful skip.
- Final installer must be reproducible, version/checksum pinned, idempotent, isolated to this project, and verified with install/update/rollback tests before release.

## Continuity
Update state, handoff and checkpoint together. Keep source review, test and runtime receipts separate; do not put a commit's own hash inside the same commit. No broad reread of unrelated projects is necessary.
- Bulk action acceptance must use isolated data; adding delete controls does not authorize deleting the user's current data. Source batches are revision-checked transactions. Config delete/disable controls expire with the original retention window.
- Panel timeouts are the actual per-request deadline, including waiting for headers/body; no shorter hidden HTTP timeout. Zero per-source override inherits the shared setting.
- Content inspection drivers label raw content without modification. Readable syntax is not core/import/network acceptance.
