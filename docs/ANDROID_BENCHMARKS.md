# Reproducible Android and Smart measurements

The production parser, normalizer, compatibility drivers, immutable blob store,
core validators and exact-byte output gates remain shared by ordinary, Smart,
Splash and Ads output. P46 adds a bounded native Android measurement path and an
explicit VLESS share-link rule; it does not replace the existing panel, queues,
retention controls or Ads engine.

## Shared repair

`internal/probe/packet_encoding.go` recognizes **VLESS `packetEncoding=none`** as
native UDP framing. This option is link metadata, not an Xray settings property.
No XUDP/Mux, TLS, key, credential, endpoint or transport is invented. An explicit
XUDP request keeps its existing tested adapter. Other protocols, duplicate values
and unknown encoding modes remain separately diagnosed. Rule provenance is
`vless-packet-encoding.v1.native-udp`; original bytes remain immutable.

Official v2rayNG 2.3.15's VlessFmt/FmtBase does not map packetEncoding to a core
field. Two real previously blocked inputs were independently tested after the
no-op translation on Xray 26.10.31 and 26.11.9 before this rule was added.
Existing generated candidate bytes are unchanged, so unrelated current proofs
and schema25 are retained. Newly prepared candidates still need their own exact
runtime evidence; a successful repair never supplies a connection pass.

## Device pipeline

| Component | Responsibility |
|---|---|
| `android-preflight.py` | Distinguish Termux readiness, authorized ADB and APK control |
| `build-benchmark-matrix.py` | Compile identical member pools at 5/7/10/15/50/100 |
| `benchmarks/core.py` | Validate original bytes, allocate loopback ports, own and clean child cores |
| `benchmarks/http.py` | Verified HTTPS, timings, bounded download/upload |
| `transport-checks.py` | DoH, DNS TCP/UDP and QUIC version negotiation |
| `benchmarks/quic.py` | Full HTTP/3 DoH using HTTP/3 curl through SOCKS UDP |
| `benchmarks/profiles.py` | Named benchmark-only timing candidates |
| `android-bench.py` | Bounded per-case runs, observation samples, thermal guard and private receipts |
| `compare-benchmarks.py` | Report metrics and withhold unqualified profile promotion |
| `benchmark-test.py` | Child cleanup, port graph preservation, strict TLS and proof boundaries |

The agent validates the untouched generated object first. To coexist with an
existing Android VPN, runtime tests remap only loopback listener ports and matching
local relay references, plus the metrics port. Original and executed SHA-256
values are both recorded. This is not an exact-byte APK/VpnService execution claim.
Final import objects must be validated separately on the pinned core and retain
client-compatible listener ports. Complex non-loopback listener graphs are left
untested by this adapter, not labelled invalid configurations.

Only the owned child process is stopped. A parent-death signal closes orphaned
cores. The existing VPN/controller is not killed. Probe workers are limited to
one or two; timed network operations, bounded body sizes and 43 C pre-case thermal
limits contain work. Existing shared phone load is a measurement limitation.
Raw endpoints and diagnostic logs belong in private directories (umask077).

## Run

On the server, obtain a private, immutable array of final member objects and use
the matching compiler:

```sh
python3 scripts/build-benchmark-matrix.py \
  --input PRIVATE-members.json --compiler ./bin/config-probe-plan \
  --output PRIVATE-matrix.json
```

On the authorized Termux device, after independently verifying the official
Android binary checksum:

```sh
python scripts/android-preflight.py
python scripts/android-bench.py \
  --corpus PRIVATE-matrix.json --core PATH-TO-VERIFIED-XRAY \
  --core-sha EXPECTED-SHA256 --output PRIVATE-results \
  --stage full --workers 1 --timeout 6 --observe-seconds 55 \
  --network-label CURRENT-MEASURED-NETWORK
python scripts/compare-benchmarks.py PRIVATE-results/report.json \
  --output PRIVATE-comparison.json
```

Use a fresh output directory and repeat paired tests on each available operator.
For an interrupted run, `--resume` accepts completed receipts only for the same
core, configuration, network label, stage and observation duration. It preserves
their original timestamps and marks them historical; unfinished case directories
are archived before retry. It never refreshes health evidence from an old pass.
A registered SIM name does not establish the active route. Compare current route
observations before and after a series. Never copy an MCI result into an Irancell
or Wi-Fi row. The native agent cannot grant Android permissions, change SIMs,
control app UI or measure protected interfaces without an authorized device
control channel. Those scenarios stay `not_run`.

`--stage full` adds seven public HTTPS destinations, DNS/UDP/QUIC and full H3.
The upload uses generated random bytes, not user data. A valid HTTPS response
and exact submitted/received byte count are required; the upload response is
not a cryptographic server-side payload hash. Short transfer measurements include
setup and do not establish maximum line rate. HTTP responses do not establish
Telegram message/media or Instagram Feed/Reels/Stories functionality. SOCKS setup
and TLS/TTFB timings must not be represented as separately measured remote DNS
and remote TCP handshakes. DNS is tested independently.

## Tuning and qualification

The production strategy stays `leastLoad`, expected3, maxRTT2s, tolerance0.15.
Native Xray 26.10.31/26.11.9 clamps burst intervals below10s. Its periodic sweep
is interval multiplied by sampling: baseline15s*3 is45s; the experimental10s*2
profile is20s. This does not guarantee two-second failover. Cached successful
samples and failure phase affect actual exclusion time.

The alternative is benchmark-only. Partial transfers, failed destination checks,
missing build/start evidence or fewer than three paired repetitions prevent
qualification. Even repeated success requires review of identical members, core,
network, resources and confidence before production promotion. There is no silent
runtime profile change and no automatic security downgrade.

Use `smart-responsive-test.py` in its closed network namespace to inject a failed
peer and verify leastLoad exclusion, DNS and fail-closed behavior. Its synthetic
failure timing is not carrier failover, radio handover or Android battery evidence.

## Coverage and limitations

Keep parse, capability, core-build, startup, connection, bidirectional transfer,
application function and stability results distinct. Report missing server data
separately from core rejection. Native protocols unavailable in the current source
snapshot can be covered by synthetic build fixtures, but real network success
requires actual compatible endpoints. Core changes require a new pinned matrix
and fresh affected runtime evidence. A failing driver must not mark valid raw
inputs deleted, or reuse earlier-core passes as target-core qualification.

Run `scripts/check.sh` for the repository gates. The Android benchmark contract
suite is included. Production activation requires the common deployment lock,
a matching stopped database/blob/key/release backup, source-setting fingerprints
and six executable hash checks. Same-schema rollback restores the former release
without rewinding the database or external deletion journal.

## Official references

- https://github.com/2dust/v2rayNG/releases/tag/2.3.15
- https://github.com/2dust/v2rayNG/blob/2.3.15/V2rayNG/app/src/main/java/com/v2ray/ang/fmt/VlessFmt.kt
- https://github.com/2dust/v2rayNG/blob/2.3.15/V2rayNG/app/src/main/java/com/v2ray/ang/fmt/FmtBase.kt
- https://github.com/XTLS/Xray-core/releases/tag/v26.11.9
- https://github.com/XTLS/Xray-core/blob/v26.11.9/app/observatory/burst/healthping.go
- https://github.com/XTLS/Xray-core/blob/v26.11.9/app/router/strategy_leastload.go

## Network-scoped device quality gate

`build-device-smart.py` creates a private candidate using the existing production
compiler. Eligibility is bounded to an explicit core checksum, observed network,
complete configuration hash and a 60..3600-second receipt lifetime. It takes the
latest matching receipt; a recent failure overrides an older pass. All seven
HTTPS destinations, complete 512 KiB download/128 KiB upload, DNS TCP/UDP, DoH,
QUIC and verified full HTTP/3 are required for this particular device profile.
TCP-only or currently unreachable routes remain stored; exclusion from this
strict profile is not a parser/core-invalid verdict or a deletion rule.
The newly composed Smart bytes require their own core/runtime/traffic tests.
There is no global automatic promotion from a device-specific pass.

Use repeated current-network receipts and a fresh destination directory:

```sh
python3 scripts/build-device-smart.py --corpus PRIVATE-corpus.json \
  --report PRIVATE-results/report.json --core-sha TARGET-ANDROID-CORE-SHA \
  --network-label CURRENT-MEASURED-NETWORK --max-age 1800 \
  --compiler ./bin/config-probe-plan --output-dir PRIVATE-device-candidate
```

The `stable-path-v1` benchmark changes only leastLoad expected to1. It preserves
every member and fail-closed routing. Three paired phone rounds at7 and15 members
did not establish consistently complete transfers, so neither this profile nor
the faster observatory profile is promoted to production. Production remains
expected3 and the established15s/3/5s sampling parameters.
