# P8 — durable single-config output

User request: add a separate link which prints one config per request, with every config allocated once before restarting the list. The all-config JSONL link remains unchanged.

## Contract and implementation

- Added `rotationModule`, the `/subscription/{token}/next.json` route and `single_export_url`. Both links are visible/copyable in Output. Response is one exact compact JSON plus newline, inline UTF-8 plain text, no-cache. No eligible data produces an empty204.
- Schema7 holds only a global cycle counter and the pending hashes of the current snapshot. New arrivals join the next cycle; removed, disabled, expired or no-longer-accepted candidates are skipped. Candidate-hash uniqueness matches the all-config export. The finite snapshot prevents endless arrivals from starving a cycle reset.
- SQLite write locks and final cycle/head/eligibility checks prevent duplicate allocation even across distinct Store instances. A local mutex reduces contention. Exact blob validation runs outside DB transactions to avoid vault/GC lock inversion. Broken blobs trigger existing regeneration. HEAD/prefetch/invalid tokens and the all-config link do not consume turns.
- Allocation is durable before writing the response; a disconnected client can lose an allocated response. This is not an acknowledgement protocol or a claim of guaranteed delivery. The queue survives process/server restart; no historical config/repair bank is introduced.

## Verification

- Full gofmt/module verification/vet/race tests, four builds, pinned-core template matrix and nine Python sandbox regressions passed.
- Dedicated tests cover migration6 to7, exact bytes, two complete sequential rounds with reopen,25concurrent requests across two independent Store instances producing five full distinct rounds, concurrent garbage collection, new arrival deferral, disable/delete/expiry, stale claims, missing blobs, cancellation, non-consuming requests and empty204.
- Existing browser fixture now navigates both links and verifies zero downloads, single JSON, automatic reset and durable position across process restart. Mobile/desktop layout and expiry checks passed.
- Isolated sandbox worker with4read-only canaries/9total samples passed SIGKILL/lease recovery. It had8core passes,2two-destination connections and7eligible JSON outputs. Protocol conversion/worker algorithms are unchanged.
- Production TLS-verified desktop and emulated Android navigation passed:14single requests including12parallel requests, zero duplicate allocations or downloads. Original Android app/VPN status remains not_run.

## Deployment

Release: /opt/config-repair/releases/p8-18dfef6d1458ebb97f3a

Panel SHA256: 18dfef6d1458ebb97f3a24d8d3eb4b37126b6db8780c1ace779a016a52199796

Worker SHA256: 4dd17163b3607c372990df88701e0b660fc16cbfb151a433045999db445e8acb

Migration preserved all54source definitions/revisions and source/conversion settings by fingerprint; foreign_key_check was empty and quick_check was ok. Both services are enabled/active. Matching stopped-service DB/blob/key/release backup: /var/backups/config-repair/pre-rotation-20261030-121235. Schema6 binaries cannot open schema7. Rollback requires matching backup data and executable, with both services stopped.

Private receipts: `.local/rotation-targeted.log`, `.local/rotation-check.log`, `.local/rotation-browser.log`, `.local/rotation-worker.log`, `.local/rotation-deployed.json`, `.local/rotation-live-browser.json`, `.local/rotation-published.json`. Publication/readback and installed process hashes are checked separately. Final reproducible installer remains a separate release task.
