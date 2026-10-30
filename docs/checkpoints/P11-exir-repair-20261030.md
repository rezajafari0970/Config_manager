# P11 — Exir structural and balanced-template conversion

The user's current task is to resolve mixed/invalid Exir conversion using the shared JSON envelope, preserving original source bytes and accurate evidence. The permanent bank remains removed.

## Root causes and implementation

- Empty arrays in DNS and blackhole outbound settings were not covered by the object repair allowlist. Structural v2 permits only empty-array to empty-object replacement for these built-ins, alongside freedom. Nonempty values and required proxy fields stay diagnosed.
- Valid multi-proxy custom configs with an explicit universal balancer route were rejected as ambiguous. Template v2 resolves that route, expands prefix selectors, retains original member/fallback/chain tags and observer/strategy settings, and uses the common SOCKS/DNS envelope. It preserves the original first real proxy to prevent implicit fallback into loopback. Collision-checked helper tags cannot enter the balancer/observer pool. Missing, cyclic, nonproxy or ambiguous graphs fail explicitly.
- A known XHTTP URI used x_padding_bytes. It now maps to extra.xPaddingBytes with positive ordered int32 validation and conflict detection. Unknown/duplicate parameters and non-XHTTP use are not silently discarded.
- IPv6-only members no longer prevent a mixed pool's IPv4 test. The exact candidate keeps every member. All endpoint resolutions still undergo public-address checks; the sandbox's inet default-drop policy and endpoint allowlist remain. Entirely IPv6-only connections stay pending; pool success does not imply every member works.
- Input uses the same source-first canonical conversion ranking as Output for ready/connected/repaired/pending/blocked counts. Raw-response quality remains a separate diagnostic. Fresh connection evidence is time-bounded; old template versions are pending instead of shown ready.

Pinned source builders/router/selector/loopback/XHTTP code was inspected; reference commits and core checksum remain unchanged. Template v2 automatically invalidates old-version export eligibility and schedules source-owned originals for current validation. Schema8, retention, original bytes, source definitions, settings, tokens and rotation state are unchanged.

## Validation

Full formatting/module verification/vet/race/build checks passed; the 19 existing template cases plus four balancer strategy fixtures passed exact pinned-core validation. Eleven Python sandbox regressions passed. Added tests cover structural allowlists, XHTTP aliases/conflicts, pool/fallback/selector collisions, explicit routing priority, ambiguity, cycles, idempotence, immutable originals and canonical source-owned summary evidence/expiry.

Synthetic desktop/mobile browser and process-restart tests passed. Isolated ctw worker integration passed9samples/4live,8core passes,2two-HTTPS passes,2partial responses,8exports and SIGKILL/lease recovery with3parallel slots. Nonpublic synthetic endpoints remained blocked. Android app/TUN was not run.

Private current/historical Exir canaries:20distinct original byte representations,20core passes after conversion,6two-destination HTTPS passes,2partial,9failed connections and3not-run (2IPv6-only,1DNS failure). These are different upstream snapshots, not20simultaneously active configs.

Production refresh at2026-10-30T18:52:27Z:7received/7unique,6originally structurally valid/1invalid. All7derived outputs passed the pinned core and were exported;6passed two HTTPS destinations and1IPv6-only network test remained pending. All7original download SHA256s matched their stored hashes. Inline JSONL had no duplicate lines and only owned accepted candidate hashes; single-output HEAD was non-consuming. Desktop and emulated Android browser navigation passed with0downloads,0JavaScript errors and no horizontal overflow. Counts and network results can change on later upstream refreshes.

## Deployment and rollback

Release:/opt/config-repair/releases/p11-2578fa427fa20339a69f
Panel SHA256:2578fa427fa20339a69f4c735628239384bcd5ec302f3318b9fed3bc56f6888a
Worker SHA256:dde8331f49b28f8b884cbc79ab425c84af67b6162d55644ba12683cbf0f8ea52
Sandbox driver SHA256:c979512fb5ecd21e2cf97b0d73169aa338684b1cbdd40ab36a325b7b149199de

Both services enabled/active; actual process hashes and installed helper matched tested files. Seven source definitions/revisions and all settings matched before/after deployment; database quick_check passed and foreign_key_check was empty. Matching stopped-service backup:/var/backups/config-repair/pre-exir-repair-20261030-185005. Same-schema rollback switches panel, worker and helper together to the previous release while services are stopped; retain current user data/edits.

Private receipts under .local: exir-repair-check.log, exir-repair-browser.log, exir-repair-worker.log, exir-repair-deployed.json, exir-repair-live.json, exir-repair-live-browser.json, exir-repair-published.json; private canary results under exir-repair/. No actual configs, source URLs, DB, logs or credentials are committed. Git publication/readback is verified separately from runtime installation.
