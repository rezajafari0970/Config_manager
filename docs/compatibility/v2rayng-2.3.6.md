# بررسی مبنای سازگاری V2RayNG 2.3.6

این سند نتیجهٔ بررسی اولیهٔ مسیرهای مرتبط است؛ **ممیزی کامل همهٔ قابلیت‌ها یا تست واقعی Android نیست**.

مرجع Android: commit `13138dddcab291ba3618cc54721ce4c7dd926be0` از tag `2.3.6`.
مرجع هسته: commit `5ca6f4b7d4dc20a881d4330e498892697627ec0c` از tag `v26.7.28`.
Lib v40 از تصویر کاربر ثبت شده است؛ hash APK و کتابخانهٔ native برای مرحلهٔ Android هنوز باید ثبت شود.

## یافته‌ها و اثر بر طراحی

1. **Custom JSON شرط واردکردن جداگانه دارد.** در `AngConfigManager.kt:324` ابتدا وجود رشته‌های `inbounds`، `outbounds` و `routing` بررسی می‌شود؛ سپس آرایهٔ JSON و بعد سند تکی امتحان می‌شود. پس پذیرفته‌شدن یک JSON توسط Xray به معنی واردشدن آن از مسیر اشتراک برنامه نیست. در محصول باید کلیدها واقعاً با نوع صحیح موجود باشند، نه صرفاً به‌صورت متن داخل یک رشته.

2. **اشتراک مخلوط می‌تواند Custom را نادیده بگیرد.** در `importBatchConfig` و `parseConfigViaSub`، ابتدا لینک‌ها خوانده می‌شوند؛ Custom فقط وقتی تعداد لینک‌های پذیرفته‌شده صفر باشد امتحان می‌شود. خروجی URI و JSON را نباید بی‌قاعده کنار هم چاپ کرد. JSON آرایهٔ مستقل یا قالب مشترک اثبات‌شده در خود برنامه لازم است.

3. **JSON سالم باید حفظ شود.** `CustomFmt.kt` با DTO اطلاعات نمایشی را استخراج می‌کند؛ `CoreConfigManager.kt:89` دادهٔ خام ذخیره‌شده را برای اجرای Custom می‌خواند. تعمیر باید روی دادهٔ کامل باشد؛ تبدیل آن به یک outbound تنها، routing و زنجیره‌ها را از بین می‌برد.

4. **تنظیمات برنامه بر Custom اثر دارند.** در `buildV2rayCustomConfig` فعال/غیرفعال‌بودن آمار باعث افزودن/حذف stats و بخش policy.system می‌شود. حالت TUN می‌تواند inbound اضافه کند؛ قواعد process ممکن است به UID تبدیل شوند. تست Android باید تنظیمات runtime را ثبت کند.

5. **stats یک object در Xray است.** `infra/conf/xray.go:376` آن را struct و در Config یک pointer به آن تعریف می‌کند. `[]` خطای ساختار است. نخستین قاعده فقط آرایهٔ خالی top-level را به `{}` تبدیل می‌کند؛ آرایهٔ دارای داده، کلید تکراری یا JSON ناقص اصلاح حدسی نمی‌شود.

6. **قالب flat جدید و قالب‌های قدیمی را تفکیک می‌کنیم.** `CoreOutboundBuilder` برای VMess/VLESS/Trojan مقدارهای address/port/id/password را مستقیماً در settings می‌سازد. هستهٔ دقیق در `infra/conf/vmess.go` و `vless.go` مسیر `vnext` و در `trojan.go` مسیر `servers` را نیز دارد. بنابراین nested بودن به‌تنهایی دلیل خرابی یا الزام تبدیل نیست. متادیتای نمایش برنامه برای JSON قدیمی هنوز به تست Android نیاز دارد.

7. **Hysteria در این هسته نسخهٔ ۲ است.** `infra/conf/hysteria.go:13` فقط version=2 را می‌پذیرد. برنامه در `toOutboundHysteria2` از protocol=hysteria و settings.version=2 استفاده می‌کند و network=hysteria و ALPN=h3 می‌سازد. auth در hysteriaSettings حمل می‌شود. تبدیل Hysteria 1 به JSON این محدودیت را رفع نمی‌کند.

8. **Query parser محدودیت‌های نسخه‌ای دارد.** `FmtBase.getQueryParam` با split روی `&` و `=` کار می‌کند. مقادیر دارای `=` خام، پارامتر بی‌مقدار و duplicateها نیازمند نمونهٔ دقیق و تست رفتاری هستند؛ percent-encoding باید بدون تغییر رمز و معنا مدیریت شود. هر پارامتر ناشناخته را نمی‌توان دور انداخت.

9. **WireGuard conf نیازمند بررسی جداست.** در `WireguardFmt.parseWireguardConfFile` endpoint با اولین `:` جدا می‌شود و همهٔ Peerها در یک map نوشته می‌شوند. فایل IPv6 یا چند Peer نباید بی‌دلیل به یک لینک ساده تقلیل یابد. JSON هم‌ارز ممکن است مسیر حفظ معنا باشد، مشروط به تست کامل.

10. **Shadowsocks pluginها عمومی پشتیبانی نمی‌شوند.** کد این نسخه برای حالت شناسایی‌شدهٔ `obfs=http` مسیر مشخصی دارد؛ هر plugin خارجی را نمی‌توان صرفاً با parse موفق سازگار اعلام کرد.

11. **پورت Hysteria2 به رفتار واردکننده وابسته است.** `Hysteria2Fmt.parse` از uri.port و `mport` استفاده می‌کند؛ تفاوت با URIهای دارای پورت پیش‌فرض یا فهرست پورت باید در جدول تبدیل ثبت و تست شود.

12. **نبود address در Hysteria خطر crash هسته دارد.** Build در نسخهٔ هدف بدون بررسی nil از Address استفاده می‌کند. بررسی اولیه این مورد را قبل از اجرای هسته گزارش می‌کند؛ اجرای آینده همچنان باید در پردازش محدود و مجزا باشد.

## منابع دقیق

- [AngConfigManager](https://github.com/2dust/v2rayNG/blob/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/handler/AngConfigManager.kt)
- [CustomFmt](https://github.com/2dust/v2rayNG/blob/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/fmt/CustomFmt.kt)
- [CoreConfigManager](https://github.com/2dust/v2rayNG/blob/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/core/CoreConfigManager.kt)
- [CoreOutboundBuilder](https://github.com/2dust/v2rayNG/blob/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/core/CoreOutboundBuilder.kt)
- [V2rayConfig DTO](https://github.com/2dust/v2rayNG/blob/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/dto/V2rayConfig.kt)
- [Formatter directory](https://github.com/2dust/v2rayNG/tree/13138dddcab291ba3618cc54721ce4c7dd926be0/V2rayNG/app/src/main/java/com/v2ray/ang/fmt)
- [Pinned Xray config builders](https://github.com/XTLS/Xray-core/tree/5ca6f4b7d4dc20a881d4330e498892697627ec0c/infra/conf)

## موارد باز

جدول کامل transport/TLS/Reality/XHTTP/FinalMask، پیش‌فرض‌های parser، رفتار منابع Base64 و v2rayn، سایر قواعد ترمیم، نمونه‌های اتصال واقعی، hash دقیق APK/native library و تست واقعی واردکردن/اجرا روی Android هنوز تکمیل نشده‌اند. این موارد در P1/P4/P5/P6 دنبال می‌شوند.

## قواعد ساختاری P4a — ۲۹ اکتبر ۲۰۲۶

مراجع همچنان همان commitهای قفل‌شدهٔ بالا هستند. `infra/conf/xray.go` نوع object بخش‌های stats/log/dns/policy/routing و بارگذاری settings خروجی را مشخص می‌کند. `infra/conf/freedom.go` تنظیمات خالی freedom را می‌پذیرد. `infra/conf/dns.go` برای hosts نگاشت object دارد و `infra/conf/policy.go` برای levels/system نوع object تعیین می‌کند. فقط آرایهٔ **خالی** در این مسیرهای مجاز به object تبدیل می‌شود؛ محتوا یا کلید ناشناخته دست نمی‌خورد.

تعمیر، نسخهٔ جداگانه می‌سازد. حذف کامای انتهایی بیرون رشته‌ها فقط پس از parse سخت‌گیرانهٔ بدون کلید تکراری پذیرفته می‌شود. در Custom دارای outbounds، نبود inbounds/routing با ظرف خالی تکمیل می‌شود؛ هیچ سرور، رمز، UUID، TLS یا مسیر ترافیک حدس زده نمی‌شود. فایل ناقص یا آرایهٔ دارای داده تبدیل حدسی ندارد. مثال‌های هم‌ارز مصنوعی با هستهٔ دقیق آزموده شدند؛ این شواهد جای تست هر کانفیگ واقعی روی Android یا اتصال را نمی‌گیرد.

`infra/conf/wireguard.go` و `WireguardFmt.kt` بررسی شدند: پارامترهای افزودهٔ Jc/Jmin/Jmax/S1…/H1… در مسیر پشتیبانی‌شدهٔ این هدف اثبات نشده‌اند؛ حذف آن‌ها ترمیم نیست. WireGuard پیوسته از پاسخ ترکیبی با بایت‌های اصلی استخراج می‌شود؛ خطوط پراکنده با کلیدهای نامعلوم به هم چسبانده نمی‌شوند.
