# عملکرد اسمارت — P20

نسخهٔ هدف: v2rayNG 2.3.6 / Lib40 / Xray26.7.28. نسخهٔ اسمارت v7، قالب عادی v4 و schema10.

## قالب فعلی

دو بالانسر leastLoad با expected3، maxRTT2s و tolerance0.15 فعال‌اند. گروه اصلی همهٔ اعضا را دارد؛ Bootstrap فقط از سرورهایی استفاده می‌کند که آدرس و وابستگی اتصالشان IP مستقل است. تعداد و عرض شمارهٔ تگ‌ها از همان خروجی محاسبه می‌شود. نبود سرور IP مستقل، خطای روشن تولید می‌کند.

دو Bootstrap تونلی Google/Cloudflare و دو DNS عمومی Cloudflare/Google، timeout2500ms، پرس‌وجوی موازی، کش و پاسخ stale60s مطابق درخواست اعمال می‌شوند. دامنه‌های full فقط endpoint واقعی‌اند. burst15s/sampling3/timeout5s، Keepalive45/15، آمار سیستم و متریک محلی11111 فعال و لاگ‌ها خاموش‌اند. اطلاعات اتصال و دادهٔ خام حفظ می‌شوند؛ خروجی‌های عادی تغییر نکرده‌اند.

محدودیت مشاهده‌شده: UseIPv4 طبق درخواست حفظ شده، اما در هستهٔ هدف، خطای DNS داخلی می‌تواند به تلاش برای DNS سیستم منجر شود؛ این حالت در راه‌اندازی و قطع Bootstrap ممکن است. فایروال آزمایش این تلاش‌ها را مسدود و شمارش می‌کند؛ آن فایروال جزو JSON خروجی نیست. بنابراین این قالب به‌تنهایی تضمین ضدنشتی نیست.

نتایج فعلی در checkpoint P20 ثبت می‌شوند. این تغییرات ادعای انتخاب قطعی زیر۲ثانیه، افزایش سرعت اپراتور، مهاجرت نشست TCP یا حذف کامل اختلال ندارند. Custom JSON قبلی باید دوباره دریافت یا ایمپورت شود.

## گزارش تاریخی P17 با leastPing

متن و جدول زیر رفتار نسخهٔ قبلی را ثبت می‌کنند؛ استراتژی فعلی leastLoad است.

## اصلاح‌های عملی

خروجی تکی فقط میان JSONهای آماده و معتبر می‌چرخد. عضو در حال بازسازی، منقضی یا لغوشده دور را قفل نمی‌کند. در هر دورِ مجموعهٔ آماده، جایگاه اول تکرار نمی‌شود؛ موارد تازه‌آماده وارد دور بعد می‌شوند. اگر هیچ خروجی معتبری موجود نباشد، پاسخ503 با پیام و Retry-After داده می‌شود؛ پاسخ موفقِ خالی داده نمی‌شود. HEAD و prefetch نوبت مصرف نمی‌کنند. نوبت‌ها همچنان پایدار، مشترک بین مصرف‌کنندگان همان لینک و مستقل از خروجی‌های عادی/منبعی‌اند.

قالب اسمارت، به درخواست جدیدِ بهبود عملکرد، از قرارداد لفظ‌به‌لفظ P15 عبور کرده است. همهٔ outboundهای اعضا، اطلاعات اتصال، TLS و ترتیب پایدار حفظ می‌شوند. فقط نام‌گذاری لازم برای جلوگیری از برخورد تغییر می‌کند. لاگ‌ها خاموش‌اند. دو helper عمومی DNS اضافه می‌شوند؛ پنج ورودیِ چهارتایی در این حالت24outbound دارند.

| بخش | P15/P16 | P17 |
|---|---|---|
| انتخاب مسیر | leastLoad، انتخاب تصادفی میان مجموعهٔ expected3 | leastPing، مسیر زنده با کمترین تأخیر مشاهده‌شده |
| بررسی کلاینت | burst، interval20s، sampling3، timeout4s | burst، interval10s، sampling1، timeout3s |
| نبود مسیر مشاهده‌شدهٔ سالم | بدون fallback صریح | block؛ عدم خروج مستقیم ناخواسته |
| DNS برنامه | بدون مسیر اختصاصی پورت53 | DNS outbound برای ورودی socks و tun |
| DNS غیر A/AAAA | وابسته به DNS مقصد | قواعد صریح؛ عبور از loopback و balancer تا resolver روی TCP |
| راه‌اندازی نام سرور | بدون bootstrap مستقل | دو DoH محلی، فقط full: نام endpoint، skipFallback و disableFallbackIfMatch |
| پذیرش عضو | دو مقصد عمومی HTTPS | همان دو مقصد و پاسخ200/206 صفحهٔ اصلی Instagram، با TLS معتبر |

عبارت direct در قانون DNS به معنی عبور دادن سؤال به اتصال خروجی DNS است؛ آن اتصال با dialerProxy به loopback و سپس به balancer پروکسی می‌رود. مسیر اینترنت مستقیم ایجاد نمی‌کند. A/AAAA به DoHهای داخل تونل می‌روند؛ سؤال‌های دیگر از تونل به1.1.1.1:53 روی TCP فرستاده می‌شوند. این تفکیک با سؤال A و TXT در sandbox آزمایش شده است.

bootstrap صرفاً از آدرس واقعی endpointها، WireGuard peers و XHTTP download endpoint استخراج می‌شود؛ SNI، Host و قوانین دامنهٔ برنامه‌ها وارد این فهرست نمی‌شوند. داده‌های خام و outboundهای اصلی بازنویسی نمی‌شوند. این bootstrap هنوز به دسترسی شبکهٔ کاربر به resolverهای مشخص‌شده وابسته است؛ برتری آن روی همهٔ اپراتورها اثبات نشده است.

## چرا تست سایت اضافه شد؟

در بررسی دو اسمارت روی شبکهٔ سرور، یکی صفحهٔ اصلی Instagram را با200 دریافت کرد، ولی دیگری با وجود موفقیت گوگل و کلادفلر، خطای TLS داشت. بنابراین سلامت عمومی یا پینگ Google شاهد کافی برای آن سایت نیست.

آزمون Instagram پس از موفقیت دو مقصد عمومی، در همان sandbox و از همان کانفیگ اجرا می‌شود؛ زمان، اندازهٔ پاسخ و TLS محدود/کنترل شده‌اند. فقط نتیجهٔ تازه و متعلق به همان source/raw/candidate با کد https_multi_site_verified وارد اسمارت می‌شود. شکست یا نتیجهٔ قدیمی، عضو را کنار می‌گذارد؛ موفقیت بازآزمایی خودکار آن را برمی‌گرداند. دو مقصد عمومی همچنان معیار وضعیت اتصال عادی‌اند؛ شکست سایت، خروجی عادیِ core-valid را حذف نمی‌کند. تاریخچه یا بانک دائمی ساخته نمی‌شود.

این شرط، موفقیت صفحهٔ اصلی در شبکهٔ سرور را می‌سنجد، نه پخش ویدئو، همهٔ CDNها، حساب کاربر یا اجرای اپلیکیشن. برای ورودی دارای بالانسر داخلی، موفقیت مسیرِ ورودی اثبات سلامت تک‌تک سرورهای داخلی نیست. مشاهدات کلاینت، اعضای اصلیِ گسترش‌یافته را جداگانه بررسی می‌کنند، اما معیار فعلی کلاینت همچنان HTTP204 عمومی است.

## شواهد آزمایش

مقایسه با همان دو peer مصنوعی در شبکهٔ بسته انجام شد؛ یکی120ms تأخیر در هر جهت داشت. در24درخواست، قالب قبلی11درخواست را به مسیر کند فرستاد و صدک95حدود243.63ms بود. قالب جدید24درخواست را از مسیر سریع گذراند و صدک95حدود2.04ms شد. این اعداد، تأخیر مصنوعی آزمایشگاه‌اند؛ افزایش سرعت اینترنت یا دانلود گوشی نیستند.

پس از قطع عمدی peer سریع، در همان اجرا مسیر دوم پس از5.48ثانیه پاسخ داد؛ چهار درخواست هنگام کشف خرابی شکست خورد. این شاهد بازیابی است، نه سوییچ بی‌وقفه یا سقف تضمین‌شدهٔ زمان بازیابی. زمان‌بندی burst تصادفی است؛ interval10s وعدهٔ تعویض دقیق هر10ثانیه نیست.

آزمون دامنهٔ endpoint، DNS نوعA/TXT، TCP، UDP و UDP443، ماتریس پذیرش هسته1/2/10/50عضوی، رقابت نوبت‌دهی، ورودی قدیمی، بازسازی کش، HEAD، مرورگر و بازیابی کارگر پوشش داده می‌شوند. echo روی UDP443 مذاکرهٔ واقعی QUIC/HTTP3 نیست.

## محدودهٔ نتیجه

| شبکه/شاخص | وضعیت |
|---|---|
| هستهٔ دقیق روی سرور و شبکهٔ آزمایش بسته | آزمایش شده |
| درخواست واقعی سایت از sandbox سرور | در receipt خصوصی ثبت می‌شود |
| همراه اول / ایرانسل / رایتل / Wi-Fi گوشی | not_run |
| ایمپورت و VPN در APK واقعی هدف | not_run |
| سرعت دانلود گوشی، jitter، پکت‌لاس واقعی UDP و HTTP3 | اندازه‌گیری نشده |

تغییر outbound اتصال‌های جدید را هدایت می‌کند. این JSON نشست TCP موجود را میان دو سرور مستقل منتقل نمی‌کند. برای انتخاب بهترین تنظیم در هر اپراتور، باید زمان بارگذاری Instagram و CDNها، صدک95تأخیر، دانلود محدود، بسته‌های UDP شماره‌دار و بازیابی هنگام جابه‌جایی شبکه در همان گوشی ثبت شود. موفقیت آزمایش سرور جای این اندازه‌گیری را نمی‌گیرد.

در بررسی تاریخی P17 تعداد اعضا10بود؛ مقدار P19 برابر35است. بیشترکردن آن معادل تجمیع سرعت سرورها نیست. Custom JSON که قبلاً در برنامه کپی شده، خودکار تغییر نمی‌کند؛ JSON تازهٔ لینک تکی را دوباره ایمپورت کنید.

## مبنای فنی

سورس‌های pinned:
- Xray commit5ca6f4b7d4dc20a881d4330e498892697627ec0c: app/router/strategy_leastping.go، app/observatory/burst/healthping.go و burstobserver.go، proxy/dns/dns.go و infra/conf/dns_proxy.go.
- v2rayNG commit13138dddcab291ba3618cc54721ce4c7dd926be0: CoreConfigManager.buildV2rayCustomConfig؛ در حالت TUN ورودی tun اضافه می‌کند.

مستندات مرجع:
- https://xtls.github.io/en/config/routing.html
- https://xtls.github.io/en/config/observatory.html
- https://xtls.github.io/en/config/dns.html
- https://xtls.github.io/en/config/outbounds/dns.html

## P24 exact observer timing

The requested leastLoad/15s interval/3 samples/5s timeout envelope is unchanged. Pinned burst healthping StartScheduler uses interval*sampling (45s) as its scheduling window and randomizes probes within it; maxRTT2s is an admission threshold, not a failover timer. shouldSelectNode excludes failed fractions above0.15 even if alive remains true. The P24 fixture now checks that predicate before declaring recovery, instead of three random successful requests. One controlled run measured24.61s until observed exclusion and26.33s until verified recovery, with11 failed requests during detection and24/24 subsequent successes. This test neither establishes a general upper latency bound nor Android/operator performance. UI/API latency improvements are separate measurements.
