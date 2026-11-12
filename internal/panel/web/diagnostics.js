export const conversionCodes={
legacy_http_transport_removed:'انتقال قدیمی H2/HTTP در هستهٔ هدف حذف شده است؛ تغییر نام به XHTTP اتصال هم‌ارز ایجاد نمی‌کند.',
legacy_quic_transport_removed:'انتقال قدیمی QUIC حذف شده است؛ این محدودیت به معنی مسدود بودن ترافیک UDP/443 نیست.',
legacy_vless_transport_requires_driver:'این انتقال VLESS بدون TLS هنوز آداپتور تأییدشده ندارد؛ نسخهٔ خام حفظ شده است.',
relay_download_requires_driver:'مسیر جداگانهٔ دانلود XHTTP بدون TLS به بررسی اختصاصی نیاز دارد؛ تغییر حدسی اعمال نشد.',
relay_stream_override_requires_driver:'گزینهٔ انتقال یا ماسک این ورودی برای آداپتور بدون TLS تأیید نشده است.',
relay_proxy_settings_requires_driver:'زنجیرهٔ proxySettings با آداپتور فعلی نیازمند بررسی اختصاصی است.',
relay_egress_shared:'مسیر انتقال ثابت بین چند مصرف‌کننده مشترک است؛ حذف یا بازنویسی آن مطمئن نیست.',
protocol_requires_other_client_core:'پروتکل در هستهٔ Xray هدف پشتیبانی نمی‌شود؛ تغییر قالب جای هستهٔ سازگار را نمی‌گیرد.',
transport_verification_incomplete:'هسته پذیرفت و HTTPS برقرار شد؛ بررسی DNS یا UDP کامل نبود. این خطای ساخت قالب نیست.',
probe_preparation_failed:'قالب توسط هسته بررسی شد، اما شرایط اجرای آزمون شبکه فراهم نشد؛ نیازمند تشخیص مسیر است.',

client_vless_encryption_required:'ناسازگار با هستهٔ جدید: در این کانفیگ، رمزنگاری VLESS فعال نیست؛ TLS، REALITY یا VLESS Encryption باید واقعاً در سرور پشتیبانی شود. نسخهٔ خام حفظ شده است.',
client_trojan_tls_required:'ناسازگار با هستهٔ جدید: این سرور Trojan به TLS واقعی نیاز دارد. نسخهٔ خام حفظ شده است.',
client_core_incompatible:'JSON خروجی توسط Xray 26.11.9 پذیرفته نشد؛ تا اصلاح سازگاری وارد خروجی ساده یا اسمارت نمی‌شود.',
client_core_validation_timeout:'بررسی سازگاری با هستهٔ جدید به مهلت رسید؛ بررسی تکرار می‌شود.',
client_core_asset_missing:'فایل وابستهٔ هستهٔ بررسی در دسترس نیست؛ این وضعیت خرابی سرور نیست.',
client_core_infrastructure_error:'بررسی سازگاری با هستهٔ جدید موقتاً در دسترس نیست؛ این وضعیت خرابی سرور نیست.',
reality_authentication_rejected:'احراز هویت REALITY پذیرفته نشد؛ تطابق کلید، شناسه و تنظیمات سمت سرور باید بررسی شود',peer_reset_connection:'سرور یا مسیر شبکه اتصال را قطع کرد',tls_certificate_failure:'اعتبارسنجی گواهی TLS موفق نبود',connection_timeout:'مهلت اتصال به پایان رسید',invalid_download_endpoint:'آدرس یا پورت مسیر دانلود XHTTP ناقص است',https_payload_verified:'دانلود ۲ مگابایت، آپلود ۱ مگابایت و بررسی پس از انتقال روی سرور تأیید شد؛ آزمون گوشی جداست',https_payload_incomplete:'HTTPS برقرار است اما انتقال کامل دانلود یا آپلود تأیید نشد؛ ورود به اسمارت متوقف است',https_multi_site_verified:'اتصال عمومی و صفحهٔ اصلی اینستاگرام روی سرور تأیید شد',https_proxy_verified_site_unavailable:'اتصال عمومی برقرار است؛ آزمون اینستاگرام موفق نبود و این ورودی وارد اسمارت نمی‌شود',ambiguous_proxy_selection:'چند خروجی پراکسی دارد؛ انتخاب مسیر مبهم است',hysteria1_unsupported_by_target_core:'Hysteria 1 توسط هستهٔ هدف پشتیبانی نمی‌شود',missing_proxy_dependency:'وابستگی مسیر پراکسی ناقص است',cyclic_proxy_chain:'مسیر پراکسی دارای حلقه است',missing_proxy_outbound:'خروجی پراکسی قابل تبدیل پیدا نشد',public_endpoint_requires_tls:'سرور به TLS یا رمزنگاری سازگار نیاز دارد',reality_mlkem_hint_incompatible:'گزینهٔ ML-KEM برای VLESS REALITY لازم است',reality_mlkem_disable_unsupported:'غیرفعال‌کردن ML-KEM با این پارامتر در هستهٔ هدف پشتیبانی نمی‌شود',reality_mlkem_hint_invalid:'مقدار گزینهٔ ML-KEM معتبر نیست',invalid_vmess_certificate_policy:'مقدار سیاست گواهی VMess معتبر نیست',ambiguous_vmess_certificate_policy:'گزینه‌های اعتبارسنجی گواهی VMess با هم تناقض دارند',unknown_query_parameter:'پارامتر ناشناخته؛ تبدیل مطمئن نیست',structural_unresolved:'اطلاعات ساختاری کافی برای تبدیل وجود ندارد',removed_allow_insecure:'گزینهٔ TLS در این هسته حذف شده است',prepared:'آمادهٔ بررسی',missing_reality_public_key:'کلید عمومی REALITY در ورودی نیست؛ لینک کامل با pbk یا JSON دارای publicKey/password لازم است',
invalid_reality_public_key:'کلید عمومی REALITY نامعتبر است؛ کلید صحیح منبع لازم است',
invalid_reality_short_id:'شناسهٔ کوتاه REALITY باید هگز زوج و حداکثر ۱۶ نویسه باشد',
unsupported_packet_encoding:'نوع بسته‌بندی UDP این ورودی با نگاشت فعلی سازگار نیست',
cipher_suites_require_tls:'پارامتر cs فقط برای اتصال TLS قابل اعمال است',
invalid_uri:'ساختار لینک مبهم یا نامعتبر است؛ اصلاح بدون حدس ممکن نشد',
reality_handshake_or_settings:'تنظیمات یا مذاکرهٔ REALITY پذیرفته نشد؛ اطلاعات سرور باید بررسی شود',
probe_infrastructure_error:'زیرساخت بررسی موقتاً در دسترس نیست'};

// One closed diagnostic policy for Input and Output. Diagnostic severity never
// grants export eligibility; server-side exact-byte evidence remains authoritative.
Object.assign(conversionCodes,{
 invalid_base64:'بدنهٔ لینک پس از جداسازی نام و بررسی encoding قابل Decode نیست؛ دادهٔ اصلی ناقص یا نامعتبر است. این ردشدن توسط هسته نیست.',
 invalid_port:'پورت موجود نیست یا خارج از بازهٔ ۱ تا ۶۵۵۳۵ است؛ شمارهٔ پورت سرور حدس زده نمی‌شود.',
 incomplete_transport_parameter:'لینک در پارامتر انتقال قطع شده است؛ نسخهٔ کامل منبع لازم است و اطلاعات حذف‌شده حدس زده نمی‌شود.',
 invalid_vmess_payload_escape:'کدگذاری درصدی بدنهٔ VMess ناقص است؛ متن خام حفظ شده است.',
 invalid_reality_version_annotation:'توضیح نسخهٔ REALITY قالب شناخته‌شده ندارد؛ گزینهٔ شبکه حدسی حذف نمی‌شود.',
 conflicting_vmess_cipher:'دو مقدار متفاوت برای رمزنگاری داخلی VMess داده شده است؛ انتخاب خودکار انجام نمی‌شود.',
 invalid_vmess_cipher_alias:'فیلد معادل رمزنگاری VMess مقدار معتبر ندارد.',
 unsupported_vmess_cipher_alias:'رمزنگاری داخلی VMess نگاشت شناخته‌شده ندارد؛ به none تنزل داده نمی‌شود.',
 relay_invalid_destination:'مقصد برای واسط ثابت معتبر نیست؛ کروشهٔ IPv6 معتبر خودکار اصلاح می‌شود، اما آدرس ناقص حدس زده نمی‌شود.',

 unknown_vmess_field:'فیلد ناشناختهٔ VMess بررسی نشده است؛ نسخهٔ خام حفظ شده و گزینهٔ مؤثر حدسی حذف نمی‌شود.',
 vmess_extension_requires_driver:'این افزونهٔ فعال VMess هنوز نگاشت تأییدشده ندارد؛ مقدار خالی با گزینهٔ فعال یکسان نیست.',
 invalid_vmess_metadata:'فیلد توضیحی VMess باید متن باشد؛ ساختار ورودی نیازمند اصلاح است.',
 invalid_tls_verification_flag:'مقدار گزینهٔ بررسی گواهی معتبر نیست.',
 conflicting_tls_verification:'گزینه‌های بررسی گواهی با هم تعارض دارند؛ TLS بدون بررسی گواهی فعال نمی‌شود.',
 invalid_share_metadata:'فیلد توضیحی اشتراک نامعتبر است.',
 concatenated_config_uri:'چند لینک داخل فیلد انتقال به هم چسبیده‌اند؛ منبع باید لینک‌های جدا و کامل بدهد.',
 wireguard_extension_unsupported:'افزونهٔ این WireGuard در درایور فعلی نگاشت تأییدشده ندارد؛ کلیدها و گزینه‌های اصلی حفظ شده‌اند.',
 conflicting_finalmask:'دو مقدار متفاوت برای FinalMask داده شده است؛ انتخاب حدسی انجام نمی‌شود.',
 legacy_vless_transport_requires_driver:'ترکیب انتقال این VLESS قدیمی به درایور سازگاری جدا نیاز دارد؛ رمزنگاری ساختگی اضافه نمی‌شود.',
 transport_driver_pending:'نوع انتقال در درایور فعلی پشتیبانی نشده است؛ نسخهٔ خام نگهداری می‌شود.',
 outbound_driver_pending:'درایور اجرای این خروجی در محیط آزمون موجود نیست.',
 protocol_driver_pending:'پروتکل ورودی هنوز درایور تبدیل تأییدشده ندارد.',
 ipv6_only_probe_pending:'سرور فقط IPv6 دارد؛ محیط آزمون فعلی مسیر IPv6 ندارد. این نتیجه خرابی کانفیگ را ثابت نمی‌کند.',
 endpoint_dns_failed:'نام میزبان سرور در محیط آزمون resolve نشد؛ آزمون با زمان‌بندی محدود تکرار می‌شود.',
 dns_failure:'پاسخ DNS در مسیر آزمون دریافت نشد؛ دریافت لینک با این خطای اتصال متفاوت است.',
 tls_name_mismatch:'نام مورد انتظار TLS با گواهی ارائه‌شده تطابق ندارد؛ SNI یا گواهی سمت سرور باید اصلاح شود. بررسی گواهی خاموش نمی‌شود.',
 tls_untrusted_certificate:'زنجیرهٔ گواهی TLS مورد اعتماد نیست؛ گواهی معتبر یا پین تأییدشدهٔ منبع لازم است.',
 tls_expired_certificate:'گواهی TLS منقضی شده است؛ اصلاح گواهی سمت سرور لازم است.',
 peer_closed_connection:'طرف مقابل اتصال را بست؛ این خطا به‌تنهایی نشانهٔ خراب‌بودن قالب نیست.',
 endpoint_refused:'سرور اتصال به پورت مشخص‌شده را نپذیرفت.',
 network_unreachable:'محیط آزمون به شبکهٔ مقصد مسیر ندارد؛ وضعیت خود سرور نامشخص است.',
 proxy_request_failed:'درخواست از تونل پاسخ معتبر نگرفت؛ آزمون خودکار تکرار می‌شود.',
 test_destination_partial:'فقط بخشی از مقصدهای آزمون موفق بود؛ سلامت کامل هنوز تأیید نشده است.',
 transport_verification_incomplete:'بخشی از آزمون‌ها موفق بوده، اما تأیید DNS یا UDP/QUIC کامل نیست؛ جزئیات آزمون را ببینید.',
 candidate_blob_unavailable:'فایل خروجی در دسترس نیست؛ بازسازی و اعتبارسنجی تازه لازم است.',
 probe_preparation_failed:'آماده‌سازی محیط آزمون کامل نشد؛ این نتیجه ردشدن اتصال سرور نیست.',
 removed_allow_insecure:'گزینهٔ قدیمی غیرفعال‌کردن بررسی گواهی در هستهٔ هدف حذف شده است؛ فقط آزمون با گواهی معتبر قابل انتشار است.'
});
const diagnosticCategories={
 environment:new Set(['ipv6_only_probe_pending','client_core_asset_missing','client_core_infrastructure_error','probe_infrastructure_error','probe_preparation_failed','candidate_blob_unavailable','network_unreachable']),
 capability:new Set(['unknown_vmess_field','vmess_extension_requires_driver','unknown_query_parameter','wireguard_extension_unsupported','legacy_vless_transport_requires_driver','transport_driver_pending','outbound_driver_pending','protocol_driver_pending','hysteria1_unsupported_by_target_core','unsupported_packet_encoding']),
 network:new Set(['https_proxy_verified_site_unavailable','endpoint_dns_failed','dns_failure','peer_closed_connection','peer_reset_connection','endpoint_refused','connection_timeout','proxy_request_failed','test_destination_partial','transport_verification_incomplete','client_core_validation_timeout']),
 server:new Set(['missing_reality_public_key','invalid_reality_public_key','invalid_reality_short_id','reality_authentication_rejected','reality_handshake_or_settings','tls_name_mismatch','tls_untrusted_certificate','tls_expired_certificate','tls_certificate_failure','removed_allow_insecure'])
};
export function conversionDetail(code) {
 const category=Object.keys(diagnosticCategories).find(k=>diagnosticCategories[k].has(code))||'input';
 const labels={environment:'محدودیت محیط آزمون',capability:'نیازمند درایور سازگار',network:'آزمون اتصال',server:'مشخصات یا پاسخ سرور',input:'ساختار ورودی'};
 return {category,label:labels[category],tone:category==='network'||category==='environment'?'wait':'bad',message:conversionCodes[code]||'نتیجهٔ این مورد نیازمند بررسی است؛ کد فنی در جزئیات آمده است.'};
}
export function evidenceRows(evidence) {
 const names={target_core:'هستهٔ هدف',bundled_core:'هستهٔ همراه برنامه',app_import_contract:'ساختار Import',https:'HTTPS با گواهی معتبر',dns_tcp:'DNS روی TCP',dns_udp:'DNS روی UDP',doh_cloudflare:'DoH کلادفلر',doh_google:'DoH گوگل',quic_vn:'پاسخ QUIC روی UDP/443',http3:'انتقال HTTP/3',android:'اجرای واقعی Android'};
 const states={passed:'موفق',failed:'ناموفق',partial:'ناقص',not_run:'اجرا نشده',unsupported:'پشتیبانی نمی‌شود',infrastructure_error:'خطای محیط آزمون'};
 const checks=evidence?.checks||{};
 return Object.entries(names).map(([key,label])=>({key,label,status:states[checks[key]??evidence?.[key]]||'نامشخص'}));
}
