# P12 — Open TCP, UDP and QUIC traffic

The user explicitly requested all three traffic types enabled in generated configurations. QUIC is carried over UDP; no unsupported quic routing-network value or forced transport change is introduced.

## Implementation

The existing common envelope already enables SOCKS UDP, HTTP/TLS/QUIC sniffing and tcp,udp proxy/balancer routing. Source-global block rules are replaced by that envelope. The missing gates were outbound-local: VLESS Vision rejects UDP/443 without its supported -udp443 suffix; existing Mux defaults xudpProxyUDP443 to reject. Template v3 opens those exact gates throughout flat/nested users, retained chains and balancer pools. Explicit skip stays, null/absent mux stays, and no multiplexing/concurrency/auth/TLS/endpoint changes are made. Unknown policies remain diagnosed. Source originals are immutable.

Pinned source inspection: proxy/vless/outbound/outbound.go, infra/conf/vless.go, infra/conf/xray.go and app/proxyman/outbound/handler.go at the existing locked commit. VLESS strips the UDP443 suffix before sending the original Vision flow. The template version bump invalidates stale output proof and revalidates active source-owned data.

## Verification

- Full formatting/module/vet/race/build checks, pinned-core19protocol/envelope fixtures plus4balancer strategies,11Python sandbox regressions and browser/restart checks passed.
- Protocol matrix confirms UDP/sniffing/tcp,udp routing and no generated block route. Tests cover flat/nested Vision, pool/chain propagation, null/absent mux, allow/skip preservation, malformed policy rejection, idempotence and immutable input.
- Real isolated packet test for both Vision and Mux: TCP and ordinary UDP pass before/after; UDP443 fails before and passes after. Fresh SOCKS UDP associations prevent an existing session from bypassing the negative control. Synthetic VLESS peers use certificate pinning. The existing sandbox and its fixed DNS exception remain; fixture traffic stays on loopback. A narrow test-only freedom rule allows its local echo endpoints. No production egress policy changed.
- Worker integration: 9samples, 8core passes, 2two-HTTPS passes, 8exports; SIGKILL/lease recovery passed.
- Production audit at 2026-10-30T19:54:06.601528+00:00: 312exported JSON lines; all had UDP-enabled SOCKS, QUIC sniffing and tcp,udp routing; no Vision UDP443 rejection or disallowed Mux policy remained. No old conversion-job versions remained at that snapshot. Inline headers and zero duplicate lines verified.
- Real upstream QUIC/HTTP3 negotiation and Android app/TUN testing are not_run. Removing local policy gates does not add UDP capability to a TCP-only remote protocol/server.

## Deployment

Release: /opt/config-repair/releases/p12-0df73ef54bd1e27ac219
Panel SHA256: 0df73ef54bd1e27ac2196d19633ed607aa52d85733b00f95b3e85712ce6804c2
Worker SHA256: f697f1fafdd026687675b0f8302c35204dd01b4df8fb53830fdf689f12072b6a
Unchanged sandbox driver SHA256: c979512fb5ecd21e2cf97b0d73169aa338684b1cbdd40ab36a325b7b149199de

Schema8 unchanged; 8source definitions/revisions and all settings matched before/after; SQLite quick_check/foreign_key_check passed and both services are enabled/active. Matching stopped-service backup: /var/backups/config-repair/pre-traffic-20261030-195106. Same-schema rollback switches the release while preserving current user data.

Private receipts under .local: traffic-check.log, traffic-browser.log, traffic-worker.log, traffic-targeted.log, traffic-before.json, traffic-deployed.json, traffic-live.json and traffic-published.json. Initial packet-fixture failures were caused by the core's default private-destination block and by reusing a UDP association; the final fixture explicitly handles both and the final before/after checks pass. No live configs, credentials, source URLs or private captures belong in Git. Publication readback and installed-process hash verification are separate.
