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

## Single-config rotation

Output also exposes `single_export_url` through `/api/conversions`: `/subscription/{token}/next.json`, with a separate derived token. Each ordinary GET allocates one exact eligible JSON plus a newline, served as inline UTF-8 plain text with no cache. The existing all-config link is unchanged and never advances the rotation. HEAD and explicit prefetch requests do not consume turns. If no eligible config exists, the single endpoint returns HTTP204 with an empty body.

Schema7 adds only a singleton cycle number and a bounded queue of candidate hashes for the current round. A round snapshots unique currently eligible output; new arrivals join the next round. Missing, removed, disabled, expired or no-longer-accepted candidates are skipped; corrupt blobs trigger the existing regeneration path. After the remaining queue is exhausted, the next request starts a new snapshot. All clients of this link share one rotation, persisted across service/server restarts. No config body or historical repair data is stored in the rotation tables.

Selection and claiming use SQLite write transactions, including guards across multiple Store instances. Blob reads occur outside transactions to avoid a vault/GC lock inversion; eligibility and queue position are checked again before the atomic claim. The allocation is committed before HTTP writing. Therefore concurrent successful allocations do not repeat within a round, but a broken client connection after commit can lose that allocated response; HTTP without a client acknowledgement cannot prove delivery. `X-Export-Cycle` and `X-Content-SHA256` identify the allocation without adding anything to the body.

Regressions cover multiple complete rounds, restart/migration, concurrent independent Store instances with garbage collection, dynamic additions, disable/delete/expiry, stale claims, corrupt blobs, cancellation, token scope, HEAD/prefetch, exact single-line output and empty204. The browser fixture navigates both export links, checks no downloads, and verifies the persisted position after process restart.

## Per-source exports

Each source has full JSONL and rotating single-JSON links, available under its Input row and the searchable, paginated source list in Output. `/output?source={id}` shows only that source's conversions and copy actions. `/api/conversions?source={id}` returns its scoped entries and links; `/api/export/sources` lists source names, states and links without revealing input URLs. Existing global export URLs and their rotation position are preserved.

Ownership is filtered by `memberships.source_id` **before** canonical identity ranking. A passing representation from source A cannot be borrowed by B, even when both share a canonical artifact identity. An identical config may belong to multiple sources: it appears in each owner's output but remains deduplicated in the global output. Immutable blobs and conversion results are still shared by content hash; source membership is not duplicated or rewritten.

Schema8 scopes the current-round state/queue by source ID (empty scope is global). It migrates the existing global round without restarting it. Each source has its own finite snapshot, cycle counter, atomic claims and cancellable in-process lock. Source locks are released/reclaimed independently. SQLite still serializes short write transactions; no blob/network I/O holds the DB lock. A blocked source lock cannot hold another scope. Source deletion cascades its state/queue only; disables, expiry, generation replacement and proof failures are rechecked before every claim.

Source export tokens are bound to source ID and output mode. Deleted sources return404; disabled/no-eligible sources have empty full output and204 single output. Both formats are inline text, never attachments. New candidates join that source's next round. The global round and other source rounds do not advance when a source-specific link is used.
