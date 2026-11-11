# P36 video diagnostics and acquisition recovery — 2026-11-11

Worktree: `work/p36-video-errors-20261111`. P35 is an independently active deployment and must be preserved. This checkpoint does not itself assert final deployment or publication.

The complete supplied 80-second video was inspected across forty evenly spaced frames and five contact sheets; the highlighted input details were read at full size. Source inventory was present. The visible failures belong to several different stages, not one failed fetch.

Changes:

- Shared `probe.v5` / `share-compat.v1` recognizes only named VMess metadata, exact-empty known placeholders and the TLS verification alias. Conflicting verification flags retain strict verification and require fresh connection proof. Raw bytes remain immutable. Unknown active extensions stay explicit.
- Closed query annotations `x-sub` and `Telegram` remain in raw input; no annotation URL is fetched. WireGuard's documented DNS/finalmask aliases use existing core semantics; equivalent JSON masks compare semantically, conflicting masks and unverified legacy noise extensions remain blocked.
- Exact field membership replaces substring allowlists. Strict TLS alternatives retain every applied compatibility rule in provenance.
- Input diagnostics distinguish source input, server, network, environment and unsupported capability. Output exposes current-candidate-bound per-check evidence; HTTPS success cannot hide QUIC failure or imply Android execution. Browser fixture listeners are independently allocated and readiness is process-bound.
- `http.v2` adds one optional body-recovery attempt for successful chunked responses ending in unexpected EOF. A fresh HTTP/1.0 representation must complete inside the original deadline. The first spool is discarded. DNS/IP pinning, redirect policy, TLS certificate validation, cancellation and bounded headers remain enforced. No content-length truncation, invalid HTTP/1.0 transfer encoding, endless retry, HTTP error response or partial data reaches publication.

Evidence before final integrated deployment:

- Thirteen previously parser-blocked inputs build on Xray 26.10.31 and 26.11.9. Ten pass the exact candidate's real HTTPS, DNS TCP/UDP, DoH and QUIC checks on 26.10.31. One needs IPv6; two fail their actual remote connection. This is not APK/carrier or long-term availability proof.
- Seven DNS-failed profiles were independently rechecked. Six resolve their endpoint names but still fail later authentication/reset checks; one remains DNS-unresolvable. No success was fabricated from DNS recovery.
- A source's incomplete HTTP/1.1 chunk framing was reproduced with two User-Agents and identity/gzip negotiation. A fresh HTTP/1.0 response succeeds. The implemented Go driver fetched 42,965 complete bytes and extracted 148 items (85 VLESS, 28 VMess, 33 Shadowsocks, 2 JSON profiles). Counts are the measured response, not a promise that the rotating source always returns that set.
- Two actual OpenAI Responses API reviews used `gpt-6-astra`, code only, `store:false`. Six concrete findings were addressed with regression coverage. Runtime fetching and export never depend on a model or API availability.

Do not publish raw URLs, credentials, original configs, key files or private `.local` receipts. Do not reset P35, downgrade schema21, weaken target-core/HTTPS/UDP/QUIC eligibility, extend retention, resurrect explicit deletions, or rewind the external residential revocation journal. Complete the final integrated tests, matched backup, deployment/readback and branch CAS publication before marking this phase complete.
