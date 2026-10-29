# P3b — acquisition controls and diagnostics

The user requested bulk delete-all controls, selected disable/delete, live-source diagnosis, immutable raw data, deduplication, independent parallel collection, startup/recovery and configurable timeout seconds. The 96.8-second recording and screenshots showed Android's decimal keyboard preventing easy entry of a range; the interval field now uses text input.

## Implementation

- Source/config bulk enable, disable, delete plus distinct clear-all buttons; destructive confirmation and CSRF protection.
- Transactional source revision checks, cancellation and stale-completion rejection. Config tombstones expire with original retention; disabled config bytes are unchanged and excluded from ordinary exports. Clear-all cannot be undone by an already-running old job.
- Source timeout override and global default in seconds. Actual network wait uses that context; hidden transport cutoffs removed. Effective concurrent capacity is displayed.
- Distinct readable/mixed/invalid/empty/unrecognized/error outcomes, raw response sizes and candidate counts. Original responses retained even on non-2xx. Malformed JSON siblings no longer erase good siblings. JSON string Custom payloads supported without reserialization.
- InspectionDriver extension boundary with local panic containment. Corrupt source-record decoding cannot block other due sources.
- Schema 2→3 migration preserves source secrets, names, schedules, enabled flags, revisions and global settings.

## Evidence

Full module/format/vet/race/build gate passed. A network regression actually waits 15.2 seconds for headers under a 20-second context and checks a short independent timeout. Transaction, tombstone, inspector, raw-span and timeout-isolation tests passed. Browser tests passed bulk operations, mobile text input, settings, actual restart and byte-exact ZIP/expiry.

Isolated actual-source sweep: 54 sources, peak 16 parallel requests, 12.91 seconds, 44 readable / 1 invalid / 6 empty / 3 unrecognized in that sample. 950 raw/config downloads verified SHA256; no raw-identical duplicate identities. This is a sampled acquisition result, not a VPN health or throughput guarantee. Penguin's sampled JSON has stats arrays and freedom settings arrays; raw bytes remain unchanged.

Production migration preserved 54 sources and settings, with unchanged nginx files and boot-enabled service. Running candidate SHA256: `c295a35a3a01516257ab67e797b08e0a4cd7fae15028ec24cf6b01f9485395d6`. Publication, CI and executable readback are separate post-commit receipts under `.local/`.

No deletion control was exercised on the user's production data. No host reboot, Android test or arbitrary-config core execution was performed. Restart/expiry control tests use isolated fixtures; watchdog stall recovery uses a separate transient service. Temporary copies of real config payloads are removed after audit.

Final installed-version sweep completed for all 54 enabled sources and verified every original-response download hash, with zero identical raw duplicate identities. Its sample classified 41 readable, 2 mixed, 2 invalid, 6 empty and 3 unrecognized responses. A new-binary transient SIGSTOP test received real watchdog heartbeats and restarted automatically with a different PID. Production data was untouched by that recovery test.
