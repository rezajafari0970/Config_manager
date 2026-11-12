# P51 device payload qualification and native-delay boundary

Production remains P47/schema26. P51 changes device assessment/export tools only;
it does not deploy P49/P50, blacklist a carrier-specific path globally, edit the
installed APK, or force an artificial negative native ping.

## Implemented

A shared benchmarks.quality contract requires complete 2 MiB download, complete
1 MiB upload and verified post-load HTTPS. android-benchmark.v2 records its version
per case; resume refuses old-contract receipts without changing their timestamps.
device-quality.v2 and benchmark-comparison.v2 use the same predicate. Incomplete
transfers retain diagnostics, but failed local upload writes are no longer shown
as successful throughput. Invalid/boolean/nonfinite receipt times do not qualify;
a newer interrupted attempt overrides an older successful observation.

The private device export emits unchanged qualified Simple objects, their JSON
array, and a separately unqualified composed Smart candidate. All seven HTTPS
destinations, DNS TCP/UDP, both DoH checks, QUIC VN and verified HTTP/3 remain
required. Scope is exact config, core checksum, explicit network and receipt age.
Fresh directories prevent stale candidate reuse. Original raw/server/security
settings are preserved. This tool does not rewrite public subscriptions.

## Recovered phone evidence

Android16, native Termux, user-reported MCI. Both SIM operator names were visible,
but protected routing inspection was denied; this is not an independent active-SIM
or APK/VpnService verification. ADB had no attached device. Existing unrelated
phone controller was left running, so CPU/network measurements have shared load.

The twelve new P50 native-UDP repeats finished despite a temporary control-channel
outage. Path A baseline upload failed4/4; TCP-mux/native-UDP A succeeded only1/4.
Control B passed complete download, upload and independent 1 MiB exact echo4/4.
All twelve cases passed both short HTTP/3 DoH checks and all three DNS UDP checks.
Thus native UDP avoids the earlier measured XUDP DNS failures, but does not make
the upload alternative reliable. Do not activate that alternative globally.

Four final Simple profiles were then assessed on both pinned cores:
- A passed all seven HTTPS sites and full download, but failed upload on both.
- B passed all device quality gates on both.
- WS/TLS transferred payload on latest but timed out on YouTube; bundled also
  failed some destinations/download and transport checks.
- The original legacy path failed complete download and upload on both.

All eight built; only one distinct path qualified for this current-network snapshot.
The supplied family itself is not invalid: B belongs to the same legacy family.

## Final device Smart

Only B was eligible. Composition preserves leastLoad and fail-closed routing, with
one actual member, so these tests do NOT establish redundant-server failover.
Three final native Android repetitions per core completed all seven HTTPS sites,
2 MiB down,1 MiB up,post-load HTTPS,DNS TCP/UDP,DoH,QUIC VN and HTTP/3 DoH.
First verified HTTPS:1.010–1.066 seconds. Download:1.305–3.306 seconds.
Upload:1.159–1.328 seconds. These finite timings include protocol/setup overhead,
are not maximum line rate or pure network RTT, and do not prove all app functions.

One bundled attempt disappeared before writing its receipt. Its incomplete
directory was archived, and only that case was retried. Previous completed cases
kept their original timestamps. Do not hide this interruption as a stability pass.

Exact delivered Simple and Smart bytes also passed both Linux cores in
endpoint-restricted namespaces, with complete payload and DNS/UDP/HTTP3 proof.
A Linux Smart upload took13.803 seconds in one run; retain that variation rather
than presenting uniformly low latency. Phone tests remap loopback ports and
canonicalize JSON; original and runtime hashes are recorded.

Simple SHA256:862cd523e4711d946d25faa43d58992445712abb1ecb475aa3aa9158f104691b
Smart SHA256:b1168c228ef8f1d1912d71109aa82ba676fd51f1acfdc80908dc421b7f078fac

## Why stock negative ping cannot be promised by JSON

Official AndroidLibXrayLite Lib40 commit
2ffe5f794a32ba15b122e177b5acffc6f11c4959 and current source remove inbounds,
routing,DNS and observation apps for MeasureOutboundDelay. Only outbound,
dispatcher and log apps survive. The probe issues two GET requests to one
caller-supplied URL, consumes the body, accepts200/204 and returns the shortest
successful duration. There is no upload body or multi-destination/QUIC policy
specified by imported JSON. A successful short request is therefore not false
latency; it is insufficient evidence of complete connectivity.

Changing the app's delay-test URL to a bounded substantial download may expose
download stalls, but cannot qualify uploads or every destination. Blackhole-first
or blocking the probe URL would create a forced negative, including on a working
carrier, and is not the implemented solution. To make the official UI return -1
after a real multi-stage failure, the client test path must run that assessment;
an external evaluator cannot silently alter the stock app's internal result.

Source:
https://github.com/2dust/AndroidLibXrayLite/blob/2ffe5f794a32ba15b122e177b5acffc6f11c4959/libv2ray_utils.go
https://github.com/2dust/AndroidLibXrayLite/blob/main/libv2ray_utils.go

## Validation and limits

17 device/export regression tests and4 existing payload-contract tests passed;
3 existing P50 QUIC-contract tests also passed before the independent P51 branch
was based on canonical source. The canonical compiler built with Go1.26.0.
No Go runtime/database/public-service changes were made.

APK import,VpnService,real Telegram/Instagram UI,Irancell/Wi-Fi/handover,
large HTTP/3 payloads,radio packet loss and attributable battery drain remain
unverified. QUIC VN is distinct from the completed short HTTP/3 DoH checks.
No zero-loss,zero-lag,all-network or permanent-health claim is made. The one-member
device snapshot is a measured alternative path, not repair of the failing endpoint.
