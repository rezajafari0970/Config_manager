# P25 — truthful receipt accounting and automatic inventory reacquisition

Parent: b977fbb7ea817105b2842ae2bb790485ddb9a9ae. User reports small/zero source counts despite upstream content and requests project-wide correction. P24 is the deployed baseline; target V2RayNG2.3.6/Lib40/Xray26.7.28, Smartv8 and normal templatev4 stay unchanged.

## Reproduced causes

- The first screenshot's retained upstream body was825bytes,6VLESS lines, with2distinct connection identities: one repeated5times with label-only differences. The existing count2 was a unique-inventory count, but the UI called it simply configs and mixed response duplicates with previously stored/shared identities. A later fresh fetch through the exact collection driver returned186lines/153identities/33repeats. This provider's changing response and the configured randomized schedule explain different snapshots; no collector item quota or truncation was found.
- V2pro's6retained profiles had expired under the fixed2hour retention, while the next user-scheduled fetch was still in the future. Receipt counters continued to say6stored although current inventory was0. A fresh response had6whole JSON profiles and12internal proxy outbounds. Whole profiles must remain intact;12outbounds are not12independent imported profiles.
- Partial expiry could still send conditional HTTP validators and accept304 while some previously stored members were absent. Global expiration in another source completion could erase ownership before GC knew which source to refresh.
- Separate reads in one state response could combine source counts and conversion summaries from different committed generations or different sides of a TTL boundary.

## Changes

Schema14 adds source receipt_unique, receipt_new, receipt_at and refresh_reason. -1 means unmeasured historical receipt; migration never fabricates a dedup count from the old combined duplicate field. The original duplicates field remains API-compatible. New receipt diagnostics distinguish extracted occurrences, unique identities, repeated occurrences within that body, new identities, previously present/shared identities and explicitly deletion-suppressed identities.304 and transport failures do not overwrite the previous body's receipt time/counters. Existing raw bytes and per-source identity representation remain unchanged. Direct completion now deduplicates the same way as staged collection.

The UI labels the current count as retained unique inputs, distinguishes complete JSON profiles from their internal proxies, shows the time of the body receipt separately from the last request, and shows current disabled inputs. Input/global summary and source-export count labels use the same terminology. Empty-source summary now counts enabled sources without current inventory, not merely a historical empty HTTP body. Zero inventory is not a statement that the upstream URL has no configs.

Every expiration path queues affected enabled sources before cascading ownership deletion. Completion caps the next run at the earliest retained artifact/response expiry. Startup repairs previously swept missing-inventory gaps once. Partial or absent inventory forces an unconditional fetch; a304without complete retained inventory becomes cache_missing with a bounded5s→5min recovery retry. Failed refreshes never resurrect expired bytes, bypass deletion/disabled controls, borrow another source's proof or change Splash ownership. Source min/max/enabled settings remain user-owned; event-driven expiry recovery is allowed to run earlier than the normal random schedule. Fixed lifetimes are not extended by repeated200/304responses; fresh acquisition after expiry starts a new lifecycle and validation.

State HTTP presentation uses one bounded read-only WAL transaction and one expiry timestamp for its source list, statistics and conversion/repair summaries. Writes/claims/rotations remain on the serialized writer; no vault reads are introduced inside transactions.

## Acceptance

Targeted regressions cover staged/direct duplicates, shared identities, suppression, unchanged receipts, partial304, expiration through GC/another completion/settings, restart/backoff, disabled sources, TTL non-extension, definition-preserving migration and concurrent snapshot/time-boundary consistency. Existing full race/vet/core/build and scope/isolation tests remain mandatory.

Browser fault injection verifies6received/2unique/4repeats and explicit recovery status. The old TTL browser expectation assumed an available upstream would stay empty until the ordinary schedule; it is updated to simulate an upstream outage, verify expired outputs disappear, then restore upstream availability and verify automatic refill without manual action. Initial browser/network tests collided with host network changes; browser reruns are separated from traffic tests. Failed intermediate logs are not final acceptance evidence.

Final receipts, deployment/runtime hashes, public counts and publication readback will be appended after validation. No model change or OpenAI API execution is claimed: no API key/execution integration is available in the authorized environment. Android/carrier performance and external provider availability remain unverified; no zero-defect guarantee.

## Upgrade/rollback

Before deployment take matching stopped-service DB/blob/key/release backup. On an isolated production snapshot, all35table fingerprints excluding the4new fields were preserved through13→14, guarded14→13(marker only; additive fields retained), an actual P24 Open, and14re-upgrade. Upgrade invalidates legacy receipt measurements so an old binary's later fetches cannot leave apparently current derived counters. Same-data rollback requires stopping all four services, setting user_version=13 only for this additive migration, then activating P24; never run P24 directly against schema14 or restore an old DB over later user edits. Earlier pre-Splash downgrades remain forbidden after adoption.
