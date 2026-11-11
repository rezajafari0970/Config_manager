# P34 acquisition recovery — 2026-11-11

Parent:4371c3dbacc305e48a3a5e25ac45f058bc57867d.
Branch:go/v2rayng-2.3.6. Legacy main remains unchanged.

## Reproduction

The failing source returned HTTP200,31KB and186 occurrences in1.308s; standalone
extraction took<1ms and found148 unique identities in that particular response.
The running collector instead failed at30s repeatedly and had0current memberships.
Another source fetched62 occurrences/31identities in204ms but historical collector
processing took17.466s. A97-occurrence source fetched in22ms but took21.572s inside
the collector. These are independent snapshots; source bodies can rotate.

The old timeout covered network, vault fsync, inspection and every per-item FULL
SQLite transaction. GC additionally waited on the serialized writer while holding
the vault publication mutex and repeatedly evaluated an eligibility view plus an
unindexed retained-record candidate lookup. This combined path mislabelled local
processing timeouts as upstream failures.

## Change

- Network timeout ends only after clean full-body EOF into an immediately unlinked
  spool. Processing has its own120s context, final commit30s, lease30s extra margin.
  Existing per-source override and cancellation remain effective.
- StageBatch is an optional repository driver. Batches hold at most32 records,
  discard in-memory candidate payloads and retain counted pins until commit. Old
  drivers retain Stage fallback. Complete alone atomically publishes a generation.
- Exact healthy duplicates reuse verified encrypted files without repeated fsync/
  rename. Corruption is still healed through atomic replacement. Underlying read
  errors, empty ciphertext and chunk-boundary truncation fail explicitly.
- Candidate-reference indexes are additive/idempotent on schema20. GC protects
  retained conversion result rows until ordinary transactional reconciliation;
  it no longer runs the entire publication eligibility view per encrypted blob.
  On a private snapshot,63reference checks fell from109.16ms to2.92ms with identical
  decisions. This is DB cost evidence, not a universal end-to-end speed guarantee.
- UI separates fetch_timeout, processing_timeout and incomplete_body. Existing
  retained inputs remain visible when a newer fetch fails. No failed generation
  or old core/connection evidence becomes a healthy output.

## Validation / release

Regression fixtures cover62/97/186items, delayed storage beyond the network
budget, bounded batches, deduplication, pin cleanup, cancellation, rollback,
stale source revisions, incomplete network bodies and authenticated storage EOF.
Existing expiry/304/recovery, source isolation, encryption, core/Smart and browser
checks remain required. The full race run started before one old test was updated to expect
`fetch_timeout` instead of `timeout`; that test-only assertion was updated and
passed independently, with every non-test runtime file unchanged. Preserve both
logs and the combined validation receipt; do not call the superseded run green. Final results and runtime identity are recorded in
.local/p34-check-final.log, p34-browser.log, p34-workers.log, p34-deployed.json,
p34-live-acquisition.json, p34-live-check.json and p34-published.json.

OpenAI gpt-6-astra review: first call reached its output budget; second completed
and found the storage EOF bug, now covered by a regression. Estimated total API
cost:$0.59533; actual billed amount is not available. No source URLs, credentials
or raw configurations were included. No runtime AI dependency was introduced.

## Limits

An upstream observed returning HTTP200 with zero bytes remains empty. Prematurely
closed responses must be retried; partial content is not complete acquisition.
Fixed retention, rotating-source generation semantics and all current health
admission gates stay unchanged. No promise of permanent upstream availability,
unbounded input processing, Android/carrier success or all-server connectivity.
The source's current unique input count is distinct from occurrences, internal
outbounds in a whole Smart JSON, and connection-qualified exports.

## P34 deployed evidence

Installed release: p34-25bcb3a7bb488e1434d7 (schema20). All six services and five
fresh worker heartbeats were checked after activation; PID executable hashes
match the tested binaries. Source controls, rotations and publication gates
were preserved. All33 sources completed a fresh collection:32 have inventory,
with724 unique inputs at the verification snapshot. The formerly empty worker
source succeeded three times:186 occurrences each,142/151/149 unique inputs,
12.289/3.639/2.848s. Rotating content explains differing unique counts.
Exir still returned an empty body; sub returned incomplete_body and retained
its161 previous inputs. Neither is relabelled a successful fresh import.
Live mobile/desktop rendering verified the recovered count. Final exact-export
build checks and existing real-runtime proof checks are in P34-validation.json.
API review used gpt-6-astra; no ongoing AI dependency was added.
Publication and canonical rebuild identity are recorded privately in
.local/p34-published.json; main remains unchanged.
