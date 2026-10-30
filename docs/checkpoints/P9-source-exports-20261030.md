# P9 — independent per-source exports

User request: keep configs attributed to each input link and provide both full and single-config output for that source, alongside the global exports.

## Changes

- Added source export metadata/routes with source/mode-bound tokens, Input links, a searchable paginated Output source list and `/output?source={id}`. Individual JSON view/copy is scoped too. Both global URLs remain unchanged.
- Source membership is filtered inside the conversion CTE before canonical identity ranking. A valid candidate/proof from one representation cannot be borrowed by a different source merely because it has the same canonical identity. Shared exact configs remain eligible in each owning source, and deduplicated globally. Original storage/memberships/lifetimes are unchanged.
- Schema8 scopes the round tables by source, preserves the existing global cycle and every remaining queue item, and cascades only the deleted source's state. Cancellable reference-counted per-scope locks replace the single in-process lock; SQLite transactions retain cross-process atomicity. No vault I/O holds the DB connection.
- Each scope snapshots its own unique eligible candidates and wraps independently. New arrivals join its next round; replacement, disable/delete, expiry and invalid proof remove eligibility. Deleted sources return404; disabled/empty source full output is empty and single output is204. Inline text/one-JSON-per-line and HEAD/prefetch behavior are preserved.

## Verification

- Full formatting, module integrity, vet, race tests, four binaries, pinned-core matrix and9Python sandbox regressions passed.
- Source tests prove overlapping ownership, filtering before identity ranking, cross-source/mode token rejection, exact scoped bytes/copy, independent global/source rounds, replacement/stale claims, disable/delete/FK cascade, independent cancellable locks and lock reclamation.22parallel allocations across two Store instances and three scopes produce two complete distinct rounds in each scope. A schema7 fixture with an active global queue migrates without resetting it.
- Synthetic browser phases cover Input/Output, searchable source list, scoped view/full/single outputs, separate cycle resets and persistence across process restart, inline navigation and mobile/desktop layout. Existing global tests remain passing.
- Isolated worker acceptance passed9samples including4read-only canaries:8core passes,4two-destination connections,8eligible JSON outputs, SIGKILL/lease recovery and maximum3parallel workers.
- Production TLS-verified source output checks covered3sources, exact owned hashes,12parallel single allocations with zero duplicates, and desktop/emulated Android inline navigation without downloads/overflow. No Android app/TUN validation is claimed.

## Deployment

Release: /opt/config-repair/releases/p9-cf488e252ef57e4d49e9

Panel SHA256: cf488e252ef57e4d49e973a0179e46908706cde30abee7d7c40929050757e80e

Worker SHA256: 4dd17163b3607c372990df88701e0b660fc16cbfb151a433045999db445e8acb

The live source set had7definitions before this deployment; definitions/revisions/settings matched afterward. Older54-source counts are historical. Global cycle78and the pending queue were preserved exactly. SQLite quick_check was ok and foreign_key_check empty; both services are enabled/active.

Matching stopped-service backup: /var/backups/config-repair/pre-source-export-20261030-125245. Restore matching data/key/blobs/release for rollback: schema7 binaries cannot open schema8. Do not restore historical sources to match old counts.

Private receipts: `.local/source-export-targeted.log`, `.local/source-export-check.log`, `.local/source-export-browser.log`, `.local/source-export-worker.log`, `.local/source-export-deployed.json`, `.local/source-export-live.json`, `.local/source-export-published.json`. Source publication/readback and process hashes are verified separately. Final reproducible installer remains a separate release task.
