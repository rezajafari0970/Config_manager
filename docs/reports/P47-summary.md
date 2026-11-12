# P47 activated Smart recovery and video follow-up

Implementation commit: 6628dcf70db5de97e7f24f25634689797be55cd0.
Release: p47-b51b595c6ec184771530. Schema26. See ../SMART_RECOVERY.md
for architecture, boundaries and repeatable commands.

The 296-second recording shows short native ping success alongside browser/media
stalls and strongly asymmetric Speedtest throughput. The opening Compose
Constraints exception is a UI-layout error; no attributable app frame is visible.
It was not fixed or reclassified as an Xray configuration error.

Changes add scoped observed leastLoad recovery, shared complete-payload evidence,
bounded stage deadlines, version fencing, and conservative JSON/XHTTP/socket
preservation. No primary timing promotion, direct fallback, TLS bypass or raw
mutation was introduced. Ordinary v9 and Ads classifier v5 remain.

## Measured evidence

On each of Xray26.11.9 and26.10.31, a closed TCP relay adding1150ms each direction
made the baseline fail0/3 while the generated candidate succeeded3/3; fast3/3,
DNS A/TXT TCP+UDP and observed all-dead blocking passed. Historical receipt scope
text says10s/1; that describes the baseline fixture only. Generated candidate
timers were production15s/3/5s. The harness label is corrected, original receipts
are retained unchanged.

Linux A/B held an8MiB download at64KiB/s:127.79s baseline,127.72s candidate,
36 successful concurrent HTTPS checks each. This does not show higher peak speed.
Post-deployment exact Simple, Smart, Splash and Smart Ads object bytes passed
both cores and full2MiB down/1MiB up, TLS, DoH, DNS TCP/UDP, QUIC VN and HTTP/3.
The first Ads26.10.31 attempt had an incomplete site check despite passing
transport/payload; one same-byte repeat passed. Both results are retained.
An earlier export check stripped terminal whitespace; the later exact-byte
receipt is authoritative. An ordinary array item, if selected, is explicitly
serialized as one importable object rather than claimed to be the array bytes.

Final release/race/vet/build, Python, core-enabled Smart, migration, queue SIGKILL
and mobile/desktop browser checks passed. Initial in-progress envelope tests
failed while composition changed; final suites passed after implementation and
assertions were updated. Initial browser invocation lacked existing dependency
environment variables; the corrected invocation passed without reinstalling.
Queue synthetic receipts are not traffic proof. Bootstrap fixtures used real
TLS/HTTP2 DoH without endpoint host pre-seeding; system DNS attempts were zero.

## Preservation and limitations

Stopped migration under the shared deploy lock preserved33sources,886members,
source controls, settings and raw-membership hashes. A matched recovery set is
in /var/backups/config-repair/p47-20261112-011313. All six services and six rebuilt
executables were checked. DigitalOcean readiness passed; no all-provider
transaction claim is made. No owned test namespaces remained.

Phone was offline for P47. APK UI/VpnService, actual Telegram/Instagram features,
carrier handovers, long-term stability and phone battery/CPU remain not_run.
The three-member candidate uses previously tested MCI members, but its new
P47 evidence is server-only. Corpus-wide P46 receipts are historical and are not
relabelled new. Active queue requalification can temporarily withhold exports;
do not publish short-only or stale Smart evidence to hide that state.

OpenAI API review used gpt-6-astra pro/max, store=false; sanitized code/evidence
only, no secrets or private chats. Recommendations were individually tested.
Private receipts and user deliverables remain in .local; no credentials are
committed. The new quality stage adds bounded bandwidth and time per candidate.
