# P28 — Residential Google Ads: implementation, tests and deployment

Date: 2026-11-10 UTC. Parent: `660cf470f62639bcf23cfd34d8f5667600150448`.
Status: **deployed with closed publication; live residential evidence pending**.

## نتیجه و وضعیت مراحل

زیرساخت تشخیص مستقل، نگهداری بدون سقف سنی برای موارد پذیرفته‌شده، قرنطینه،
حذف با تأیید چندمرحله‌ای و سه خروجی مشترک روی سرور فعال شده است. این گزارش
به‌معنای تکمیل تشخیص زنده برای تمام پروتکل‌ها یا تأیید رزیدنتال‌بودن ورودی‌ها نیست.
در بررسی ثبت‌شده، هیچ عضو دارای شواهد کافی برای انتشار رزیدنتال نبود.

| مرحله | وضعیت واقعی | نتیجه / محدودیت |
| --- | --- | --- |
| بررسی آزمایش قبلی | انجام شد | `probe.py`، `report.json` و `fast_probe.py` بررسی شدند؛ تفاوت ASN یا پاسخ تشخیصی فاقد اعتبار مبدأ مجوز پذیرش نیست |
| قرارداد و سیاست مستقل | پیاده‌سازی و تست شد | سلامت، خروجی مشاهده‌شده و نوع شبکه مستقل؛ شواهد امضاشده، محدوده و زمان اعتبار جدا |
| ورود همه منابع | فعال | در snapshot زنده، هر ۲۹ منبع حفظ شده و هیچ عضویت جاری از صف جا نمانده بود |
| آزمون هسته و شبکه | بخشی عملیاتی | Xray ثابت و آزمون TCP ایزوله؛ پروتکل ناسازگار `UNSUPPORTED`؛ پوشش کامل IPv6 اجباری، UDP/QUIC و تمام مسیرهای Balancer پیاده نشده |
| اثبات خروجی و نوع IP | درایورها آماده؛ تأیید زنده مسدود | مشاهده‌گر مجاز و منابع تخصصی مستقل روی این استقرار پیکربندی نشده‌اند |
| نگهداری و قرنطینه | پیاده‌سازی و تست شد | سقف عمومی ۲ ساعت برای پذیرفته‌شده‌ها اعمال نمی‌شود؛ انقضای شاهد فوراً انتشار را متوقف می‌کند |
| حذف و جلوگیری از بازگشت | پیاده‌سازی و تست شد | epoch، نسخه رکورد، نشان حذف و دفتر مستقل از backup؛ اختلال مشترک باعث حذف انبوه نمی‌شود |
| سه خروجی و رابط | فعال و بررسی زنده شد | JSONL، تک‌JSON چرخشی، ZIP خام با حفظ بایت؛ اکنون هر سه صفر عضو و HTTP503 بدون جایگزین |
| آزمون پذیرش | سناریوهای کنترل‌شده موفق | نتیجه آزمایش مصنوعی، مدرک رزیدنتال زنده نیست؛ Android و اپراتورهای ایران آزمایش نشده‌اند |
| بازبینی API | انجام شد | دو پاسخ کامل GPT-6 Astra و یک اجرای ناقص؛ هزینه تخمینی کل $1.30336 |
| استقرار | انجام شد | schema16، شش سرویس فعال، منابع و تنظیمات حفظ شدند؛ نسخه نهایی R2 |

## Evidence boundaries and scope

Read [RESIDENTIAL.md](../RESIDENTIAL.md) for contracts and private adapter setup.
The existing experiment under `/tmp/shen-ads-detector` had a path that could
assign a residential storage bucket from repeated differing ASNs even while its
residential result was UNKNOWN. That conclusion is not imported as evidence.
The mismatched-Host diagnostic on an Ads TLS connection has no established
capability/origin/response-semantics guarantee. Its IP remains an unverified
candidate. Separate control IPs are never attributed to Ads.

Policy `shen.residential.v1`, domain revision
`google-ads-explicit.2026-11-10.1`, explicitly lists:

```
pagead2.googlesyndication.com
googleads.g.doubleclick.net
pubads.g.doubleclick.net
securepubads.g.doubleclick.net
www.googleadservices.com
adservice.google.com
app-measurement.com
www.googletagservices.com
www.googletagmanager.com
admob.com
```

This finite list is not a claim to cover all Google Ads. The default acceptance
matrix requires both raw and common-template variants, every identified selectable
path, IPv4 and IPv6, TCP/UDP/QUIC, and three fresh waves. A missing cell or unresolved
path cannot qualify for full public output. The policy can represent these scopes;
the shipped live probe does not execute the entire matrix. The initial TCP screen
uses two Ads hosts and two independent controls; deep screening expands the Ads
host list when configured or rechecking a previously accepted member. Safe
`/robots.txt` requests do not intentionally create an ad impression or click.

An authorized server-flow observer must prove correlation of the exact test with
the actual egress flow. Ed25519 receipts bind scope, hashes, nonce, stream identity,
time, response digest and IP. The owner verifies the receipt and registered
capability. Signing a guessed IP is not proof. Fresh exact-IP intelligence requires
two independent configured groups, at least one specialist, with explicit class,
source, time, country and ASN; disagreement is Unknown. The trusted collector's
domain-separated HMAC prevents untrusted completion data from forging provider
facts. It does not guarantee a provider's accuracy. Neither trusted observer nor
specialist providers are configured on this deployment.

## Acceptance results

Unless labelled live, results below use deterministic controlled clocks and
synthetic signed/provider data. They test decisions and lifecycle behavior, not
real residential classification. All listed tests passed.

| Scenario | Evidence / test | Verified result |
| --- | --- | --- |
| More than 2h, source deletion and GC | `TestResidentialRetentionTTLSourceDeletionAndGC` | Accepted raw/template references survive normal TTL, source removal and GC; exact bytes preserved |
| Three exports and concurrency | `TestResidentialExportParityExactRawAndConcurrentRounds` | Equal eligible set/generation, exact raw ZIP entries, 16 concurrent requests over durable rounds, no repetition within a round |
| Independent source links and restart | `TestResidentialSourceRoundsIndependentAndDurable` | Per-link rounds persist across store reopen and do not consume each other |
| Residential to Datacenter / connection failure | `TestResidentialFreshNegativeConfirmationAndSharedCircuit` | First credible negative quarantines; spaced fresh confirmations required for destructive revocation |
| Rotating egress / equal control and Ads | `TestRotatingMixedExitCannotQualify`, `TestQualifiedAndAllResidentialControls` | Mixed nonresidential rotation rejected; all-residential equal exits are not rejected just for equality |
| Conflict, provider outage and uncertainty | `TestResidentialUnknownConflictOutageNeverDeletes` | Quarantine/retry without treating missing/conflicting information as a destructive negative |
| Shared infrastructure failure | negative-confirmation and policy tests | Independent controls and destructive-action circuit prevent bulk deletion |
| Evidence expiry at request time | residential export/retention tests | Expired output hidden without waiting for scheduler; no old fallback |
| IPv4/IPv6, UDP/QUIC, domain and raw/template scope gaps | `TestEvidenceBoundaries`, preparation tests | Missing matrix coverage is PARTIAL/UNKNOWN and cannot publish; not a live transport-success claim |
| Policy/template changes, disable and startup | `TestResidentialPolicyChangeDisableAndStartupFence` | Dependent proof invalidated; startup fence applies before listeners |
| Stale worker and restored DB | `TestResidentialRevocationStaleJobsAndBackupReplay` | Old completion cannot resurrect revoked configuration; independent deletion journal replay wins |
| Crash, truncation, partial append | `TestResidentialJournalCrashReplayTruncationAndRestart`, `TestResidentialPartialJournalAppendIsRecoverableOnlyBeyondWatermark` | Missing committed journal fails closed; only an uncommitted tail beyond authenticated watermark may be repaired |
| Readmission | `TestResidentialAutomaticReadmissionRequiresNewEvidence` | New epoch and genuinely post-revocation observations required |
| Forged provider facts, unbound negative and manual aliases | `TestResidentialForgedFactsUnboundNegativeAndManualAlias` | Authenticated facts and owner-derived hashes/paths enforced; new labels cannot bypass manual deletion |
| Already prepared legacy bytes / 304 | `TestResidentialLegacyOutputPreparedBeforeRevocationIsBlocked` | Final fence blocks old buffered output and conditional response after revocation |
| New UNKNOWN result versus ordinary exports | `TestResidentialUncertainNewAssessmentDoesNotInvalidateUnrelatedOutput` | R2: initial uncertain assessment does not bump membership generation and invalidate unrelated long exports |
| Independent result axes | `TestHealthDoesNotEraseIndependentEgressEvidence` | Connection uncertainty does not erase separately verified egress/network evidence |
| Signature replay | `TestAttestationCannotBeReplayedAcrossScope` | Receipt cannot be reused for another configuration/runtime/domain/family/path/wave |

Full `scripts/check.sh` passed with pinned core and traffic fixture environment:
format/module verification, vet, race tests, builds, 15 probe-driver tests, four
IP-intelligence tests, three residential-probe tests and client JS regressions.
Full panel race run took 526.322s. A subsequent SQL/index refinement passed all
panel and policy tests (13.203s), followed by focused race (29.103s) and vet.
The final R2 generation correction passed all Residential race tests (31.707s)
before its panel binary was rebuilt and deployed. No later executable source
change was made without this corresponding verification.

Worker integrations passed for ordinary conversion, Smart, Splash, Smart Ads and
Residential using distinct test resources. Residential integration used four
read-only current inputs and two synthetic inputs, bounded concurrency two and
SIGKILL recovery. It finished with five UNKNOWN and one UNSUPPORTED, zero eligible,
and three HTTP503 exports. Recovery took about 128.82s including the 120s lease
wait. Some TCP controls succeeded and diagnostic candidates were returned; those
were correctly not promoted to verified Ads egress or Residential. Existing
Smart integration tests are controlled fixtures, not proof of a live Ads exit.

Synthetic browser tests passed at widths390/1440, including three-output generation
parity, copy, recheck quarantine and explicit deletion. A separate read-only live
browser check at `2026-11-10T03:00:10.977Z` passed both widths, no horizontal overflow,
three links, clipboard copy and zero page errors. The existing panel self-signed
certificate was accepted only for that UI check; probe TLS verification is strict.

## Live measurement snapshot

At Unix milliseconds `1794279573424` (2026-11-10 UTC), the final R2 deployment had:

| Measurement | Value |
| --- | ---: |
| Preserved sources | 29 |
| Residential queue records | 937 |
| Records assessed at snapshot | 176 |
| Current memberships missing from residential queue | 0 |
| Executed assessment metric samples | 188 |
| Eligible residential members | 0 |
| Active worker slots | 2 |
| Rolling five-minute rate | 20.6 configurations/minute |
| Assessment p50 | 5212.612ms |
| Assessment p95 | 13346.418ms |
| Mean measured CPU per assessment | 129.168ms |
| Peak recorded assessment RSS | 66,899,968 bytes |

These are server screening measurements under current load, including incomplete
or failed assessments. They are not full-matrix proof latency, steady-state maximum
capacity, VPN throughput, or phone/carrier performance. Record and sample counts
differ because rechecks are separate measurements. Membership changes continue
after this snapshot.

Live `/residential`, state summary, Smart, Splash and Ads metadata returned200.
Measured response times were respectively about4.0/6.4/84.5/27.7/95.6ms. All three
residential exports returned503, count0, `Cache-Control: no-store`, the same
generation, and no substitute configuration. Internal health returned200. The
public nginx `/healthz` path remains404 by existing design. Successful metadata
responses do not imply an available healthy Smart Ads configuration.

## Runtime identity and preservation

Final release: `/opt/config-repair/releases/p28-d5c9dc60e5946dfd03fb-r2`.
Schema16. Target remains V2RayNG2.3.6 / Lib40 / Xray26.7.28.

| Artifact | SHA-256 |
| --- | --- |
| Panel | `d5c9dc60e5946dfd03fbf4c7e8d523c3c3d0a5c29a9b5dcbd107bec1dd96ffbf` |
| Existing probe worker, unchanged | `905696e03dbbc706261d203febad43493ae5d9f7c98b8cf1bf5bb1ce56ea4c50` |
| Residential supervisor | `9fb05a82fc8c2484a306b05f54419695b99b139dedf9425e930d5e4548ba9c01` |
| Xray | `64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d` |

Six services are active/enabled: panel, prober, Smart, Splash, Ads and Residential.
All29sources and current settings were compared before/after and preserved. SQLite
quick_check and foreign_key_check passed. A matching stopped-service DB/blob/key
backup preceded schema migration; another preceded R2. Five worker heartbeats and
running process binary hashes were checked. The 211 runtime source/asset files
match the tested manifest; binaries are tracked separately in that manifest.

The revocation ledger is outside the ordinary backup tree at
`/var/lib/shen-residential-revocations`. It must not be rolled back with the DB.
A schema15 runtime cannot open schema16. An old runtime that bypasses the ledger
is not a valid rollback. Acknowledged/downloaded client files cannot be recalled.

Public links are generated by `/api/residential` and displayed at `/residential`.
Their private capability tokens are intentionally absent from Git; the user's
delivery includes the actual URLs. Raw export is ZIP to preserve arbitrary original
multiline/URI bytes exactly, with a manifest of the common generation. This is not
a native V2RayNG JSONL batch-subscription claim.

## OpenAI review receipt

The server-side API returned actual model name `gpt-6-astra`. Only sanitized code
and architecture were submitted, excluding production keys, UUIDs, passwords,
private URLs, configuration payloads and database contents. The running detector,
classification and publisher do not call the model and continue without the API.

| Review | Result | Input / output tokens | Estimated USD |
| --- | --- | --- | ---: |
| Architecture, high reasoning | Completed | 707 / 5068 | 0.26047 |
| Large code review, high reasoning | Incomplete; output limit; no usable final answer | 28973 / 9000 | 0.73973 |
| Focused code review, medium reasoning | Completed | 12471 / 3569 | 0.30316 |
| Total | Two usable reviews | 59,788 total tokens | 1.30336 |

Cost uses the documented short-context rates checked on the review date; it is an
estimate, not an invoiced amount. Seven substantive review findings were addressed:
fact provenance, negative-result hash binding, owner-derived path coverage, manual
identity aliases, torn journal tail handling, conditional304 fencing, and independent
evidence dimensions. No AI verdict can directly delete a configuration.

## Private receipts and remaining work

Receipts remain under `.local` on the server, excluded from Git:
`p28-check-final.log`, `p28-query-refinement.log`, `p28-final-focused-race.log`,
`p28-r2-tests.log`, worker receipts, `p28-residential-browser.json`,
`p28-live-browser.json`, `p28-live-check.json`, `p28-tested-runtime.json`,
`p28-deployed.json`, `p28-r2-deployed.json`, API review receipts and
`p28-published.json`. The last receipt records the exact published commit/tree,
remote readback and installed manifest identity separately.

The next acceptance milestone requires authorized observable egress-flow access
and documented independent exact-IP specialist sources. It also requires actual
IPv6-forced, UDP/QUIC and multi-path probe drivers and controlled/live receipts for
those paths. Until those exist, the default full-matrix policy must remain closed;
do not silently narrow policy and call the result full Google Ads coverage.
Android/VPN/carrier testing and a checksum-pinned standalone installer remain
unperformed. Limited sampling cannot establish permanent residential behavior or
100% classification accuracy.
