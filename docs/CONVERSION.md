# Shared JSON conversion pipeline

Current contract: P7, 2026-10-30. Fetch → extract immutable originals → prepare recoverable protocol mapping → apply @shenvpn envelope → exact pinned-core test → isolated HTTPS test → transient output.

## Envelope and repairs

`internal/convert` owns the template. Every output has the user's log, three proxied DoH resolvers, tagged local SOCKS10808, sniffing, DNS-out/proxy/direct/block routing, stats and policy. The first proxy outbound retains actual credentials and transport/security. VLESS/VMess use vnext when the flat form has no incompatible advanced options. WireGuard and Hysteria2 retain their native settings; a universal VLESS settings object would be invalid.

Domain endpoints get a domain-scoped `https+local://1.1.1.1/dns-query` bootstrap resolver, preventing the proxy from needing itself to resolve its own server. Ordinary queries continue through the requested proxied DoH resolvers. Existing outbound socket options are retained; absent IPv4 strategy is supplied. Missing VLESS encryption receives the standard `none` value; this never adds remote encryption or TLS support.

Existing deterministic structural drivers handle recoverable JSON container/trailing-comma/URI encoding errors. Trojan TLS and strict-certificate alternatives are explicit hypotheses and require fresh actual connection evidence before export. Credentials, Reality keys, endpoints and remote protocol capabilities are never invented. Necessary chains are preserved; ambiguous primary outbounds, missing/cyclic dependencies, Hysteria1 and unknown unsupported fields stay diagnosed.

## Lifecycle and output

Schema6 drops all bank/probe history/model/settings tables, and creates a transient conversion queue/results/settings/worker set. Active source memberships own the queue. Replacing, deleting, disabling or expiring the last owning source/config removes eligibility and invalidates late completion. Empty/error responses retain previous membership only until fixed TTL. Identical received data never extends that TTL. Exact raw blobs and derived blobs have separate hashes and GC references.

The public Output page has only a config list and one tokenized `/subscription/{token}/configs.jsonl` link. The response uses `Content-Type: text/plain; charset=utf-8` and `Content-Disposition: inline` so browsers display it without downloading. Each line is the exact compact candidate accepted by the pinned core; no JSON array, report, separator or wrapper is emitted. Canonical identity and final JSON hash deduplicate output. Failed/unsupported/pending conversions appear with their status but are excluded from the export. Ordinary core-valid configs may have failed connectivity; no healthy guarantee is implied. Semantic hypotheses require a fresh two-destination pass. Settings retains clearing of source/config data.

The old `/api/bank`, `/bank/*`, `/api/probe/settings`, `/sub/*` routes and bank UI are removed. Original downloads remain available to Input diagnostics. There is no persistent test bank, variant history, learned priority or manual bank import.

## Compatibility and execution

Pinned source: v2rayNG `13138dddcab291ba3618cc54721ce4c7dd926be0`, Xray `5ca6f4b7d4dc20a881d4330e498892697627ec0c`.

Reviewed Xray infra/conf log, DNS outbound, VLESS, VMess, SOCKS and app/dns local DoH selection; tested the complete envelope with 19 protocol/transport/envelope fixtures on the exact core checksum in internal/probe/types.go.

v2rayNG AngConfigManager imports URI lines first, then a Custom JSON array or one object. Multiple raw JSON objects separated by newlines are not a native Custom subscription format. JSONL was explicitly requested and remains unchanged; individual JSON can be copied for import. Android app/TUN validation is not_run.

The existing systemd prober is now a conversion validator, not a laboratory bank. Private Unix claim/renew/complete/status are never served on public TCP. Source collection continues independently. Durable leases, cancellation, backoff, process restart and owned-resource cleanup provide recovery. The exact JSON is core-tested. Network testing uses a recorded runtime clone with isolated inbound, restricted logging, pinned endpoint DNS and blocked direct egress; network evidence is not an Android receipt or a guarantee for every client network.

## Checks

`internal/convert/template_test.go`: matrix, pinned binary checksum/core acceptance, template idempotence, immutable input, credentials/transport/numbers, default repair, ambiguity, chain dependencies and hypothesis gate. `internal/panel/conversion_test.go`: schema5 removal, source retention, dedup/JSONL, private proof validation, stale leases, deletion/TTL and hypothesis freshness. Browser tests use explicitly synthetic UI evidence; the sandbox integration uses its own ctw prefix and can read a bounded sample of authorized live raw outputs without altering production data.
