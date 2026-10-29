# Persistent sample bank and isolated worker

The P6a user instruction preserves test samples across source changes and expiry. Source membership and original response retention remain transient; accepted samples are copied by reference into an independent encrypted archive. SHA-256 distinguishes exact original representations, while canonical identities deduplicate subscription entries. Structural models exclude credential and endpoint values.

## Lifecycle and evidence

An accepted collector generation or manual import queues each new sample. Original bytes are immutable. Preparation produces separate JSON candidates for the pinned target. Model-specific successful rules may be tried first; every candidate must still receive its own evidence. Attempts have distinct preparation, core, connection and Android states. Hysteria 1 is recognized as unsupported by this core, not relabelled as Hysteria 2.

Schema 5 adds samples, revision-checked leased jobs, latest results, candidate history, structural models, rule outcome counts, probe settings and worker heartbeat. Renewals and completions require a current token/revision. Explicit deletion creates a tombstone and removes variant references. Disable/retest invalidates current subscription evidence. Global pause revokes running leases. Expired leases recover after a supervisor restart. Every fourth available claim prioritizes due retests, preventing a continuous stream of new samples from starving health refreshes. Collection and probing have separate concurrency/deadline controls.

All candidate bytes selected by a completed trial remain encrypted in candidate history, even when a later trial chooses another variant. Historical download is not a fresh health certificate. Garbage collection retains samples and history until explicit bank deletion; source deletion and transient retention do not remove them. The archive grows with unique representations; existing collector free-space protection still applies.

## Worker boundary

`config-panel` owns SQLite and the encryption key. It creates a 0600 Unix socket in its private database directory. `/claim`, `/renew`, `/complete`, `/strategy` and `/status` are mounted only there, never in public HTTP. The root `config-probe-worker` supervises a bounded pool, calls `probe-one.py` over stdin, and sends only candidates plus structured hash/code evidence back through the socket. It does not receive the master key or accept shell arguments from configuration content.

The core checksum is pinned to `64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d`. Each trial gets its own network namespace, public endpoint/port allowlist, private-address denial, cgroup limits, unprivileged UID and read-only filesystem. HTTPS trust uses the host CA bundle. The runtime adds a temporary SOCKS inbound, blocks freedom/direct routing, and pins resolved endpoint addresses. A core syntax pass alone cannot approve a subscription entry. Port hopping is validated before constructing firewall expressions; same-host multiple ports are retained.

Resources use the strictly owned `crw` prefix and a run ID plus worker slot. Cleanup after each trial is scoped to its run, so an old trial cannot delete a replacement trial's resources. Startup cleans abandoned owned resources. The Python supervisor monitors its Go parent and exits on parent death; this avoids Linux parent-thread death semantics when launching from a multithreaded Go process. `systemd` restarts the supervisor and enables it at boot. A server that is powered off cannot run jobs; work resumes when it is powered on and networking is available.

## Subscription contract

Output exposes two tokenized HTTPS URLs: `all.json` and `repaired.json`. Both are homogeneous JSON arrays because the pinned Android subscription parser handles URI imports before custom JSON. Each candidate inside the envelope is written with its exact tested bytes. The repaired-only feed excludes ordinary URI-to-JSON conversion and unchanged original JSON.

At request time, export requires an enabled, nondeleted sample; current planner version; a successful pinned-core check; a successful HTTPS request to **both** test destinations; and a nonexpired evidence timestamp. Partial, failed, disabled, expired and older-rule proofs are excluded. Canonical sample identities and exact candidate hashes prevent duplicate exports. A failed retest removes the previous entry from the feed immediately; the sample and historical candidate remain available for diagnosis.

Headers and UI identify the scope as Linux core plus two HTTPS destinations. Android import, VPN routing, UDP payload behavior and every possible custom routing branch are **not** certified by this test. TLS verification is not weakened, credentials are not guessed, and dead endpoints cannot be repaired by JSON wrapping.

## Operation and validation

The panel retains its existing unprivileged `config-repair.service`. The separate root unit is `deploy/config-repair-prober.service`. Install the tested worker binary and trusted scripts under the root-owned current release; install the pinned core and any required geo assets under `/opt/config-repair/core-26.7.28`. Dependencies are Linux cgroup v2, network namespaces, nftables, iproute2, bubblewrap, curl, Python 3, system CA certificates and IPv4 forwarding. Preserve matching database, encrypted blobs, key and prior binary before schema migration.

```sh
bash scripts/check.sh
PLAYWRIGHT_MODULE=/path/to/playwright-core CHROME_PATH=/path/to/chrome python3 scripts/panel-browser-test.py
python3 scripts/probe-worker-test.py --live-canary 12
```

The last command needs root, uses a separate private database/listener, exercises supervisor SIGKILL and lease recovery, and optionally reads current authorized production samples without changing them. **Run it only while the permanent prober is stopped**: its final cleanup intentionally removes all owned abandoned `crw` resources. It never certifies Android. Its protected receipts contain counts, hashes and diagnosis codes, not configuration credentials. The reproducible complete installer/update/rollback distribution remains a separate release milestone.
