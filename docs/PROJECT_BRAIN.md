# Project Brain — P8

Current user instruction (2026-10-30) replaces the persistent repair laboratory with a shared JSON conversion flow. Read PROJECT_STATE.json, HANDOFF.md, docs/CONVERSION.md and the P7 checkpoint. Earlier checkpoints describe history only.

Go fetches source links independently, extracts exact encrypted originals, maps supported protocols to the user's @shenvpn full JSON envelope, validates using pinned Xray26.7.28 and tracks connection separately. Output is a clean list plus the all-config JSONL and single-config rotation links. The bank, manual import, historical variants, learned model/rule data and old subscriptions are removed by schema6.

The actual endpoint/auth/transport must be retained, with explicit errors for unrecoverable/unsupported mappings. Common DNS/inbounds/routing/log/policy intentionally replace source globals per the new user instruction. Domain endpoints receive scoped local DoH bootstrap to avoid DNS recursion. TLS hypotheses require fresh connection proof. Core-valid ordinary mappings may be exported with unsuccessful/unknown connection shown accurately. Never claim Android validation; JSONL is not the pinned app's native JSON-array subscription format.

Source ownership, fixed default2h retention, global identity dedup, exact raw bytes, independent revisions/leases/deadlines, encrypted storage, passwordless HTTPS/CSRF, bounded parallel work and systemd recovery remain. Conversion data cannot outlive active memberships. Untrusted execution stays in the proven sandbox; private worker API is Unix-only.

Main boundaries: internal/collect drivers; internal/repair deterministic structural candidates; internal/probe protocol preparation; internal/convert common envelope; internal/panel/conversion*.go transient queue/outputs; cmd/config-probe-worker + scripts/probe-one.py/live-probe.py sandbox validator. No dynamic upload/execution of arbitrary plugins.

P8 adds a separate single-config rotation link. Schema7 stores only the current cycle and pending candidate hashes, with atomic SQLite claims and blob validation outside transactions. The global round is persistent across restart; new configs wait for the next snapshot, unavailable ones are skipped. See docs/CONVERSION.md and docs/checkpoints/P8-single-output-20261030.md. All-config export remains unchanged.
