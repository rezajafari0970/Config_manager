# P39 — complete Ads egress replacement

User explicitly authorized deletion of the old Residential subsystem and a new
implementation. Started from published e6d5a6e, preserving the newer canonical
c88a43f acquisition fixes when concurrent P36 publication was discovered.

Removed legacy internal/residential, panel handlers/schema/gates/attestation,
worker, scripts, service and dedicated documentation. New internal/egress owns
measurement policy and protocol plans; egress_* owns transactional state and
exports. Version23 replaces all old feature tables with a fresh current-input
queue. Ordinary input collection and templates retain their current versions.

Private real-database rehearsal preserved33 sources and819 memberships, seeded
715 unique raw inputs, found zero old table/view/trigger names and schema23.
Current original Stockholm canary: all three pinned Xray versions built raw and
template;90 total measured requests across three attempts timed out, zero egress IPs.
No MO or residential conclusion was carried forward.

New observation driver successfully obtained same-TLS Ads error-page IPs from
other live acquired configurations. Exact-IP Proxycheck returned Hosting for
the observed examples (including IPv6). This is evidence the plumbing functions,
not proof of any Residential member. Paired IPv4 and IPv6 control groups are
independent; mismatched families never establish a split route.

Initial browser QA caught long retention text overflowing mobile width; fixed
the label and scoped wrapping. Real browser checks cover390 and1440 widths,
four output links, diagnostics and expanded evidence without script errors.
Worker qualification includes protected Unix claims, two-slot limit, killed
supervisor recovery and five isolated inputs (three live, two synthetic).
Receipts will be finalized after full release qualification and activation.

Evidence limitations, lifecycle, classifier budget, architecture and command
contracts: docs/EGRESS.md. Never claim100% detection or apply observed TCP scope
to every real ad request. Runtime API use of an LLM is unnecessary and absent.

Final fresh worker fixture: five of five inputs completed, three live Hosting
results and two synthetic UNKNOWN results. All three live results obtained
42 samples including paired IPv6 controls and reported overlapping Ads/control
egress. Forced SIGKILL recovered both leased jobs after expiry; attempts>=2
checked, maximum two active,136.19 seconds after browser validation. Browser
mobile390/desktop1440 passed. Targeted race checks also caught and fixed ordinary
source restore regression: permanent exclusions apply only to accepted/revoked
feature members, preserving ordinary source-scoped undo semantics.

Activation complete: p39-ebd1c323ba6e0e2d4879, schema23, six active services,
matched running hashes. Cutover snapshot33 sources/825 memberships/724 raw inputs.
Legacy unit and guard removed, old tables/references zero, all current raw hashes
registered. Initial live readback12 assessed of776 accumulated entries, zero
eligible; background assessment continues. Empty subscription pool returns503.
Full final scripts/check.sh passed, panel race419.188 seconds, no source changes
after final-suite code manifest. A matching stopped backup is outside active code.
