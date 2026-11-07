# P20 — tunneled dual DoH and independent IP bootstrap
Date: 2026-11-07. Base source: bfebe6bc524fd8e3dd34d9fabc0e8993d33fbed5.
Target v2rayNG2.3.6 / Lib40 / Xray26.7.28; schema10, ordinary template v4, Smart v7.

## Behavior
The user's explicit template updates the shared Smart composer used by every existing individual/full/single/API/cache path. Ordinary source/global output URLs, raw encrypted sources, source acquisition and normal conversion remain unchanged. There is no new ordinary-to-Smart endpoint substitution.

Four DNS entries: endpoint-scoped Google then Cloudflare, both tagged dns-bootstrap/skipFallback true; general Cloudflare then Google, skipFallback false. Every timeout is2500ms. QueryStrategy UseIPv4, disableFallbackIfMatch, parallel queries, disableCache false, serveStale true and serveExpiredTTL60 match the request; Quad9 and generated local bootstrap are removed.

Actual endpoint names are deduplicated/sorted from compact/legacy proxy settings, WireGuard peers and known nested XHTTP download streams. SNI, Host, source DNS and arbitrary lookalike extension fields never become bootstrap domains. Main leastLoad retains all selected entries; bootstrap-ip-balancer admits only independently IP-based entries, conservatively including all endpoints and dialer dependencies. Domain members remain in main. No independent IP returns smart_bootstrap_no_ip with Persian UI diagnosis.

Both leastLoad pools use expected3/maxRTT2s/tolerance0.15/fallback block. Proxy tags have equal numeric width based on the total flattened pool, preventing prefix collisions above99. No example counts, addresses, UUIDs, SNI or domain names are inserted. Every input outbound/helper is copied; only tag/reference namespaces and explicitly requested UseIPv4/keepalive45/15 change. Unknown fields, number precision, credentials, TLS, transport and Mux remain. Dynamic substitutions occur before inserting copied outbounds, preserving nested empty domains/selector arrays.

Generated block is first and generated DNS/loopback last; no generated freedom. Total outbounds = copied input definitions+3. DNS rules follow the example, including dns-bootstrap and smart-dns-egress. Burst15s/sampling3/timeout5s/GET, system counters and localhost11111 metrics are present. All log channels remain off. Neutral remarks retain smart-parallel-v6b without a hardcoded server count.

Smart v7 invalidates older cache/leases through existing ownership/version gates. Settings, TTL, source membership and normal/Smart rotations are preserved. Existing copied Android JSON must be refreshed/reimported.

## Verification and limitation
Full check.sh passed: gofmt/modules/vet/race/build, pinned-core fixtures and existing Python/status regressions. Smart core matrix1/2/5/7/10/15/50/100; dynamic prefix regressions through150; no-IP, chained/domain-download, malformed tags and exact outbound/envelope preservation. Previous v6 ready cache and occupied leases are fenced.

Closed synthetic fixtures passed requested leastLoad distribution, DNS A/TXT, dead-peer recovery, TCP/UDP/UDP443 and two real TLS HTTP/2 DoH resolvers through the independent IP proxy; the domain proxy subsequently connected. HTTP/2 fixture CA is bound only inside the test sandbox, not host trust. Domain-bootstrap observation is accelerated only in the isolated fixtures; exact shipped timing is separately asserted and retained in the IP distribution/recovery comparison. Local metrics are read to wait for actual observations.

UseIPv4 is explicitly retained, but pinned Xray falls back to its system resolver when internal resolution fails. The final TLS fixture counted 8 blocked system-DNS attempts during startup and 20 after cold restart with the IP peer unavailable. These counts are one synthetic run, not a stable rate. The fixture firewall prevents egress; it is NOT in exported JSON. Therefore the template does not by itself guarantee zero DNS leakage, even with disableFallbackIfMatch. No silent switch to ForceIPv4, endpoint IP rewriting or TLS weakening was made.

Smart worker: 55 profiles of50members,3synthetic sources,2slots, SIGKILL/lease recovery, independent first-position rotation and inline exact bytes passed. Browser/restart tests passed. Public390px/1440px Smart API, individual hash verification, full-profile bytes and6single responses passed without downloads/JS errors. Private cache audit reconstructed 4 profiles/140 selected inputs/560 copied outbounds byte-exactly using the shared composer. Non-Smart conversion/collection/probe-driver source diff is empty.

No Android, MCI/Irancell, real QUIC handshake, throughput or2-second selection guarantee is claimed. Synthetic proxy recovery is not TCP session migration.

## Deployment
Release: /opt/config-repair/releases/p20-7715237b769db399549a-r1
Panel SHA256: 7715237b769db399549a1c4af71c2e71f0878520c1bfd02062af83ee668fac67
Worker SHA256: afe8de61cb84a3f74dc502c579c1b85f988653efb67dc0361f88637e5a35e638
Unchanged sandbox driver SHA256: 7c72e55456eb7bac75f1284560f61ef3d220ef16dc7c3fd17d678f547d73a136
Previous release: /opt/config-repair/releases/p19-7ef4bd9a853895ffa91e-r2
Matching stopped-service backup: /var/backups/config-repair/pre-tunneled-bootstrap-20261107-042555
All 18 source rows and settings/rotations/ownership/artifacts/normal conversion state matched before/after offline initialization; count35/revision10 preserved. SQLite quick_check=ok, no FK errors. Three services active/enabled and running binaries matched release hashes.

Same-schema rollback: stop all3services, select the previous schema10 release and restart; its versioned Smart cache/views revalidate. Preserve live data. Full matching backup restore is only for actual data recovery, never an automatic overwrite of newer user changes.

Private receipts: .local/p20-validation.json, p20-check.log, p20-traffic.log, p20-smart-worker.log, p20-browser.log, p20-deployed.json, p20-public.json, p20-audit.json, p20-runtime.json and p20-published.json. The final runtime and publication receipts must be checked separately. Failed intermediate fixture experiments are diagnostic only, not deployment evidence.
