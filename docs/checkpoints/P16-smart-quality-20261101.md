# P16 — Smart admission based on fresh connection evidence

Date:2026-11-01. Parent15f74516d6980b855ae33e701368463bf2effccc. Branch go/v2rayng-2.3.6. Smart v3; schema9 and normal template-v4 unchanged.

## Evidence and scope

The screen recording shows successful core startup and stalls while loading applications. It is not a packet capture or Android speed benchmark. The active member count was50/revision4, not10.

A read-only live snapshot before the fix found907source-ranked eligible rows:753fresh passed,72failed,73partial and9not_run. A separate ready-cache snapshot had650member selections, of which102lacked fresh connection proof. These are changing snapshots, with repeated selections across profiles, not unique failure totals.

The concrete bug was the smart_inputs condition admitting ordinary core-valid candidates regardless of connection outcome. Source fairness could therefore force known-failing or untested members into a profile. P16 removes this path: every selected input requires fresh exact-owner connection=passed; smart_inputs expiry is min(artifact expiry, proof deadline).

Every selected input's complete outbound definitions and the ordered P15 envelope stay unchanged. Existing normal output rules, sources, count/settings and independent rounds remain. A composite input pass concerns its tested route and does not prove every inner server. A server-side pass is not a phone-network speed guarantee.

## Verification

- Full modules/vet/race/build plus11Python sandbox checks passed.
- New regressions cover failed/partial/not_run/expired ordinary core-valid inputs; immediate old-cache exclusion; recovery; normal export preservation; same-identity cross-source proof isolation; proof-bounded cache; stale worker completion rejection.
- Pinned core1/2/10/50, exact envelope and logs-off TCP/UDP/UDP443 fixture remain passing.
- Browser mobile/desktop/restart/full/single passed. Fixture explicitly uses synthetic successful connection evidence; that harness is never shipped.
- Isolated real core worker:59profiles,50inputs/profile,four sources/four current accepted canaries,2slot maximum,SIGKILL recovery and unique head round passed. Fixture member proof is synthetic; this is core/recovery evidence.
- Production:8ready profiles and400member selections with0unverified; current cache recomposed from encrypted member data:2profiles,100input selections,400copied outbounds. No raw values logged/committed.
- Android throughput,p95 response time,UDP loss and live QUIC handshake are not_measured/not_run. No speed percentage improvement is claimed.

## Performance strategy

Read docs/SMART_PERFORMANCE.md for the Persian evidence/plan. Keep the exact requested configuration as baseline. Compare a smaller qualified pool before changing expected/burst; compare one variable at a time on the same client network. HTTPing failure rate is not UDP packet loss; leastLoad does not aggregate one connection's bandwidth. Client-specific stability and session continuity need actual client observation and, for hysteresis, explicit client control rather than invented JSON fields.

## Deployment

Release:/opt/config-repair/releases/p16-0da8ca3cf86edbb5ec23
Panel SHA256:0da8ca3cf86edbb5ec2374ff340dbb94230bd53a90bf2615a39854cbf1b83dcb
Worker SHA256:ee00499a1009142447a989e69393f9fb3f0509beb3f73b5f2d5df74d8c8d6839
Previous:/opt/config-repair/releases/p15-b003f05d4d978a956f0e
Matching stopped backup:/var/backups/config-repair/pre-smart-quality-20261101-065158

No schema migration. All10live source definitions, current settings/count50 and normal/Smart queue fingerprints preserved across stopped-service deployment. All three services active/enabled; runtime hashes checked independently. Smart cache rebuilds under v3. Matching prior binaries can roll back same-schema state but would restore weaker admission. Do not restore historical source data or reset user turns.

Private receipts:p16-deployed,quality,audit,public,published JSON and p16-check/browser/worker logs. Publish/read back exact tested changes separately from runtime verification.

Public browser readback passed at both viewport sizes with exact P15 globals, all copied outbounds, no downloads or JS errors. Single-output status is recorded per viewport;204 during a preserved round waiting on uncached heads is the documented no-repeat response, not a JSON error. Full/single completed rounds also passed the isolated browser/worker tests.
