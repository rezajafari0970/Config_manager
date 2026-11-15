# Residential: sole active policy, P66 / ads-egress.v9 / schema37

This document supersedes earlier policy descriptions. Historical reports remain
records of their own releases, not selectable strategies. `internal/egress.Evaluate`
is the only publication classifier; `internal/adsroute` is a discovery adapter to
the same route observation engine and has no publication authority.

Residential means the project's verified, destination-specific Ads split. It does
not imply a household ISP, a particular ASN, one fixed country/IP, all Google Ads,
UDP/QUIC or all future requests. A complete named profile qualifies independently;
other profiles' shared or unobservable paths remain visible as partial coverage.

## One admission pipeline

1. Register current collector identities, choose one canonical raw representative,
   preserve its bytes and use protocol-specific preparation with pinned compatible
   Xray/sing-box drivers. A cosmetic name/source duplicate cannot create a new test.
2. Test both exact raw and generated-template runtimes, with strict TLS and bounded
   isolated processes. Startup incompatibilities try compatible pinned cores;
   ordinary network timeouts do not multiply across every core.
3. For every member of a named profile, require at least two paired waves, a valid
   same-stream diagnostic source observation, stable same-family witnesses from
   two control groups including ordinary Google, fresh nonces and exact bindings.
   Shared/rotating controls and contradictory waves cannot fabricate a split.
4. Require verified, bounded HTTPS retrieval of that profile's bootstrap JavaScript
   in both runtimes. The compiled delivery registry and profiles are included in
   the policy digest. Host control measurements can corroborate failure only.
5. Apply all-Ads Iran guards, including other attempts and minority-country conflict;
   then independently apply the existing selected-country ANY publication rule.
6. Publish unchanged tested bytes from the same eligibility snapshot in raw.txt,
   configs.jsonl, configs.json and next.json. One collector identity produces one
   full-export row; different identities can legitimately emit identical templates.

| Profile | Required routing destinations | Necessary bootstrap asset |
| --- | --- | --- |
| doubleclick.v1 | googleads.g.doubleclick.net, pubads.g.doubleclick.net, securepubads.g.doubleclick.net | securepubads.g.doubleclick.net/tag/js/gpt.js |
| ads-serving.v1 | pagead2.googlesyndication.com, adservice.google.com, www.googleadservices.com | pagead2.googlesyndication.com/pagead/js/adsbygoogle.js |

Verified sharing across every profile member in both variants yields NO_SPLIT.
Missing, unsupported, conflicting or mixed evidence stays UNKNOWN/UNSUPPORTED;
ASN, country, latency, timeout and general IP-check sites cannot fill evidence gaps.
The peer diagnostic remains an undocumented observation of the measured TLS
stream, not a signed statement about arbitrary actual advertising requests.

## Ads availability and removal

The asset check validates status200, trusted TLS, complete bounded body, JavaScript
MIME, expected marker, size and digest. It makes no ad requests, clicks or impressions.
Downloading this script does **not** establish ad fill or device rendering. SDK
configuration, consent and inventory can prevent display independently of the VPN.
Actual render/fill is explicitly not measured by this server-side driver.

A credible transport failure or authenticated blocking response in both runtimes,
including SOCKS handshake failures ([curl error97](https://curl.se/libcurl/c/libcurl-errors.html)),
for both profile assets, counts as unhealthy only when the same assets are freshly
available through the independent host control. Parser errors, local resource
pressure, missing runtime files, 429, 5xx, authenticated partial HTTP200 downloads or unfamiliar HTTP200 content
and failed host controls stay uncertain.
Healthy controls are cached for at most20s (5s on failure), and negative corroboration must still be within30s of the decision, not report start. Stale optional controls cannot erase valid VPN-positive evidence.

On the first credible unhealthy result, or verified NO_SPLIT for an accepted input,
withdraw it from every output, including aliases. After three independent reports
spaced at least30s with the same failure reason, remove usable records, memberships,
candidates, reports and source-response references globally. Retain a minimal
identity exclusion/tombstone so recurring source imports cannot resurrect it.
Successful eligible recovery clears pending retirement. Unknown does not count
as a negative. Confirmation has a10min window measured from the first confirmation; a reason change starts a new series. An expired confirmation series resets its hits but does not lift identity withdrawal without eligible recovery.
Confirmed Iran retains the existing stronger, independent guard and external
revocation semantics. All-Ads minority Iran conflicts immediately withdraw all
output types without pretending that conflict is confirmed country consensus.

Healthy accepted entries are exempt from ordinary age cleanup but require periodic
fresh Ads health/routing proof; stale evidence cannot publish. Completed initial
classifications cache by identity and policy. Accepted safety renewal, bounded
UNKNOWN follow-up, local infrastructure repair, and an explicit policy change are
separate from duplicate initial classifications. A route-qualified input with
unproven asset delivery keeps a bounded, backed-off recovery schedule instead of
being forgotten after the initial retry budget. Explicit transient DNS and driver subprocess faults also remain recoverable after deep attempts, with the existing bounded backoff. Generic route uncertainty and completed negatives keep their finite/cache rules. The UI has no processing buttons.

## Modules, speed and recovery

| Component | Responsibility |
| --- | --- |
| profiles/targets registries and route_map.go | Shared evidence and named-profile decisions |
| delivery.go / delivery_targets.json / egress-delivery.py | Necessary Ads asset availability |
| protocol preparation and pinned core drivers | Exact raw/template compatibility |
| egress_retirement.go / Iran guard | Independent, durable safety withdrawal/removal |
| egress_inventory / egress_work | Current unique scope and bounded automatic follow-up |
| egress_rescan.go / live-throughput observer | First assessment versus definitive timing |
| worker resource/lease/watchdog drivers | Admission limits, crash recovery and fencing |
| common eligibility/export views | Consistent panel and four exact-byte outputs |

Configured worker ceiling31 retains CPU/RAM/I/O governors and namespace/cgroup
limits. After ten seconds of recovery, admission grows by bounded steps of two
to four jobs; every new pressure sample can still reduce it immediately. Idle slots above the reduced capacity release their helper, namespace and country caches after their current job ends; recovered capacity starts a clean helper. Iran,
pending spaced-removal confirmations and still-valid proofs within120s of expiry
are urgent. Already expired proofs remain withdrawn and use follow-up recovery;
their stale deadline cannot monopolize initial capacity forever.
Routine renewal uses reserved follow-up capacity and cannot monopolize a nonempty
initial-assessment lane; idle lanes can still steal work. No proof expiry is extended. Near-expiry work overrides a stale future next_run; completion schedules from the actual proof deadline. Ordinary follow-ups are FIFO so unaccepted recovery cannot starve behind renewals.
New-input fast checks use two paired3s waves. Accepted renewals and incomplete
initial probes use two sequential8s waves with a third adaptive recovery wave
only when required evidence is incomplete, including missing same-family paired
controls despite successful source echoes. Assets have a6s initial /8s deep completion budget;
valid partial progress cannot become a destructive health failure. Optional service reachability
and generic bulk transfers leave the publication critical path; Ads asset retrieval
and route observations overlap within the same runtime and join before teardown.
This removes redundant work without relaxing the evidence necessary for acceptance.

Targets are1000 unique inputs within600s and2000 within900s. Measure both first
assessments and definitive results from immutable reset-cohort timestamps. Retries,
archive entries and UNKNOWN must not inflate the definitive count. Fewer than2000
actual cohort members cannot prove the2000-input target. See the live receipt for
measured results; configured concurrency is not a throughput guarantee.

## One-time reset and deployment

The user's P66 instruction authorizes exactly one new Residential reset. Atomic
schema36 clears old test receipts/statistics and seeds current unique plus retained
safety inventory. Preserve source controls/raw inputs, country selection, ordinary
settings, manual exclusions and sticky Iran protection. Durable feature/schema
markers prevent repetition on restart or unrelated schema repair. Schema37 is a non-reset hardening migration: preserve the original reset clock, proofs and settings; retain pending withdrawals while discarding their old confirmation counts, and correct definitive totals using a durable unique-input settlement ledger. Renewals increment attempts, not unique definitive counts. Older binaries must reject schema37. Generic version reconciliation only invalidates stale proof;
it must never reset statistics implicitly.

Deploy with the shared lock, matched stopped database/blob/key/unit/release backup,
copied-database rehearsal and six tested matching binaries. Validate full checks,
real canaries, all four exports, isolated crash/watchdog/UI recovery, then the live
cohort benchmark. No AI service is a runtime dependency or network-evidence source.

P66 is active with schema37. The measured throughput targets were not met; see
[the final report](reports/P66-report-fa.md) and its validation receipt. This policy
is an admission/lifecycle specification, not a claim that every current input has
already completed or that actual advertising display has been measured.

P67 maintenance keeps ads-egress.v9/schema37 and the original reset epoch.
Both queue summaries use the same urgent deadline rule. The attempted inventory
CTE optimization regressed the complete live API and was rolled back; the original
snapshot query plan is retained. No eligibility cache is introduced.
The cohort panel distinguishes definitive first assessments from later settlement.
No classifier, country selection, proof TTL or tested raw/template bytes change.
Activation and validation status belong to the P67 receipt.
