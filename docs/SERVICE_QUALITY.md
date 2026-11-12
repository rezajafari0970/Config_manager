# Service-aware Smart experiment (P52)

This adds an isolated, opt-in control layer beside the existing project compiler.
It does not change public subscriptions, the production database, or the user's
running VPN. Production promotion requires the final profile and the Android
integration to pass separately. Neither a successful small GET nor server-side
health alone qualifies a mobile-network route.

## Components and contracts

- `scripts/servicequality/catalog.py`: versioned explicit domains and read-only
  HTTPS drivers for Instagram, Telegram, X, WhatsApp, Google, YouTube, speed tests
  and default traffic. No geosite download or broad shared-CDN capture.
- `policy.py`: independent service decisions, scoped to the exact config, core,
  network label and network epoch. Two fresh, distinct completed rounds and the
  shared exact 2 MiB download / acknowledged 1 MiB upload / post-load HTTPS proof
  are required. New incomplete results revoke old passes. Receipts expire in
  900 seconds; driver/contract changes invalidate old evidence.
- `probes.py`: one core per candidate, two concurrent destination requests,
  sequential payloads. HTTPS allows 15s, connection setup 8s, full payloads 60s.
  Slow complete transfers rank lower; partial HTTP 200 never earns a pass.
  Redirects are bounded to three HTTPS hops, and the final host must belong to
  the service. Uploads are never redirected, retried or replayed automatically.
- `profile.py`: adapts the existing Go Smart compiler's output, retaining
  outbound credentials/transports and earlier scoped rules, including Ads.
  DNS and bootstrap use an independently reachable IP candidate. No direct
  Internet fallback and no blanket UDP/443 block are added.
- `service-smart.py`: measure, resume, compose, plan, export and run. The owned
  runtime uses loopback-only Xray RoutingService balancer overrides with
  readback. New flows change route; established TCP streams stay on their old
  route. A crash/exit stops only the owned core, so stale API overrides do not
  survive the controller. Failed API/readback or changed supplied network token
  terminates that runtime. It never attaches to an arbitrary existing API.
- `servicequality-final-test.py`: validates original export bytes, then runs the
  same parsed profile with isolated listener ports, full web/payload tests,
  DNS TCP/UDP, both DoH resolvers, QUIC negotiation and verified HTTP/3 DoH.

## Selection and resource policy

The initial bounded score weights are reliability 35%, generic download 20%,
generic upload 20%, service request completion 15%, completion variation 10%.
Throughput contributions saturate; a fast but incomplete route cannot win.
These weights are starting heuristics, not empirically established universal
optima. HTTP completion variation is not UDP jitter or packet loss.

A healthy incumbent stays unless a challenger improves by 10 score points for
three distinct measurement waves and the incumbent has been held for 120s.
Polling the same receipt does not create new votes. Stale/failed incumbents bypass
the hold. No eligible candidate yields block, never an untested direct route.
LeastLoad remains in the generated runtime, but a verified service override
pins its selection instead of randomly switching between healthy peers.

Each run has an explicit duration (30..3600s), interval (30..900s), corpus limit
(100 distinct Simple members), two destination workers and one background
candidate worker. The default observed HTTP-body budget is 64 MiB per run;
it may overshoot by one bounded probe. It is not physical wire-byte accounting.
Budget exhaustion stops new probes; expiry still applies. Testing every one of
100 routes continuously has a real data/time cost; do not hide that tradeoff.

## Running

Keep corpora, receipts, manifests and exports private (umask 077). `--network`
is a measured/user-reported network label; `--epoch` must change after a network
change. Android currently denies network-route introspection without additional
permissions. The optional `--network-token-file` connects an external network
monitor; it is not automatic Android carrier detection by itself.

```sh
python3 scripts/service-smart.py measure --corpus corpus.json --core /path/xray \
  --network MCI --epoch SESSION --output receipts --rounds 2
# Only unfinished cases may be resumed; completed receipts keep their ages.
python3 scripts/service-smart.py measure --corpus corpus.json --core /path/xray \
  --network MCI --epoch SESSION --output receipts --rounds 2 --resume
python3 scripts/service-smart.py compose --corpus corpus.json --core /path/xray \
  --network MCI --epoch SESSION --compiler /path/config-probe-plan --output composed
python3 scripts/service-smart.py export --corpus corpus.json --core /path/xray \
  --network MCI --epoch SESSION --base composed/manifest.json \
  --receipts receipts --output export
python3 scripts/servicequality-final-test.py --config export/candidate.json \
  --core /path/xray --network MCI --output final-check
python3 scripts/service-smart.py run --corpus corpus.json --core /path/xray \
  --network MCI --epoch SESSION --base composed/manifest.json \
  --receipts receipts --output owned-runtime --duration 600 --max-mib 64
```

`run` exposes its allocated SOCKS port in its private runtime-addresses.json.
It does not install a VpnService, rewrite v2rayNG's active config or bypass an
existing VPN. APK integration must prevent recursive tunneling of Termux traffic.
An exported JSON is a fixed snapshot, not an embedded Python controller. Its
expiry is enforced at selection/export time, not magically by stock Xray after
import. Do not publish snapshots as perpetually self-testing configurations.

## Evidence boundaries

Public web probes and a Cloudflare upload prove only their own transactions.
They do not prove Instagram uploads, Telegram media, WhatsApp calls, streaming
rebuffer rates or game latency. Those need authorized app/controlled-endpoint
drivers and actual device integration. Those categories remain explicitly
unmeasured; unclassified IP traffic uses the qualified default route.
Domain matching cannot reliably classify IP-based MTProto, shared CDNs, ECH,
encrypted DNS, calls or game UDP. No real packet-loss or jitter claim is made.

Use `servicequality-test.py`, `servicequality-runtime-test.py --core ...
--output ...`, the existing benchmark/payload tests and final export checks on
both pinned cores. Fixture observations are not carrier-performance evidence.
Reassess contracts and final exports when the target core changes.
