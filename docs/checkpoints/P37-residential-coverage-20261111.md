# P37 — residential assessment coverage and honest same-stream diagnostics

Status: deployed and verified; release p37-939e1eb750913ba8a2bb.

The request is to assess every acquired configuration and publish only proven
Residential Ads configurations. An Ads/control IP or ASN difference is not a
Residential classification. The existing strict policy, trust registry and
separate ADS_DIFFERENT pool remain unchanged. No synthetic or AI-generated
observation, specialist result or trust capability is installed.

Changes:
- Unsupported/planning-failure completions use the independent Residential plan
  identity and explicit core ID. This removes coupling to publication changes.
  The current Go publication constant is 26.11.9 and was already admitted; it
  must NOT be described as the proven cause of current HTTP400 failures. The
  Python publication adapter separately targets26.10.31. An initial diagnosis
  conflated these two constants and was corrected before deployment.
- Never-assessed records precede unqualified retries inside the acquisition queue.
  Existing accepted-member revalidation priority and per-source fairness remain.
- A fully framed 3xx Ads response on an open TLS socket may continue the existing
  same-socket diagnostic. Location is never followed and no second socket is
  opened. A certificate that does not authenticate the diagnostic authority is
  explicitly labelled diagnostic_origin_unverified. The returned IP remains
  CANDIDATE_ONLY; origin and diagnostic semantics do not become proven.

Regression tests cover unsupported completion identity, first-assessment queue
fairness and 301 same-socket continuity without redirect, reconnect or promotion.
The independent Xray 26.11.9 / 26.7.28 / 26.5.9 matrix and sing-box driver remain;
raw and template evidence are not combined across cores. Production uses crr;
worker integration uses crt and must not overlap another crt fixture.

Initial production observations: schema21,33 sources; all560 artifacts in the
coverage snapshot had a Residential record. The independent backlog contained
9184 nondeleted records,8911 never assessed, and zero proven Residential or
ADS_DIFFERENT members. These counts are time-specific and not a count of healthy
configs. Removing only old regenerable Go build-cache entries raised disk
headroom from291MiB to about1.6GiB, without deleting configurations or backups.

Limits: no administrator-trusted egress observer or specialist exact-IP provider
is configured. Built-in same-TLS diagnostics cannot prove undocumented response
semantics; missing IPv6/UDP/QUIC and complex-path evidence remains unknown. The
implementation therefore cannot honestly claim all residential inputs were
identified or manufacture a country such as MO. Unknown is not Datacenter.

Retention/publication contracts remain: qualified members are age-exempt while
proof remains fresh; uncertainty removes them from active output pending retest;
confirmed loss uses existing spaced evidence/circuit-breaker and deletion ledger.
The original raw, templated JSONL and durable rotating-single outputs share the
same strict membership. No database migration or journal rewind is authorized.

## Final verification

Integrated with deployed P36 commit95d579f before final checks. Full check passed:
Go module verification, formatting, vet, race (panel609.037s), all builds and Python/
JS suites. A corrupt system race object was bypassed with the existing isolated
Go1.27.1 toolchain. The earlier P35-base check was superseded after P36 activation.
The final worker fixture recovered SIGKILL and completed7/7, including4 live
canaries, with max2 active in190.38s. Its synthetic outcomes are not live proof.
Browser390/1440, copy, outputs, quarantine and deletion checks passed. A stale
six-link fixture expectation was corrected to eight (JSON array was previously
added alongside JSONL/single/raw for each of the two distinct pools).

Production activation preserved all source/policy settings and schema21; executable
and driver hashes matched verified artifacts. All six services active;707 distinct
acquired raw hashes had records, missing0. One fresh completion was observed within
25s of activation; this is not a throughput benchmark. Queue9679, awaiting9411;
verifiedResidential0 and attributableAdsDifferent0. All four Residential formats503.
Previous release and matched private DB/key/immutable-blob backup are retained;
rollback never rewinds external revocations. Main services were not reconfigured.

Original Stockholm canary: all three pinned Xray versions attempted; selected
report had60Ads timeouts,0candidateIPs and0passed control requests. StatusUNKNOWN,
independent_controls_incomplete, ineligible. Historical MO observations cannot be
asserted as current or Residential. No API model was used to invent network proof.
