# P29 — Fixed lifetime, direct raw output and visible route evidence

Date: 2026-11-10. Parent: c4e0af88b929834fc9de8d889aa3444ee4c314ef.
Status: deployed with guarded live evidence still pending.

## User decision and implementation

- Latest request supersedes P28 unlimited retention. Schema17 adds immutable
  generation deadline `retain_until`, populated from first_seen + shared retention.
  Production target: retention36,000s and ordinary health cache36,000s, configurable
  independently in Setting. Duplicate arrivals/retests/settings increases do not
  refresh a generation. Evidence TTL600s/retest300s remains independent.
- SQL eligible view, claim/renew/complete and final output admission gate fixed
  expiry. Cleanup handles at most64 expired records per write transaction.
  Accepted records revoke global operational references through the existing
  external authenticated journal/tombstones. Source deletion remains independent
  until the deadline; explicit user deletion remains immediate.
- Raw ZIP replaced by raw.txt (plain text, inline). Original bodies are immutable;
  one LF separates bodies. Legacy authenticated ZIP URLs308redirect. The older all-source /output/archive.zip endpoint also redirects to /output/raw.txt; the Output page exposes the raw link. Raw bodies are privately spooled and current membership rechecked before admission. All formats
  share eligibility/generation and cached/stale work cannot bypass read-time gates.
- Raw imports use the existing deterministic structural repair driver where needed;
  import_rule records the transformation. Acquired bytes remain unchanged.
  Semantic protocol/credential/TLS guessing is not admitted. Unsupported drivers
  remain UNSUPPORTED. Direct-first/unmatched fallback is not complete proxy coverage.
- Initial screening covers three configured DoubleClick domains in one bounded parallel batch, two independent
  controls including dual-stack ipify, repeated waves, separately raw/template.
  Full proof scope is not narrowed. Diagnostic summaries show candidate IPs,
  rotation, same-family comparisons and fresh corroborated ASN/country changes.
  Metadata-only IPinfo/IPWhois drivers cannot produce Residential. Their exact-IP
  TTL caches, request caps, breakers and429 Retry-After handling are independent.
- Dashboard adds counts/pagination and Persian explanations; output links remain
  stable. Intelligence/API outages do not authorize direct fallback or deletion.

## Real canaries and limits

The exact user JSON passed pinned Xray26.7.28 validation unchanged. Isolated raw
and template tests both connected:12/12 control requests passed. In the four-domain,
three-wave raw/template canary24 Ads diagnostic requests returned the same IPv6
candidate2a03:b0c0:3:f0:0:3:41dc:3000. Final measured wall time4,860.470ms, CPU144.674ms and peakRSS66,486,272bytes. Both controls returned that same IPv6; no different Ads exit was observed in this canary.
Fresh IPinfo and IPWhois agreed on DE/AS14061 for that exact IP. Both metadata-only
results remain Unknown network class; candidate attribution remains unverified.

The older separate target from /tmp/shen-ads-detector was rechecked: core validation
passed, current controls and Ads attempts timed out (77,300.383ms). This is a current
failed canary, not proof that its historical residential route became datacenter.
The old experiment assumed one VLESS target and inferred residential from route
ASNs; its observations cannot grant project-wide classification.

Current live blockers are unchanged: no authorized correlated flow observer or
configured specialist+independent exact-IP classification. The Google /sorry
response on a reused TLS stream is an experimental candidate with unverified
origin/semantics, not a documented diagnostic capability. No native forcedIPv6,
UDP/QUIC or exhaustive balanced-path proof driver is supplied. Partial or unknown
coverage stays out of full residential exports. No Android emulator/APK, Iranian
carrier or ground-truth sensitivity/specificity evaluation was performed.

## AI review

OpenAI API returned actual modelgpt-6-astra, completed, input8,981/output1,032tokens
(including8,978cache-write tokens). Estimated cost$0.163855 using published10/1/12.5/50
rates per million input/cached/cache-write/output tokens. Actual billed cost unavailable.
One concrete finding (direct-first default route coverage) fixed and regression-tested.
Only selected source code was sent; no raw config, UUID, secret or capability URL.
The runtime has no AI dependency or AI deletion authority.

## Acceptance and release evidence

Full scripts/check.sh passed (module verification, vet, race, build, pinned core/traffic, Python and JS checks). Final changed raw-output paths also passed a focused race run (96.349s) and fresh vet/build. Migration16→17 is idempotent and preserves original age/source rows. Three legacy migration tests were updated from expected schema16 to17.

Browser mobile390/desktop1440 passed: three residential generations, copy, direct raw display without downloads, per-link rotation, quarantine, manual deletion, restart,10h settings persistence and no horizontal overflow. The waiting-page test now accepts already-completed automatic recovery instead of waiting for a button that has correctly disappeared. Browser text normalizes CRLF for display; exact raw HTTP bytes remain unchanged and are checked separately.

All five worker SIGKILL/recovery integrations passed. Ordinary:22total,16live samples,20core-valid,15connection results passed; includes explicit unsupported/nonpublic cases. Smart/Splash each55profiles×50members; Ads55profiles×51members. Residential:6total (4live+2synthetic), all assessed, concurrency≤2, fiveUNKNOWN/oneUNSUPPORTED; all residential outputs503. Its finite fixture metrics:p50=8,140.012ms,p95=10,466.907ms,meanCPU=133.497ms,peakRSS=66,580,480bytes;1.2configs/min is the fixed5-minute reporting window, including crash/lease-recovery time, not steady-state capacity.

No10-hour wall-clock soak is claimed: fixed-age behavior is verified by advancing stored deadlines and checking actual read/cleanup/retest/source-deletion paths. No ground-truth sensitivity/specificity estimate is available.

Final runtime hashes follow deployment. Private detailed
receipts live under /root/projects/config-manager-production-p29/.local.

## Production verification

{
  "release": "/opt/config-repair/releases/p29-2f64c060f235239d875f",
  "schema": 17,
  "source_count": 29,
  "panel_sha256": "2f64c060f235239d875f0898ed0ee6a5cee9224405f12676436e3a23aaa8fec8",
  "residential_worker_sha256": "865b4019f27b92a2a62263c7ce7fcbb7001fc62d9505202975caec16a659e850",
  "ordinary_worker_unchanged": true,
  "settings": {
    "retention_seconds": 36000,
    "health_ttl_seconds": 36000
  },
  "raw_all_count_at_snapshot": 721,
  "eligible": 0,
  "live_at_utc": "2026-11-10T04:14:55.416634+00:00",
  "queue": {
    "healthy": 122,
    "pending": 1552,
    "tested": 1070
  },
  "total": 2622,
  "missing_memberships": 0
}

All six services active/enabled; executable and driver digests match tested files. Matching stopped-service backup: /var/backups/config-repair/pre-p29-20261110-041336. Deployment process completed in4.71seconds; this is not a zero-downtime claim. External journal was not rewound. Read-only public browser passed at390/1440widths for Setting/Residential/Output; both settings show10hours. General raw returned200/count721 at the snapshot; three residential outputs503/count0. Current pending assessments and live proof limitations remain explicit. Source publication is independently recorded in the private p29-published.json after Git readback.
