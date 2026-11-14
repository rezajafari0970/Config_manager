# Ads-exclusive strategy v6 / schema32

The user defines Residential as Ads-only dedicated egress. Controls may independently use different exits, server egress, IPv6 or explicit block. This is a finite observed routing label, not household ISP classification or proof of all Google Ads flows.

## Independent modules

`internal/egress/targets.json` is the compiled, authoritative target/driver registry. Worker requests carry it to trusted Python helpers; reports cannot reduce required coverage. It includes six Ads destinations (three DoubleClick, pagead2.googlesyndication.com, adservice.google.com, www.googleadservices.com) and five controls (Google Search web, Cloudflare, ipify, Telegram web, Instagram web). Four observation drivers are google-peer, cloudflare-trace, ipify and https-reachability. Protocol drivers and pinned Xray/sing-box versions remain independent of classification, country policy, storage and scheduling.

`exclusive.go` evaluates each service separately for exact raw and rendered variants. At least two independent observations per required destination/variant are necessary. Controls do not need one stable common IP. Each comparable control wave must have Ads observations in the same family; cross-family source identity remains unverified. Any observed source overlap vetoes acceptance. A conclusive shared result requires repeated same-wave overlaps, not a single overlap amplified across rotating pools. For never-accepted inputs, a fully bound raw report with repeated paired overlap may reject early; Go independently validates it, and this shortcut can never accept an output. Domain destinations remain domains in SOCKS so domain rules are exercised. An IP echo on another hostname is never assigned to Telegram or Instagram.

`route_scope.go` can prove only simple explicit local blackhole rules from the exact bound runtime, plus repeated failed observations. Unknown routing conditions, remote timeouts, 403, CDN destination addresses and lack of response do not prove block or dedicated proxy. Multiple fixed proxy outbounds are no longer rejected solely by count; unresolved balancers and general direct routes remain outside the isolated scope.

`ads_health.go` validates completed bidirectional application exchanges on the actual Ads streams, with bounded byte counts and both variants. This is not a 64KiB upload, throughput certificate or proof of SDK ad serving. Existing generic upload/download results are retained separately as diagnostics; their failure cannot delete an Ads-only configuration. No destructive Ads health-negative driver is currently implemented, so inconclusive/unhealthy-looking Ads evidence withdraws publication and retries rather than manufacturing permanent failure.

## Evidence and Iran protection

The Google peer driver retains strict TLS, original Ads SNI, a completed request to that hostname and a second diagnostic authority on the same TLS connection. It requires nonce-bound peer-reported source data. This undocumented diagnostic is not a signed Google API; shared frontend internals can affect its meaning. Certificates, response digests, config/core hashes, time bounds and job-local HMAC source tokens remain. No plaintext source IP is persisted.

All bounded attempts retain their own core/runtime bindings. An earlier authenticated IR observation cannot disappear because a later attempt scores better. Country metadata is attached before HMAC scrubbing with one job key across attempts. Independent fresh provider consensus still supplies Iran evidence. A fresh minority IR vote blocks country publication but is not promoted to confirmed IR. It does not alone create a permanent identity revocation.

First valid IR consensus quarantines the identity across outputs when the report commits. Two independent positive jobs separated by at least 60 seconds, with the second starting after the previous report finished, revoke and exclude reimport. Clean/unknown tests never clear the sticky guard. Different attempts within one job cannot count as two positive rounds. Reports lost before completion (worker crash/timeout) do not create evidence. Existing durable lease recovery retries them.

## Unknowns and coverage

Telegram/Instagram web drivers currently return reachability only, not attributable source IPs. Without an independently observed source or provable local block they remain ROUTE_UNOBSERVABLE and strict publication stays disabled. Therefore the v6 rollout may publish zero previously accepted remote-split configurations. This is an intentional uncertainty gate, not proof those configurations are non-Residential. Additional source-aware drivers are required to confirm these services automatically. TLS/HTTP probe success is not proof of all app transports; Telegram MTProto, SDK flows, UDP/QUIC, ECH, IPv6-only ingress, DNS/WebRTC leaks and arbitrary CDN/rotating proxy topology are not certified.

Only finite required domains and observed source families are covered. Distinct observed IPs can be produced by one rotating proxy, and shared NAT can hide multiple paths; the result is observational rather than a mathematical topology proof. No 100% detection, zero-leak future behavior, speed or uptime claim is warranted.

## Scheduling, migration and operations

Existing resource-governed worker lanes, durable revision-fenced leases, startup cleanup, bounded retries, watchdog and systemd restart/boot activation remain. Accepted evidence expires no later than ten minutes from the earliest required observation, with advance renewal priority. Queue pressure does not extend proof. General health interval settings and source/country controls are preserved.

Schema32 prevents older binaries from interpreting changed health semantics. v6 reconciliation invalidates publication and health expiry while preserving reports, decisions, tested/attempt counters, raw bytes, candidates, manual exclusions, country choices and Iran guards. Historical receipts are displayed as awaiting renewal, never current proof. Existing accounting partitions remain unchanged. The API adds the compiled targets; evidence shows per-service status, general payload diagnostics and prior attempts.

Deploy with `/var/lock/config-repair-deploy.lock`, all six services stopped for a matched DB/blob/key/release/unit backup, and six binaries verified against tested source. Do not restore an older DB alone or bypass the external deletion journal. Run race/vet/build/Python/JS checks, migration preservation rehearsal, worker crash recovery/browser integration and live API/output/accounting reconciliation. Scope and limitations must remain visible in the panel.
