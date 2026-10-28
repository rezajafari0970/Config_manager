# P0 foundation receipt

Target: V2RayNG 2.3.6 / Lib v40 / Xray 26.7.28. Date label uses Asia/Tehran.

## Completed
- Separate Go branch and server working directory; previous project/default branch preserved.
- Nine-phase roadmap, architecture decisions, project continuity and working contract.
- Source review pinned to exact V2RayNG and Xray commits; initial findings recorded with references.
- Official Linux amd64 Xray asset and executable verified by SHA-256 before execution.
- Conservative JSON diagnosis and the first auditable repair; unchanged inputs remain byte-identical.
- Random independent schedule helper and source replacement/fallback/stale-job policy tests.
- Full Go formatting/vet/race/build gate passed using isolated official Go 1.27.1.
- 13/13 controlled Xray schema cases passed. No live third-party proxy connection was attempted.

## Evidence
- `core-smoke.json`: individual core-test results and pinned binary identity.
- `android-preflight.json`: no Android test device/emulator available; not a test pass.
- `toolchain.json`: official download checksum and identified host object mismatch.

## Explicitly incomplete
Full compatibility matrix, management UI, source storage/fetch workers, other repairs, URI-to-JSON conversion, sandbox/live probe, Android E2E, subscription publication and final installer. See `PROJECT_STATE.json` and roadmap. No public management listener was opened.
