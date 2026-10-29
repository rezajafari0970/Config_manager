# Temporary probe laboratory

Target: V2RayNG 2.3.6 / Lib40 / Xray26.7.28. This laboratory does not claim Android acceptance and does not publish a healthy subscription. The existing collector and encrypted originals remain unchanged.

Build `cmd/config-probe-plan` using the project's verified Go toolchain. The Go planner reads one original on stdin and emits separate byte-preserving/base64 candidate payloads, hashes and versioned rule IDs. `scripts/live-probe.py` is a temporary privileged Linux sandbox driver; it is not the permanent collector worker. The SQLite bank stores only fingerprints, diagnostic codes, timing and separate core/connection/Android evidence. No credentials or original configs are stored in the bank. Candidate material is transient under protected `/dev/shm` directories and removed after each job.

```sh
go build -trimpath -buildvcs=false -o bin/config-probe-plan ./cmd/config-probe-plan
python3 scripts/live-probe-test.py
python3 scripts/live-probe.py --self-test
python3 scripts/live-probe.py --workers 4 --timeout 8
```

The live driver reads the local panel API using its configured public Host, skips disabled/expired originals, verifies the snapshot hash before execution and binds evidence to the exact original/candidate/runtime hashes. Source changes between reading and testing are reported instead of attributed to another original. The original retention expiry is recorded in Unix seconds (as are test times); a connection receipt is a dated observation and cannot renew retention or resurrect deleted data. All current artifacts are considered when `--limit` is absent; unsupported formats are explicitly marked untested.

## Isolation

Each candidate uses a separate network, mount, PID, user, IPC, UTS and cgroup namespace; namespace-local UID65534 and no capabilities. Host private files, service sockets and credentials are not mounted. A 256MiB/128-task cgroup, bounded log file, restricted file descriptors and request deadlines contain each job. No shell interpolates a config. Public endpoint DNS answers are resolved with a deadline; any private/non-global answer blocks execution. Namespace-local nftables only allows those public endpoint IP:port pairs, loopback within that namespace and a public DNS resolver. IPv6-only endpoints and unimplemented dial targets remain untested. A unique host NAT table matches only that namespace's source address; pre-existing firewall tables/routes are never flushed or edited. System CA certificates and pinned core assets are mounted read-only. There is one run lock; SIGINT/SIGTERM stops queued work and lets current bounded jobs clean up.

Run `--self-test` before live input: blocked egress probes plus an allowed HTTPS positive control, pinned binary checksum and a synthetic core launch. Failure is an infrastructure blocker. An initial calibration without the system CA bundle was invalidated; such runs are excluded from the knowledge aggregate. Future release acceptance must retain this distinction.

## What a connection result proves

1. Candidate bytes are accepted by the checksum-pinned Xray binary under `run -test`.
2. A separate runtime clone replaces local inbounds with a private SOCKS listener, pins public endpoint DNS answers, disables control APIs and replaces `freedom` outbounds with `blackhole` to prevent direct-route false positives. Existing proxy outbounds and routing/DNS are otherwise retained. Hash both versions separately.
3. Curl uses SOCKS5 remote DNS, HTTPS certificate verification, no ambient proxy variables and two independent destinations: gstatic `generate_204` and Cloudflare trace. Two expected HTTP statuses pass; one passes only as `partial`.

This proves those HTTPS proxy requests from this Linux vantage, not full custom routing equivalence, UDP/QUIC reachability, Android import, Android VPN operation or future availability. A probe restriction (private endpoint, IPv6-only, missing asset, unsupported driver, custom DNS mapping) is not evidence that the user's config is broken. An eight-second request failure can be transient; it is not an irreversible health decision. There is no preliminary one-second TCP-ping rejection.

## Evidence-backed repair hypotheses

- `uri-json`: represents supported VLESS/Trojan/SS/VMess/Hy2 links as a separate Custom JSON candidate. Unknown or duplicate fields are not silently discarded. KCP, WireGuard extension mappings and other unsupported inputs remain explicit pending cases.
- `trojan-tls-hypothesis`: only for explicit Trojan `security=none`, preserve address/port/auth and supplied SNI; propose TLS with certificate verification. Never assume the server supports TLS. Only a two-destination connection pass recommends it.
- `strict-tls-hypothesis`: Xray26.7.28 `infra/conf/transport_security.go:361` rejects `allowInsecure=true`. Try `false`, preserving other TLS details; never guess certificate pins or enable insecure verification. A valid remote certificate may make the candidate usable; otherwise the error remains unresolved.
- Existing structural JSON repairs remain separate, with immutable originals and strict ambiguity rejection.

The bank's versioned rule catalog is `compatibility/probe-knowledge.json`. Aggregate observations are not automatically trusted code changes. A new repair needs a pinned-source reference, synthetic regression fixture, exact-byte/idempotence checks and a connection test where relevant.

## Current limitations / integration boundary

No Android device/emulator is connected; Android status stays `not_run`. URI conversion is a supported-subset implementation, not a full reproduction of all app settings. The laboratory does not yet write evidence into panel UI or export repaired configurations automatically. The next worker integration must use repository generation/revision leases, access/expiry checks and encrypted candidate storage. No arbitrary uploaded plugin code is executed. SIGKILL or host failure can leave temporary namespace resources; run ownership receipts and automatic crash cleanup are required before promoting this temporary driver to an always-on service. Never start this privileged script from the public panel.
