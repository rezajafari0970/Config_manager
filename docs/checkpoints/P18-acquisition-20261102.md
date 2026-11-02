# P18 — truthful source acquisition and scoped recovery
Date: 2026-11-02. Schema10; normal template v4 and Smart v5 unchanged. Previous published source: b0bec8928c6372da684f1a8f9539c21485c5cb04.

## Problem and final behavior
Production had valid HTTP200 responses and extracted inputs but zero live configs for several enabled sources. Active manual-deletion controls suppressed insertion; Complete counted these as duplicates and still recorded updated/LastSuccess. The UI then chose its green badge from historical content_state=config even with config_count=0. A retained response alone also allowed an unchanged304 success without live source membership.

Schema10 adds stored_count and suppressed_count receipt fields. Both direct and staged completion distinguish deleted identities from ordinary existing/repeated items, count the source's current unexpired memberships, and record filtered/configs_deleted when all accepted items are suppressed. This does not advance LastSuccess or misclassify policy exclusion as a network-error streak. Stats.GivingConfigs now requires enabled live ownership. A304 without retained response and live members becomes cache_missing, clearing validators for a fresh fetch. Zero-current rows never show green collection success, including after manual deletion or TTL expiry.

Input details separate extracted count, stored count after the last processing, deletion suppression and duplicates. Current count remains authoritative. Stored raw input, core-ready printable output and verified connectivity remain separate. A source with stored input but zero ready and positive pending conversion shows the amber pending-output label. Ordinary core-valid output still does not imply connectivity.

The explicit restore button uses the same-origin revision-checked action. RestoreSource reads and parses only the source's retained unexpired immutable response, calculates the collector's exact identities and removes only matching deleted controls. Disabled controls and unrelated deleted identities remain. Controls are identity-global, so a shared identity becomes collectible by its other owners too. It rechecks revision/enabled/response hash/expiry inside the transaction, fences old leases/staged work, clears validators and queues fresh collection. No old output or proof is reinstated. A later manual deletion wins over an in-flight fetch. Vault reads remain outside SQLite transactions; temporary plaintext is mode0600 and removed. Source URLs, schedules, enabled state, fixed TTL, generation replacement, settings and rotation contracts are preserved.

## Evidence
- Full module/gofmt/vet/race/build checks, pinned-core regressions, 12 Python regressions and 14 source-status assertions passed.
- Backend regressions cover direct/staged suppression, partial/shared dedup, LastSuccess, scoped restoration, stale workers, later deletion, cancelled/stale/disabled/expired restoration and missing304 memberships. Browser exercises the actual delete/refetch/restore UI, raw ownership/dedup, mobile/desktop, restart and TTL.
- Normal sandbox worker:9 inputs,8 core-valid,2 connected at this run,3 concurrent slots, SIGKILL recovery. Smart worker:57 profiles of50 members,4 sources,2 slots, SIGKILL recovery. Synthetic admission is not real all-member connectivity evidence. Isolated ctw/cst resources stayed separate from production crw/csm.
- Fresh production receipts and a7/7 raw-response hash audit confirmed acquisition. Final public HTTPS browser at 2026-11-02T11:25:01.159Z: Speedray31 stored/31 printable; large Agood92 stored/92 printable; small rotating Agood1 stored/0 printable with truthful pending validation. The large source extracted103 entries but identity dedup yielded92 unique configs; do not interpret these counts as missing extraction.
- Final browser also observed Afrak20, Yak20, Mixo30 and Raytunnel1. Four paused sources remained paused. Both390px and1440px views had no JS errors/overflow and no zero-count green badges. These are time-stamped snapshots, not permanent counts or carrier/Android evidence.
- The first public export assertion incorrectly required ready output from the rapidly changing single-item Agood. Its failure is retained privately; the final check explicitly distinguishes pending validation and does not claim that empty export connected or printed. Recovery actions receipt is empty: do not attribute concurrent source recovery to an explicit assistant restore operation.

## Release and rollback
Final release: /opt/config-repair/releases/p18-671ffd82e8b215532041-r2
Panel SHA256: 671ffd82e8b2155320413b2e9ce6d779e2e37ceeb2d5779de797681c17185cb6
Worker SHA256: 274b5b5eb3b4a4edcd71aeb191b79ea4d6196568e08659aedf77487e91ed4b29
Sandbox driver SHA256: 7c72e55456eb7bac75f1284560f61ef3d220ef16dc7c3fd17d678f547d73a136
All three services active/enabled with running executable hashes checked. All11 source definitions/settings/rotations/ownership matched across each stopped-service installation.

Initial schema9-to10 receipt: .local/p18-initial-deployed.json; matching stopped-service data/blob/key/release backup: /var/backups/config-repair/pre-acquisition-20261102-110522. The first deployment preflight failed before migration because runuser was unavailable, then restored services; the successful retry used subprocess service-user credentials. Its receipt remains private.

Final UI-only deployment receipt: .local/p18-deployed.json; previous schema10 release: /opt/config-repair/releases/p18-74cb966517ab8070fc2b-r2; backup: /var/backups/config-repair/pre-acquisition-20261102-112405. Same-schema rollback can select that previous schema10 release with all services stopped, retaining live data. A schema9 binary cannot open schema10: reverting to P17 requires a deliberate additive-column/schema rollback or matching full backup, reconciling newer user data. Never blindly restore an old DB over new changes. Private key/config contents must not enter Git.

Current private receipts are p18-final-check/browser, p18-validation, p18-workers, p18-conversion-worker, p18-smart-worker, p18-recovery, p18-public, p18-initial-deployed, p18-deployed and p18-published. Publish the tested source on the existing branch; compare the exact remote tree and running hashes separately. No changes to endpoint credentials, templates, Smart admission, source intervals or carrier settings. The reproducible installer and actual Android/operator performance remain separate tasks.

Final live settings readback found user Smart count35/revision10; this newer setting is preserved. Do not reset it to the historical P17 count10.
