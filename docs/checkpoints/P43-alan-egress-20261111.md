# P43 — Alan source egress diagnosis

User requests testing Alan splash first, comparing Google Ads and controls, and
fixing the shared project behavior. The original raw was rejected before network
measurement because DNS Direct routes reached the shared health-only preflight.
The selected1.1.1.1 control also bypassed that raw proxy. Distinct Ads egress was
reproduced with exact raw/template and two independent controls; classification
found a mixed pool, never universal Residential proof.

Dedicated finite-scope preflight, corrected family-specific controls, low-burst
XHTTP/deep sessions, actionable failures, visible partial/type evidence and fair
current/deep/manual scheduling are now implemented. Failed measurements are not
converted into Residential or credible deletion votes. The classifier supports
an optional protected registered key and honors the documented unregistered100
query/day budget. Only10 remaining queries were used; no quota bypass/reset.

Full release/race suite,21 Python egress contracts, private schema25 rehearsal,
SIGKILL recovery and mobile/desktop fixture checks passed. A fragile wall-clock
concurrency assertion failed under shared host load; it was replaced by a start
barrier that directly verifies parallel scheduling, then the full check passed.
The actual network measurements and their limits are separate from unit fixtures.

Release p43-80df8988c7565dcc30c7 is active after a matching stopped recovery set.
All33 source controls/settings are unchanged and six executable hashes match.
Private receipts remain in .local; sanitized results are in P43-report-fa.md and
P43-evidence.json. The final live/export/publication receipt is P43-validation.json.
