# Project Brain

Read order: this file → `PROJECT_STATE.json` → `HANDOFF.md` → current checkpoint → exact source/tests.

## Product contract

An independent Go daemon repairs configuration representations for V2RayNG 2.3.6 / Lib v40 / Xray 26.7.28. The user manages source URLs through a Persian mobile-friendly panel: add, edit, delete, enable/disable, run now, per-source random intervals and status.

Every source has independently scheduled fetches, immutable generations and membership. A usable new generation replaces that source's previous active membership in one transaction. Empty, failed or wholly unusable fetches preserve the last generation conditionally on renewed health. Health expiry is independent of HTTP 304/last fetch time. Shared artifacts retain independent source membership. Editing/deleting a source invalidates its outstanding jobs.

Preserve raw healthy configuration. Try minimal URI repair before an equivalent full Custom JSON when enough information exists. Never fabricate missing connection data or translate a wire protocol merely by changing format. Hysteria v1 is not supported by the selected Xray Hysteria implementation.

## Modules and ownership

| Module | Location | Current state |
|---|---|---|
| Version identity | internal/target | implemented |
| Driver boundaries | internal/driver | contracts only |
| Interval sampling | internal/schedule | implemented; persistence pending |
| Source generation decisions | internal/lifecycle | pure policy implemented; database integration pending |
| Conservative JSON diagnosis/repair | internal/repair | initial rule only |
| Developer CLI | cmd/config-repair | implemented |
| Core fixture verification | scripts/core-smoke.py | controlled fixtures, no live proxy test |
| Android readiness | scripts/android-preflight.py | preflight only; Android acceptance pending |
| Panel/source store/fetch/probe/exporter | future phases | not implemented |

## Durable evidence

- `compatibility/target.lock.json`: pinned upstream identities and Linux core asset integrity.
- `docs/compatibility/v2rayng-2.3.6.md`: findings from exact source with line references.
- `docs/ROADMAP.md`: ordered parts and acceptance conditions.
- `docs/checkpoints/`: sanitized test receipts and development deltas.
- `.local/`: ignored source clones, binaries, temporary tests and operational receipts.

Production pipeline, sandbox, arbitrary input core execution and Android test acceptance are deliberately not claimed by the initial command-line foundation.
