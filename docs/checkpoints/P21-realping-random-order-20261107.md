# P21 — Smart RealPing compatibility and random delivery

Parent: 13e43c9952e685008cd69794c3298657b4dd7c01. Branch go/v2rayng-2.3.6. User requests template comparison, a fix for nonresponsive pings, and random order on each Smart output-link visit. Smart v8; schema10 and ordinary template v4 remain unchanged.

## Confirmed cause and template comparison

Pinned AndroidLibXrayLite v26.7.28 / Lib40, commit d1d167b508457ebfd650c9e99bfca50c054c98df, `libv2ray_main.go:203` loads custom JSON then removes inbounds and all apps except OutboundConfig, dispatcher and log in MeasureOutboundDelay. That removes routing, DNS, burst observation and metrics. The test therefore dials through the default outbound. Both the supplied sample and P20 put block first, causing that test to fail even when normal leastLoad traffic works.

The exact essential-app filter is reproduced against pinned Xray 5ca6f4b7d4dc /26.7.28 in a closed netns/bwrap/cgroup fixture. The identical synthetic profile fails with a closed-pipe error when block is first and passes through its VLESS peer when an independent-IP proxy is first. The loopback HTTP204 target is synthetic; this is not an APK or Internet timing measurement. Other ordinary traffic tests retain full routing and deliberately give the first proxy a dead port: traffic must use the observed second proxy, proving the new default does not bypass leastLoad in normal use.

The common composer now puts one independent-IP entry first, moves block after copied entries/helpers, and keeps generated DNS/loopback helpers last. Both leastLoad strategies, block fallback, four resolver definitions, endpoint-only bootstrap domains, timeouts/stale cache, UseIPv4/keepalive, burst15s/3/5s, policy, counters and localhost metrics stay as P20. Differences from the example are intentional: logs remain fully off; neutral remarks and server/domain/tag counts are dynamic; the default order fixes RealPing. No example credentials are imported.

Group RealPing still measures one chosen default proxy, not every pool member. An unhealthy first proxy can fail it while the full config routes via another healthy member. Pinned admission proves a balanced source's route, not each nested entry. Do not promise universal ping success or2-second selection. P20's UseIPv4/system-resolver fallback limitation remains.

## Random output without rewriting cache or source data

The full subscription shuffles both profile order and proxy object slots on every GET. The single subscription shuffles the selected profile's proxy slots. The first object always belongs to the exact independent-IP selector. Tags stay attached to their original objects; helpers, references, every copied value and all global fields remain unchanged. JSON RawMessage preserves large numbers, unknown fields and null/absent distinctions. All copies and cache proofs remain canonical; the individual JSON/API viewer stays byte-exact.

New single-output rounds randomly snapshot ready IDs in SQLite; unfinished queues, cycles, HEAD/prefetch behavior, ownership and expiry gates persist. No new tables, bank or unbounded randomization history. The canonical profile ID is distinct from the randomized first proxy. X-Smart-Profile-ID makes that explicit while retaining X-Smart-First-ID for compatibility. Single response SHA covers the returned compact JSON without the final newline; X-Smart-Cache-SHA256 identifies its validated canonical cache. Full SHA covers all emitted JSONL bytes. Both links use no-store. Repeated random draws can coincide; a lone independent IP cannot provide different first proxies.

The Smart UI's obsolete leastPing and fixed-first-member wording is corrected. Sources, raw blobs, all connection values, user settings and ordinary/source exports are preserved.

## Verification and runtime

Final private receipts are .local/p21-validation.json, p21-traffic.log, p21-smart-worker.log, p21-browser.log, p21-before-canary.json, p21-after-canary.json, p21-audit.json, p21-public.json, p21-deployed.json and p21-published.json. Initial worker-harness failure was HTTP header case sensitivity in its new checksum assertion; the app emitted the correct header. The failed receipt is retained separately and is not release evidence.

The full race/build/core suite,1/2/5/7/10/15/50/100 random permutations, raw outbound/global preservation, canonical-cache checksum/API parity, randomized durable rounds and lifecycle concurrency must pass. Traffic fixtures cover the exact Lib40 filter, full-routing TCP/UDP/UDP443 failover, TLS HTTP/2 DoH, DNS A/TXT and cold failures. The50-member/55-profile/3-source worker test exercises SIGKILL and lease recovery, and browser tests cover restart and inline navigation. Test-only timing/CA/firewall substitutions never enter exported JSON.

Two exact P20 cached profiles on the server network reached Google204, Cloudflare200 and Instagram200 through their unmodified leastLoad configs before the fix. This corroborates the distinction between group ping and connected traffic. The phone was offline; Android import/TUN, Iranian operators, throughput, media playback and real QUIC remain unmeasured.

Deployment is same-schema: take the matching stopped-service DB/blobs/key/release backup, compare source/settings/rotation/ordinary-state fingerprints before and after initialization, switch the complete release and verify all3running binaries. Old Smart v7 cache/leases revalidate under v8. Same-schema rollback selects the previous release with services stopped and preserves current live data; do not automatically restore an old database over newer user edits. Publish only the tested tree, fetch it back and compare tree/source/binary receipts separately.


## Verified P21 release
All4required checks passed, including55profiles/50members/3sources with SIGKILL recovery. Public390px/1440px checks observed26changed renditions across repeated GETs and6single requests; headers, canonical-value equality, inline display and no JS errors/downloads passed. The cache audit reconstructed2profiles/70selected inputs/284copied outbounds byte-exactly. Two exact v8 profiles reached Google204, Cloudflare200 and Instagram200 on the server network after deployment. These are time-specific receipts, not carrier or per-member guarantees.

Release: /opt/config-repair/releases/p21-584b681a0ed3b76b46f8-r1
Panel SHA256: 584b681a0ed3b76b46f8fbe845c1cdf7123be3ff8f2dec1f61907d5235b477eb
Worker SHA256: 3da71a77db09b3d2483159e758e0926c8230350861a620249ee423eb6a18c8e5
Unchanged sandbox driver SHA256: 7c72e55456eb7bac75f1284560f61ef3d220ef16dc7c3fd17d678f547d73a136
Previous release: /opt/config-repair/releases/p20-7715237b769db399549a-r1
Matching backup: /var/backups/config-repair/pre-smart-realping-20261107-052234
All18sources and settings/rotations/ownership/ordinary conversion state matched before/after offline initialization. Count35/revision10 preserved; SQLite quick_check=ok and FK check empty. Three services are active/enabled and installed/running hashes match tested builds. Exact source publication/readback is recorded separately in .local/p21-published.json.
