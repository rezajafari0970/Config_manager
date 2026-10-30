# P10 — TXT and mixed response extraction

User request: receive the provided TXT subscription and accurately extract mixed custom JSON/Base64 responses.

## Findings and change

The existing TXT source had a case-sensitive filename mismatch. A direct request to the exact spelling provided by the user returned200/text/plain; the stored capitalized spelling had returned404. Corrected that existing source via the CSRF/revision-checked edit API without changing its intervals/timeout/name. The HTTP driver already accepts TXT, attachment headers and arbitrary MIME types; it must not lowercase URL paths or guess other endpoints.

The old extractor tried whole-body Base64 only if no JSON/URI was emitted. Mixed replies therefore lost encoded siblings. The built-in text driver now recognizes encoded blocks wherever they appear between JSON/URI records. It distinguishes independent encoded lines from wrapped continuations, handles standard/URL-safe padded/unpadded forms, whitespace/BOM and short wrapping, and decodes through temporary streaming files. The registry handles each decoded payload with a bounded eight-layer envelope depth.

Strict padding validation rejects corrupt envelopes instead of stripping all equals signs. Content errors remain local to their block; cancellation/storage failures propagate. Exact source response and exact decoded payload bytes stay immutable. Full custom JSON documents retain their entire outbound/dependency structure; no child outbound is emitted as a separate config. Identity/global/source dedup and conversion gates are unchanged.

## Validation

- Formatting, module integrity, vet, Go race tests, four binaries, pinned-core template matrix and9Python sandbox regressions passed.
- Synthetic regressions cover interleaved full JSON and independent Base64 lines; wrapped standard/URL-safe forms; short lines, blank lines, BOM, nested envelopes, comment-prefixed subscriptions; actual URL alphabet; corrupt padding/sibling isolation; context cancellation/emitter errors; byte identity and dedup; TXT attachment/MIME and case-sensitive HTTP paths.
- Independent SHA256 comparisons of live snapshots matched every plain/JSON span and decoded Base64 payload. Production raw downloads for all stored items matched those hashes.
- Initial TXT capture:101items/95unique. Newer fetch after deployment:101items/94unique; all94passed pinned core and two HTTPS destinations, and94inline JSONL records exported. Counts differ because upstream content changes.
- Initial mixed capture:4JSON +3Base64 =7items, all extracted. Newer production fetch:6JSON +1Base64 =7stored items. Their conversion remained explicitly blocked:5ambiguous primary selection,1unresolved structure,1unknown query parameter. No core/connection pass is claimed for them.
- Synthetic browser/restart tests passed; source full-output headers remained inline text, exact JSONL and duplicate-free. Source single-output HEAD remained non-consuming.
- First isolated worker attempt failed startup cleanup before processing; failure retained privately. Diagnostic rerun passed9samples/4live,8core passes,4twoHTTPS,8exports and SIGKILL/lease recovery. Worker binary and sandbox implementation are unchanged. No Android app/VPN test was run.

## Deployment and rollback

Release: /opt/config-repair/releases/p10-023265dc881f0651e0b7

Panel SHA256:023265dc881f0651e0b713532378a35cbd2fe3562ffdbcd40b12e1d4ebc248ba

Worker SHA256:4dd17163b3607c372990df88701e0b660fc16cbfb151a433045999db445e8acb

Schema8 unchanged. Seven source definitions/revisions and all settings matched at deployment; the intentional TXT spelling correction was already applied before that snapshot. Both services enabled/active; database quick_check ok and foreign_key_check empty. Global/source URLs and rotation schema unchanged.

Stopped-service backup:/var/backups/config-repair/pre-txt-mixed-20261030-132831. Same-schema rollback switches to the previous schema8 release while services are stopped; preserve current user data and edits.

Private evidence: txt-mixed-check.log, txt-mixed-browser.log, txt-mixed-worker.log (initial failure), txt-mixed-worker-retry.log (pass), txt-mixed-deployed.json, txt-mixed-live.json, txt-mixed-final-output.json, txt-mixed-byte-check.json, txt-mixed-published.json under .local. Raw response captures remain private and are never committed. Publication/readback is separate from installed-process verification.

Final boundary regression also proves a wrapped JSON child object is never promoted to an independent config while its parent is incomplete. Targeted before/after receipts are .local/txt-mixed-boundary-before.log and .local/txt-mixed-boundary-after.log. The final full checks/browser run passed again; worker binary remains unchanged.
