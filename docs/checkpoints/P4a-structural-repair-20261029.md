# P4a — automatic structural candidates

The user now authorizes repairing malformed configurations as far as recoverable. Immutable original acquisition remains; automatic repair produces a separate auditable candidate on each fetch. Raw downloads and raw ZIP stay exact. No credential guessing, protocol substitution, TLS weakening, unknown-field removal or arbitrary-config execution occurs.

## Implementation

- Offline CandidateDriver registry with panic containment and fixed target/rules version. The bounded inspection sample limits repair work, not storage or source count.
- Strict duplicate-key/depth/size checks; surgical empty-array→object repairs at verified schema paths (stats/log/dns/policy/routing, dns.hosts, policy.levels/system, freedom outbound settings). Nonempty arrays are blocked. Recover trailing commas outside strings, and missing empty inbounds/routing import containers when outbounds exists. Valid unaffected bytes and large numeric literals remain unchanged.
- Mixed URI/JSON/WireGuard extraction preserves contiguous INI spans. Recognizable malformed JSON is retained as raw input for diagnosis. Orphan WireGuard sections receive a specific source error instead of being treated as JSON. No recombination of shuffled keys.
- Schema 4 stores report metadata and separately encrypted candidate references keyed by exact raw SHA. Existing source/settings/control data stays intact. Candidates inherit original availability/TTL, never become extra config identities, and become unavailable on disable/delete/expiry. Garbage collection removes orphan derivatives.
- Input shows per-source repaired/unresolved counts; Output shows per-item repair status, diagnostic paths and a separate repaired download. Raw ZIP is unchanged. Android/core/connection evidence stays not_run for fetched configs.
- Invalid Shadowsocks credentials/scheme ambiguity and WireGuard extension requirements are explicit unresolved diagnoses.

## Verification

Full format/module/vet/race/build gate, synthetic browser workflows and repaired download/report/restart/TTL checks pass. A regression test covers automatic mixed JSON collection; browser acceptance exposed a nil-error dereference added during diagnostics refinement, which was fixed before deployment. Structural fuzz testing ran 195,337 cases without failure. 24 controlled pinned-core cases pass, including 11 new structural rules/envelope/syntax cases. No live fetched config was executed by Xray.

Isolated actual-source sample: 54 sources, 915 raw outputs, 969 original/config downloads verified by SHA, zero identical raw duplicate identities. Six Penguin JSONs received verified candidate downloads (stats, freedom settings and dns.hosts). Two unresolved inputs were an ambiguous Shadowsocks URI and a scattered WireGuard header; one complete WireGuard carried unsupported/unverified extensions. Counts vary by fetch. Production data was untouched in isolated/destructive acceptance.

Actual final binary, migration, public browser, production sweep, exact Git publication and CI are recorded separately in ignored `.local/p4-*.json` receipts. No shared host reboot, no DigitalOcean project/service/proxy change, and no Android or real connection acceptance.
