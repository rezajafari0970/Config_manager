# Panel — P3

The UI has exactly three feature modules: Input, Output, Setting. Desktop sidebar and mobile drawer share navigation; source rows and bulk import expand in place. Persian digits are accepted. Each page has independent HTML/JS and backend route modules.

## Input

One URL per line in bulk import, with one shared interval expression (`2 to 3`, `۲ تا ۳`, `2-3`, or a fixed positive value), in seconds or minutes. Each accepted row is scheduled independently. Names use the parent directory (`/Api/Verde/trun.php` → `Verde`), falling back to hostname. Exact duplicate source URLs are rejected. Errors report line numbers; valid siblings are still inserted. Search/page size 100 affects display only.

Individual actions: edit URL/name/interval, run now, pause/resume, delete. Optimistic revisions reject stale edits. Editing a URL invalidates its old membership/validators. Deleting or pausing cancels that source's running job. Status reports request outcome, HTTP code, duration, counts, last success and next request; it never reports proxy connectivity.

## Output and retention

Original HTTP entity body (after standard HTTP content decoding) and each extracted config are encrypted at rest and returned byte-for-byte. URI whitespace/CRLF, JSON formatting/unknown fields/numeric literals and INI content are preserved. Base64/JSON string envelopes are decoded for extraction; the entire original response remains separately downloadable. ZIP stores each config as its own file, so mixed JSON and URI output is not misrepresented as a universally importable subscription.

Identity is separate from raw content. Known equivalent URI encodings, display fragments, VMess remarks and canonical JSON are used only for deduplication. Credentials and transport values remain identity-bearing. Membership preserves each source's own raw representation; output chooses one current source deterministically. Deleting one source cannot rewrite another's original. Unknown formats and configs larger than 1 MiB use conservative exact-content identity; arbitrary semantic equivalence is not claimed.

Retention defaults to 7200 seconds, configured as hours. The clock starts at first insertion; repeated content or HTTP 304 never slides expiry. Reads exclude expired items immediately; background cleanup removes rows/files every 5 seconds. Empty/error responses retain the previous generation only until this deadline. A later fresh 200 response can collect a deleted config again. Changing retention recalculates existing unexpired deadlines; extending does not resurrect already expired data. Deleting sources or a nonempty replacement can remove membership earlier.

No artificial source/config count, URL/name length, response-byte or 30-day interval quota remains. Positive representable intervals, valid HTTP(S), finite timeouts, bounded parallelism, private-network egress protection and physical disk/RAM capacity still apply. A 256 MiB free-disk guard pauses admission and resumes automatically. Completely unlimited resources or uninterrupted operation under every host failure cannot be guaranteed.

## Deployment

Runtime config, outside Git:

```json
{"listen":"127.0.0.1:18191","public_origin":"https://YOUR_HOST:18443","database":"/var/lib/config-repair/panel.db","master_key":"BASE64_RANDOM_32_BYTES"}
```

Keep the original 32-byte master key across upgrades. Config mode 0640 root:config-repair, database directory 0700, database/blobs 0600. Start `config-panel -config /etc/config-repair/panel.json`. There is no admin initialization or password flag. Schema 1 upgrades transactionally to schema 2 and removes old auth/session tables while preserving source URLs and schedules. Back up with SQLite's backup API plus protected runtime config before migration; rollback to P2 requires the old database backup as well as old binary.

Public management is deliberately passwordless at the user's request. HTTPS, same-origin CSRF token, strict cookie, restrictive CSP and safe DOM rendering remain. URLs/blobs are AES-GCM encrypted at rest; ordinary lists show only hostname. Full URL is available to anyone using edit, consistent with passwordless administration.

`deploy/` supplies the separate systemd unit and nginx example. systemd restarts after exits, receives watchdog heartbeats only while DB/scheduler are healthy, and kills/restarts a stalled process. SQLite leases recover after restart; a file lock prevents two owners of the same DB. Request clients, DNS-checked dial destinations, redirects, timeouts and cancellations are per job. Go plugins are compiled interfaces, not dynamically uploaded executable code. Run nginx validation before reloading. No unrelated service or credentials may be modified.

## Verification

`bash scripts/check.sh`: module verification, formatting, vet, race tests and binaries.

```sh
PLAYWRIGHT_MODULE=/path/to/playwright-core CHROME_PATH=/path/to/chrome python3 scripts/panel-browser-test.py
```

The browser test serves only synthetic fixtures on 127.0.0.1:18192. It tests bulk/CRUD/status, independent failures, exact bytes/SHA/ZIP, deduplication, expiry, persistence after full process restart, and mobile drawer/layout. Artifacts remain in ignored `.local/p3-browser/`. It does not access real subscriptions, execute untrusted configs, test Android or establish a VPN tunnel.


## P3b — bulk controls, diagnostics and real deadlines

Input's interval field uses a text keyboard so Android users can enter `۲ تا ۳`; it also accepts `2-3` and `2 to 3`. Timeout is expressed in whole seconds: Setting supplies the default, bulk import can assign one shared override, and edit can change each source separately. Empty/zero override inherits the shared setting. The HTTP driver uses the job context for dialing, TLS, headers and body; the former hidden 10/15-second transport deadlines are removed. Runtime effective parallel capacity is visible in Setting.

Sources expose enabled state separately from current request outcome and content classification: readable configuration, mixed/partly malformed content, malformed known config, empty, unrecognized body, server error response, or inspection not available. Counts distinguish readable/invalid/unknown candidates and received bytes. Original responses remain archived even for non-2xx errors. Malformed JSON siblings no longer discard unrelated recognized spans before or after the bad item. JSON string envelopes can carry exact Custom JSON payloads. InspectionDriver is a compiled extension interface and a panicking inspector only marks its item unknown. These labels never assert a working VPN or Android import.

Checkboxes select the visible page and preserve selection while navigating pages. Source enable/disable/delete uses one transaction with revision checks; stale selections fail atomically. Active jobs are cancelled only for affected source IDs. A corrupt source row is reported/retried independently instead of blocking the due-source batch.

Config disable preserves bytes and excludes the item from ZIP and ordinary download; the administrative raw preview remains available. Repeated collection and restart do not silently enable it. Enable reverses disable. Config delete leaves a metadata-only tombstone until that config's original expiry; in-flight or repeated content cannot immediately resurrect it. Controls expire and retention changes apply to their original creation time. This is not a permanent blacklist. Source-response archives may still contain a selectively deleted config; clear-all-configs removes those archives too. Clear-all invalidates active source revisions and leases, clears staging/responses/artifacts, and keeps URLs/settings. Future new configurations can still arrive. Clear-all-sources cancels/removes sources and their memberships. The UI requests explicit confirmation for destructive actions; cross-origin POST is rejected.

Schema 3 migrates source diagnostics/timeout fields, artifact quality and expiring config controls without rewriting source URLs, source settings, config bytes or retention settings. The service remains enabled at boot; watchdog recovery is tested on a separate transient service, not by rebooting the shared host. During physical power-off no fetch can run; the collector resumes after the host starts.


## تعمیر ساختاری جداگانه

در هر دریافت، قواعد نسخهٔ هدف روی نمونهٔ قابل بررسی اجرا می‌شوند. در Input تعداد اصلاح‌شده و نیازمند بررسی نمایش داده می‌شود. در Output دکمهٔ «جزئیات بررسی» و برای موارد ترمیم‌شده «دانلود نسخهٔ اصلاح‌شده» موجود است. «دانلود» و «دانلود همه» همچنان اصل دریافتی را می‌دهند. نامعتبر بودن اصل با موفقیت اصلاح یک نسخهٔ جداگانه تناقض ندارد. هیچ وضعیت تعمیر به معنی اتصال موفق نیست. نسخهٔ اصلاح‌شده با غیرفعال‌سازی/حذف/انقضای اصل از خروجی خارج می‌شود.

API: `GET /api/repairs/{id}` گزارش بدون رمزها، `GET /output/repaired/{id}` فایل جداگانه با هدر `X-Original-SHA256` و `X-Validation-Stage: structural-only`. مهاجرت به schema 4، دریافت بعدی را بدون ETag درخواست می‌کند؛ منابع و تنظیمات حفظ می‌شوند.
