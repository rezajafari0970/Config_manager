# Smart v15 candidate: carrier-local payload completion

**Experimental; not activated.** The 50-member phone pool retained upload failures, so the production P47 release/schema26 remains. See reports/P49-summary.md.

## Why the policy changed
On the current Termux route, reported by the user as MCI, the original plaintext
VLESS TCP HTTP-header path completed short HTTPS/DNS requests but repeatedly
stopped around 38–43 KiB of larger HTTP bodies. Linux completed the same payloads.
The user reports that Irancell works; an independent Irancell handover test was
not performed. This supports a path-dependent transfer failure, not a uniquely
proven DPI, MTU, server, or Android cause. P48's header/Mux/MSS/fragment trials did
not repair the original path; none of those unsuccessful rules was promoted.

Official Xray 26.11.9 and 26.10.31 burst ping implementations read the response
body to EOF with io.Copy only when httpMethod is exactly GET. generate_204 has no
substantial response body, so its success did not detect the observed failure.

## Shared generated policy
internal/smart/health.go owns the policy for normal, per-source, combined,
subscription, Splash and Ads Smart composition. The version is smart.v15;
Smart Ads is smart-ads.v8; schema 27 fences old Smart heads, views and leases.
Ordinary template v9, raw data, source controls and Ads classifier v5 stay intact.

The probe is HTTPS GET https://speed.cloudflare.com/__down?bytes=65536.
sampling=2, timeout=10s, interval=max(15, observed-outbound-count) seconds.
The count follows the final subject selectors, including the final Ads scope,
and excludes unobserved helper outbounds. No direct connectivity check is used.
The main leastLoad remains expected=3, maxRTT=2s, tolerance=0.15.
Only an empty primary pool enters observed recovery: expected=1, maxRTT=10s,
tolerance=0.15, then block. Bootstrap recovery retains its IP-only selector.
Ads has an independent observed gate even for a singleton; chain/DNS/recovery
rules precede application rules. Ads must wait for a usable main carrier.

An affected path is excluded on the device where observations fail; it remains
in the imported profile and can recover after successful observations. No
carrier-specific failure deletes raw input or globally blacklists the endpoint.
This does not implement persistent per-SIM history or retry an interrupted stream.

## Measured evidence
Native Android Xray, two-member pool (original plus independently tested WS TLS):
| Core | Old 204 policy, complete 64 KiB | New policy | 2 MiB down | 1 MiB up | Fresh first verified HTTPS |
| --- | --- | --- | --- | --- | --- |
| 26.11.9 | 4/6 | 6/6 | 1840 ms | 2187 ms | 2825 ms |
| 26.10.31 | 2/6 | 6/6 | 1851 ms | 1843 ms | 5481 ms |

Google, Telegram and Instagram HTTPS pages transferred. Bundled-core Instagram
first returned 302; the subsequent bounded HTTPS-only redirect test returned
200 with the full response. These are web transfer checks, not native Telegram
media, Instagram Reels/Stories, YouTube playback, or logged-in app UI tests.
DNS TCP/UDP and Cloudflare/Google DoH passed in both followups.
QUIC VN is inconsistent: initial latest failed, bundled passed; fixed repeats
were latest 1/3 and bundled 3/3. Retain all failures. Phone HTTP/3 was not run.
Linux exact-candidate latest passed including HTTP/3; bundled's first QUIC VN
failed while complete down/up and DNS passed. Do not claim universally healthy UDP.

Closed-loopback namespace tests on both pinned cores showed old short-request
false health, new 8/8 complete requests, all-down blocking, recovery after good
observations, and admission of a slow but complete path. Controlled degradation
detection was approximately 18.4/38.4s; recovery approximately 28.2/10.9s.
They are synthetic timing observations, not carrier handover measurements.

The exact two-member candidate SHA256 is
00e80e93f220decdaf8cf8a98ad7c156aa434754df2a5d2e02634552399c4ffe.
Phone runs change only local listener/metrics ports to avoid occupied ports;
receipts record each runtime hash. No successful v2rayNG APK import/VpnService
test is claimed: ADB is unavailable. Native-core proof cannot replace that lane.

## Limits that must remain visible
* This is a bounded body-completion gate, not exact content/status validation.
  The official observer does not enforce HTTP status or expected byte count;
  a short 403/503, redirect body or premature clean chunk termination can pass.
* A separate REALITY member passed 64 KiB but failed 2 MiB and upload on the phone.
  Therefore this gate cannot certify uploads, destination-specific policies or
  all applications. Existing server admission still requires payload_v1 complete
  2 MiB down, 1 MiB up and post-load HTTPS; that evidence belongs to its test network.
* maxRTT now measures full 64 KiB completion, not pure network RTT. Slow complete
  paths use recovery. No two-second startup or zero-loss guarantee is established.
* Conditional nominal response-body average is at most 64 KiB/s for larger pools, only if each response is exactly 64 KiB:
  2 nodes ~30 MiB/hour; 5 ~75; 7 ~105; 10 ~150; 15/50/100 ~225.
  TLS, headers, startup, retries and chained Ads carrier bytes are additional.
  The core enforces neither a response-byte limit nor a hard bandwidth/concurrency cap; oversized responses exceed this calculation. Startup sweeps are parallel;
  stock JSON exposes no limiter. Large pools can burden weak mobile networks.
* Scaling intervals controls average probe payload but slows large-pool failure
  detection/recovery. Startup can admit one good sample; two successes are not a universal admission requirement. Samples can remain valid up to four intervals after measurement and can be stale after network changes. There is no
  config-only carrier-change flush or migration of existing TCP streams.
* Cloudflare probe availability is a dependency. Its outage can exclude working
  paths; keep diagnostics and originals. Do not convert that to endpoint deletion.
* UDP/443 remains allowed; successful TCP observation does not prove QUIC.
* The legacy compatibility relay adds no encryption. allowInsecure stays false;
  no credentials, TLS/SNI/keys or server capabilities are invented.

## Reproduction and operations
Run scripts/check.sh using the project's Go 1.26.0 toolchain. The host Go 1.27.1
race object failed at link time; a private clean cache did not fix it. Project
checks use GOTOOLCHAIN=go1.26.0 with private cache/temp and bounded parallelism.
No global toolchain or unrelated cache was removed.

For each official pinned core, set XRAY_TEST_BINARY and run internal/smart tests.
XRAY_TRAFFIC_TEST=1 additionally enables
TestPayloadObserverRejectsTruncationAndRecovers and TestAdsControlledChainTraffic.
The former uses scripts/smart-payload-observer-test.py in a closed namespace. It also verifies that all-down application requests never reach either live fixture origin and explicitly reproduces short 503/200/302 false acceptance, preserving that known core limitation.
Fixtures substitute a controlled HTTP response only for synthetic tests; real
phone/Linux lanes use certificate-verified HTTPS and the generated payload URL.

Before deployment hold /var/lock/config-repair-deploy.lock, stop all six services
and take a matched DB/blob/key/release snapshot. Rehearse schema27 on a private
snapshot; verify user controls/memberships, all six executable hashes, current
ready/export versions and final export validation. Do not open schema27 with old
binaries. Restore a stopped matching snapshot only before new writers run;
otherwise prefer a forward fix and preserve later user/deletion evidence.

A sanitized code/evidence review through OpenAI API gpt-6-astra completed with high reasoning. A first max/pro attempt was incomplete and was not treated as a review. Review findings led to an allowlisted, fixed-version migration, defensive observer/recovery structure checks, explicit closed-envelope validation, stronger all-down proof and negative observer tests. API advice is not runtime evidence.
