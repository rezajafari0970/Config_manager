# P50 upload and QUIC investigation (2026-11-12)

**Not activated globally.** Production remains P47/schema26. P49 and P50 are
isolated candidates. The phone became unreachable during the second matrix,
so repeat Android qualification and carrier A/B remain blocked. Do not turn a
successful Linux test into Android or universal network evidence.

## What changed

An opt-in `--experimental-upload-candidate` mode in config-probe-plan appends
an alternative to the original prepared Simple or Smart bytes. The shared
compat driver applies only to plaintext VLESS TCP/RAW HTTP-header streams,
without active mux, flow, packet-encoding, finalmask or chain requirements.
It preserves raw bytes, credentials, endpoint, transport headers, TLS/security,
routing and original candidates. It never creates a health receipt. The new
candidate needs independent exact-output qualification. Default workers do
not use this experimental flag; no database migration or global cache
invalidation was performed.

The proposed settings are TCP mux enabled, concurrency=1, xudpConcurrency=-1,
xudpProxyUDP443=skip. UDP remains VLESS native transport inside the tunnel;
skip does not mean direct Internet egress. Do not promote the earlier
xudpConcurrency=8 variant: it regressed the measured DNS UDP lane.

A new bounded HTTP/3 payload benchmark can measure 2 MiB download and 1 MiB
upload through a SOCKS UDP association using HTTP/3 curl. It preserves TLS
verification, requires HTTP/3 and exact byte counts, limits time/packets/body,
and has no TCP/direct fallback. The target needs a contemporary positive
control before its failures can disqualify a path. Packet forwarding counters
are not physical packet-loss measurements.

## Critical version difference, confirmed by source and execution

Official 26.11.9 outbound handler promotes configured mux concurrency values
0..128 to128. Official26.10.31 only replaces0 with8. Thus configured1 is NOT
effective1 on26.11.9. This is a version-dependent behavior, not an assumed
explanation for the carrier failure.

In a closed namespace with three simultaneous cold TCP streams, the same
configuration used one outer TCP connection on26.11.9 and three on26.10.31.
Both then completed six2MiB download/1MiB upload pairs, with SHA256-checked
upload echoes, plus24/24 exact UDP echoes (32/1200/1450/1440 bytes, ports443
and24445). This proves finite controlled forwarding and constructor behavior,
not Internet stability, TLS performance, radio packet loss or maximum throughput.

Source URLs:
- https://github.com/XTLS/Xray-core/blob/v26.11.9/app/proxyman/outbound/handler.go
- https://github.com/XTLS/Xray-core/blob/v26.10.31/app/proxyman/outbound/handler.go

Run `python3 scripts/tcp-native-udp-test.py /path/to/pinned/xray` as root.
The fixture uses the existing default-drop namespace and loopback peers only.
Actual HTTP/3/HTTPS traffic is a separate certificate-verified Internet lane.

## Phone observations obtained before disconnection

The current Termux route was reported by the user as MCI. Native Android cores
were used; no APK/VpnService, actual app UI or independent Irancell test exists.

- A repeatedly passed2MiB download (~1.4s), but1MiB Cloudflare upload reset
  (~17.5s); independent httpbin echo upload timed out25s.
- B, same protocol family, passed download1.285s, upload1.427s and an exact
  1MiB echo3.193s. Both paths passed short verified HTTP/3 DoH.
- Fragment-all-writes1200/4096 bytes with1ms delay did not repair A. Those
  settings are not applied to generated production configurations.
- One A trial with TCP mux configured1/XUDP8 passed download1.854s,
  Cloudflare upload1.149s and exact echo2.091s. HTTP/3 DoH passed2/2, but
  subsequent DNS UDP failed3/3. This variant is not production-qualified.
- Large HTTP/3 to the chosen speed endpoint received no replies. Without a
  contemporaneous successful endpoint control, this is inconclusive; never
  label it proven packet loss or a server/protocol defect.
- The proposed TCP-mux/native-UDP combination passed the closed Linux fixture
  and real Linux exact-output tests, but its new Android repeat is not_run.
  Additional matrix results stored on the unreachable phone have not been
  collected and are not claimed.

A different original path investigated in P48 was not repaired by its Mux
trials. Family membership alone must not enable this alternative globally.

## Exact Internet candidate

Private Simple candidate SHA256:
b5ecbb013c8628ff98053c2fbd714938565f5a2905d9740db14836d574360e63

Both pinned Linux cores ran these exact bytes in endpoint-restricted namespaces.
Each passed HTTPS, DNS TCP/UDP, Cloudflare/Google DoH, QUIC VN, verified HTTP/3
DoH, full2MiB download, full1MiB upload and post-load HTTPS.
Latest download/upload169/225ms; bundled237/198ms. These are server-side timings
and do not estimate phone/carrier speed. Upload acknowledgement is not a
server-side cryptographic echo; the separate controlled/phone echo lanes are
explicitly identified above. No larger H3 transfer proof is claimed.

## Activation gate

Retain baseline and failed attempts. Reconnect the phone and run interleaved
baseline/candidate/control rounds on both exact cores, with TCP-only mux and
native UDP, cold and warm reuse, concurrency and idle recovery. Require actual
upload completion, exact echo, DNS UDP and verified HTTP/3, fresh same-network
receipts, and final composed Smart tests. Retain the different-core effective
mux behavior in every report. Test current installed APK separately when ADB
or another authorized app-control connection is available.

Stock burstObservatory constructs a nil-body request and only consumes GET
response bodies. It does not certify uploads, HTTP status/minimum length,
persistent per-carrier history or QUIC. leastLoad and GET64KiB cannot
automatically identify every upload-broken path. No invented JSON fields,
server capabilities, automatic upload replays or zero-loss promises are allowed.

OpenAI API review with gpt-6-astra/high completed using sanitized evidence and
official source, with no keys/private endpoints/UUIDs in the prompt. Its
concurrency finding was independently confirmed in both sources and the
controlled runtime fixture. Model advice is not a health result.
