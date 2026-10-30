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

## P11 mixed custom repairs and explicit balanced routing

Template version `xray-26.7.28.template.v2` invalidates old output proofs and schedules current source-owned originals for validation again. Structural rules are `xray-26.7.28.structural.v2`. Only empty arrays in the allowlisted DNS/blackhole/freedom outbound settings become empty objects; nonempty arrays, credentials and immutable original bytes are not rewritten.

An explicit universal routing rule takes precedence over a coincidentally named proxy outbound. If it selects a balancer, preserve that balancer's members, strategy, fallback, observer settings and necessary chain dependencies under their original tags. Prefix selectors are expanded using the pinned core's semantics. Missing/cyclic dependencies, nonproxy pool members/fallbacks, ambiguous selection and an incompatible implicit default remain explicit failures. The original first real proxy remains first so an empty balancer result cannot default into a loopback cycle. Collision-checked infrastructure tags keep DNS/direct/block/loopback helpers out of pool and observer selectors. The common SOCKS/DNS envelope routes through the selected balancer; its DNS dialer uses a loopback adapter to the same route.

For XHTTP URIs, the known `x_padding_bytes` alias maps to `xhttpSettings.extra.xPaddingBytes`. Positive ordered int32 ranges are checked and an already-present extra value must agree. Conflicting values, duplicate/unknown parameters or use on another transport are rejected rather than discarded.

The sandbox still validates all resolved endpoint addresses and blocks private/reserved destinations. An IPv6-only member no longer prevents testing a mixed pool through its actual IPv4 members. It remains present in the exact candidate; IPv6 egress is blocked in this IPv4 sandbox. Entirely IPv6-only configs remain `ipv6_only_probe_pending`. A successful pool connection proves the selected route on the test network, not every member.

Input now reports per-source canonical ready, freshly connected, structurally repaired, pending and blocked counts from the same ownership/eligibility ranking as Output. Historical raw representations no longer inflate these counters. Original-response diagnostics remain visible; core acceptance, server connection evidence and unperformed Android testing remain distinct.

## P12 open TCP, UDP and QUIC traffic

The user's explicit policy is to allow TCP and UDP, including QUIC over UDP/443. Template v3 retains SOCKS UDP, HTTP/TLS/QUIC sniffing and the catch-all `tcp,udp` proxy/balancer route. Source-global blocking rules are replaced by this common routing envelope. QUIC is not an additional value for Xray's routing network field.

The pinned VLESS client rejects UDP/443 for `xtls-rprx-vision`. Its supported `xtls-rprx-vision-udp443` variant permits it and strips the suffix before sending the same Vision flow to the server. The converter now applies that exact change to flat/nested VLESS users throughout retained chains and balancer pools. Empty/other flows, endpoints, authentication, TLS/Reality and transport settings are unchanged.

Existing mux objects with absent/empty/reject `xudpProxyUDP443` get `allow`; explicit `allow` and native-UDP `skip` remain. Multiplexing is never introduced or enabled and its concurrency is unchanged. Unknown policy values are diagnosed. This is an explicit client traffic policy, not a guessed remote protocol repair. Immutable originals remain intact.

Regression coverage includes the protocol matrix, chains/pools, source immutability and idempotence. `XRAY_TRAFFIC_TEST=1 XRAY_TEST_BINARY=... go test -count=1 -v ./internal/convert -run TestPinnedCoreTrafficForwarding` runs as root using the established network/mount/PID/cgroup sandbox, synthetic certificate-pinned VLESS/TLS peers and loopback echo endpoints. The existing endpoint-deny guard and its fixed DNS exception remain; fixture traffic stays on loopback. Separate fresh SOCKS UDP associations demonstrate each Vision/Mux gate: TCP and ordinary UDP pass before and after; UDP/443 fails before and passes after. This does not claim a real QUIC/HTTP3 handshake or that every upstream server supports UDP. HTTPS health proofs remain HTTPS proofs; Android app/TUN remains untested.

## P13 shared URI compatibility and exact source diagnostics

Template v4 invalidates previous proofs and automatically revalidates active originals. Structural v3 adds a compiled URI userinfo repair driver. It only replaces literal ASCII spaces inside an unambiguous authority credential with %20, preserving percent-decoded credentials, all other bytes, and immutable originals. Multiple @ delimiters, bad escapes, malformed hosts, missing ports and control characters remain rejected. The pinned core accepts short string VLESS IDs using its existing deterministic UUID conversion; the converter never trims or changes that ID.

TLS cs maps to tlsSettings.cipherSuites byte-for-byte; existing fp (including the core's TLS unsafe fingerprint value), SNI, mask, paths and credential data remain. The unsafe fingerprint name is not an instruction to disable certificate verification. Unknown/duplicate parameters still fail explicitly.

An explicitly supplied VLESS packetEncoding=xudp maps to mux.enabled=true, concurrency=-1, xudpConcurrency=1 and xudpProxyUDP443=allow. The pinned manager therefore carries UDP in XUDP without multiplexing TCP or pooling multiple UDP flows. No Mux is added to other inputs. Other packet encodings and other protocols remain unsupported rather than silently losing packet semantics. This explicit source option is distinct from the P12 client-policy change, which never introduced Mux on its own.

Before running the core, retained REALITY streams are checked for source-supplied publicKey/password, 32-byte raw-URL-base64 key shape and valid shortId. This also covers pool/chain members and nested download settings. Missing keys are not repairable from an address/SNI alone. They are shown as missing information rather than generic handshake or JSON failure. Core validation still performs its complete checks.

Input issue counts share Output's source-filtered canonical ranking and exclude stale/pending attempts. Input and Output import one diagnostic-message module, exposed by the asset allowlist. The original response may remain malformed while its separately derived output is repaired; neither UI claims Android testing or connection merely because JSON is valid.

New regressions cover URI immutability/idempotence/ambiguous cases, exact cipher/fingerprint/mask preservation, explicit XUDP and unsupported encodings, REALITY aliases/missing keys/pools/nesting, and source-isolated diagnostics. The pinned-core matrix has22fixtures. Real synthetic packet fixtures test Vision, existing Mux and explicit XUDP: TCP/UDP pass throughout, UDP443 fails in the negative control and passes after conversion. Live HTTPS evidence remains separate from UDP capability and QUIC/HTTP3 negotiation.
