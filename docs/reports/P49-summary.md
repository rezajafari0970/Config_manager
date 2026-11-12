# P49 experimental carrier-aware observation: rollout withheld

Production remains P47 (p47-b51b595c6ec184771530, schema26).
The candidate is isolated in fix/p49-carrier-payload-health. No production
migration, service replacement, global blacklist or deletion was performed.
Read ../SMART_CARRIER_HEALTH.md for the implementation and its limits.

## Outcome
The measured original VLESS TCP HTTP-header/no-TLS path is not repaired.
Current native Android transfers, on the route reported by the user as MCI,
stall despite successful short HTTPS/DNS. Irancell success is user-reported;
an independent paired Irancell test is still not_run.

Smartv15/Adsv8 replaces 204-only observation with a 64KiB HTTPS GET target.
Main leastLoad3/2s/.15 remains; observed recovery1/10s/.15 fails closed.
The shared module applies to every Smart composer; singleton Ads now passes
through its own observed gate. Migration27 is allowlisted, fixed-version and
fences only old Smart receipts/leases, leaving ordinaryv9/Ads classifier v5,
raw inputs and user settings intact. Defensive checks prevent duplicate
recovery tags, malformed selectors and silent closed-envelope field loss.

## Phone evidence
Two-member original+working-WS-TLS pool:
* 26.11.9: old4/6 complete64KiB, candidate6/6;2MiB down1840ms,1MiB up2187ms.
* 26.10.31: old2/6, candidate6/6;down1851ms,up1843ms.
* Fresh first verified HTTPS2825/5481ms. No universal two-second guarantee.
* DNS TCP/UDP/DoH passed. Initial latest QUIC VN failed; repeats1/3. Bundled
  initial passed and repeats3/3. Phone fullHTTP3 and APK/VpnService remain not_run.
* Google/Telegram/Instagram public HTTPS transferred, with bounded redirect
  follow for Instagram. This is not native media/feed/messaging proof.

A frozen production pool with50 real members was then tested:
* First candidate run:64KiB6/6 and2MiB down on each core. FirstHTTPS1345/1467ms.
  Latest1MiB upload reset; bundled upload passed.
* Three latest-core paired rounds: old2MiB download1/3 vs candidate3/3.
  Old1MiB upload2/3 vs candidate2/3. The improvement is download-specific;
  upload stability is not solved. Candidate download ranged1509–8272ms.
* Last candidate round's harness parent disappeared before writing a receipt.
  Its incomplete files were archived; only that unfinished round was rerun.
  It is not represented as an uninterrupted controlled trial.
* Forcing individual routes implicated proxy-46 in the frozen pool:2MiB down
  passed,Cloudflare1MiB upload reset;proxy-27/44/36 passed that upload.
  These tags identify this exact frozen profile, not permanent server identities.
* Follow-up:proxy-46 also failed1MiB upload to httpbin;proxy-27 received an exact1MiB echo with verified HTTPS. Requesting HTTP2 did not repair46; the successful27 control negotiatedHTTP1.1, so this is not HTTP2 proof.
* An alternate httpbingo1MiB test returned400. Direct control confirmed its
  request-body-size limit; that is a test-endpoint rejection, not proxy failure.

A different screened REALITY member passed64KiB but failed2MiB/upload. Thus a
download observer cannot establish per-device upload/application eligibility.
Stock core JSON has no verified upload probe or persistent per-carrier quality
state. Do not invent unsupported fields, globally blacklist a family, or call
this a complete health strategy. A stronger device-scoped bidirectional check
and tested client refresh/control integration are needed for that requirement.

## Validation
Full release suite with Go1.26.0 passed: vet, race, six builds, Python and JS.
The first full run had one stale schema26 test expectation; it was corrected
to27, the focused race case passed, and the complete final suite passed.
Host Go1.27.1 race linking failed independently; no global toolchain was changed.

Both pinned official cores passed generated5/7/10/15/50/100 structural validation,
partial-body exclusion, slow-path recovery and all-down origin non-delivery.
Known observer short503/200/302 false acceptance is explicitly reproduced.
Both cores passed Ads chain and cold-cache tunneled DoH with production payload
observation timing. Synthetic fixtures use loopback HTTP for the body target;
real phone/Linux lanes verify HTTPS certificates.

Normal, Splash and Ads queue tests include SIGKILL recovery, ownership, rotation
and immutable exports. Their connection receipts are explicitly synthetic,
not phone or Internet health proof. Final private migration rehearsal preserved
33sources/897members and settings; it did not touch production.
The final compiler emits the same tested two-member bytes:
00e80e93f220decdaf8cf8a98ad7c156aa434754df2a5d2e02634552399c4ffe.

Sanitized OpenAI API review completed with gpt-6-astra/high. The first max/pro
attempt was incomplete. API advice was used for hardening and negative tests,
never as a traffic pass. No keys or private configuration credentials were sent.

## Release decision
Do not activate this candidate as a universal repair. It improves the observed
download-stall class but the50-member upload gate failed and QUIC remains
intermittent. Keep the tested two-member JSON labelled experimental, not a
guarantee for every app, operator or future server state. Existing imports are
not automatically updated by server source changes.
