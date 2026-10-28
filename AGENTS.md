# Working contract

Read docs/PROJECT_BRAIN.md, PROJECT_STATE.json and HANDOFF.md first. Verify actual git HEAD, branch, status and any runtime receipt before continuing. Read local overrides if later introduced.

## Scope and invariants
- This is an independent Go project. Do not modify the DigitalOcean project, its database, services, proxy paths or credentials.
- Deployment target is the server; shared-hosting portability was explicitly removed from scope.
- Compatibility is pinned to V2RayNG 2.3.6 / Lib v40 / Xray 26.7.28, not latest/master.
- Read pinned upstream source before adding a version-sensitive rule. Preserve reference commits in evidence.
- Raw healthy artifacts remain byte-identical. Repairs are versioned, minimal, auditable and reversible. Never guess credentials, keys, SNI or other connection semantics, or downgrade TLS to make a test pass.
- A JSON conversion can only preserve recoverable information and supported protocol semantics.
- Syntax, Android import, core validation and actual connection are separate statuses. Never claim a later stage passed because an earlier one passed.
- An unverified or unsupported candidate cannot enter a healthy output.
- Keep previous source generation on empty/failed/unusable fetch only while health evidence remains fresh. A usable new generation replaces previous source membership atomically.
- Jobs carry source generation/revision; stale completion cannot reactivate deleted/edited source data.
- Store secrets only in protected runtime storage, never Git, diagnostics or fixtures. Fixtures use synthetic credentials and documentation addresses.

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
