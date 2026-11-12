# P52 service quality experiment — 2026-11-12

Based on P51; no production deployment, database migration or public output change.
Production remains P47/schema26. Read docs/SERVICE_QUALITY.md.

Implemented service-quality.v2/service-drivers.v2: scoped complete bidirectional
proof, independent service scoring, expiry, hysteresis, bounded probes/budget,
resume fencing, explicit domains, common Go compiler adaptation and private
snapshots. Owned Xray uses verified loopback API overrides; never attaches to
or stops an unrelated user's core. An imported snapshot is not a controller.

Twenty-nine new regression tests,17 existing device tests and4 payload tests pass.
Closed routing/API fixtures pass on both pinned cores, Linux and native Android:
new streams switch, existing streams remain, service groups are independent,
and unqualified new flows block. Phone API switch measured58.20/89.81ms;
Linux23.90/26.31ms. Cached policy on100 synthetic candidates:median0.092ms,
p950.179ms. A real eight-group/two-route decision took1.505ms. None is a
physical Internet connection or failure-detection latency guarantee.

Live phone:8 completed old-path trials failed;7 fresh-family first-wave trials
had incomplete payloads, followed by one interrupted second-wave attempt.
The v1 diagnostic used3s setup/12s payload and incorrectly treated a WhatsApp
302 as destination failure. V2 follows bounded HTTPS redirects within the
service and allows8s setup/60s exact payload. Two promising fresh members
(Shadowsocks and VLESS/gRPC/REALITY) then passed all4 latest-core v2 rounds:
all service web checks,2MiB down,1MiB up,post-load HTTPS. On bundled26.10.31,
three observed completed repetitions again failed full downloads, including
60s partial HTTP200; the fourth and final phone snapshot run are unconfirmed
after loss of device control. Do not attribute time-separated changes solely
to core version or timeout policy. Do not publish this as a stable phone fix.

Four exact Simple profiles passed latest-core isolated Linux live tests.
The reconstructed service snapshot SHA256
22f192186f62b1fb49e8b8ed5f6ce179e4ce21c5ade2ea1adff7e58a11f3dc99 passed exact
core validation plus all service HTTPS, full payload, DNS TCP/UDP, both DoH,
QUIC VN and verified HTTP3 DoH on BOTH Linux cores. Startup131.75/162.96ms;
download491.07/457.98ms;upload455.15/489.86ms. These are Linux measurements,
not Android performance. Preserve failed sandbox-launch attempts as harness
errors, not core rejections. The corrected launcher enters its cgroup before
network namespace, uses absolute exec paths and the explicit HTTP3 venv driver.

Source audit of official2.3.15 custom builder: stats/policy.system may change,
TUN may be added and process names translated; route/controller behavior still
requires APK/VpnService validation. ADB was empty; Android denied DUMP and
network-interface access. Carrier was user-reported MCI, not independently
verified active SIM/VPN path. Public web/generic upload tests do not prove
app media, authenticated service uploads, calls, gaming, streaming buffering,
packet jitter/loss, other carriers, long-term stability or battery drain.

Private evidence: .local/fixture-{latest,bundled},linux-live.json,
final-linux-{26.11.9,26.10.31}/result.json. Phone evidence lives in
~/shen-p52-service-quality. Preserve interrupted receipts and re-use only
matching finished rounds. Never globally blacklist another-carrier healthy
inputs or claim native v2rayNG delay reflects the new quality score.

The end-to-end owned controller also ran in an endpoint-restricted Linux
namespace after four real v2 member measurements passed. Eight service
overrides were acknowledged/read back. Changing the external network token
stopped the owned core; its PID was gone. This validates lifecycle wiring,
not automatic Android network detection.

Final concurrent-state observation: production moved independently during this
work to p48-617ea4b2e50ce60726e3-r1; all six services were active. P52 did not
perform that deployment. Earlier P47 statements describe the starting state.
Keep P52 isolated; do not roll back or overwrite the concurrently updated release.
