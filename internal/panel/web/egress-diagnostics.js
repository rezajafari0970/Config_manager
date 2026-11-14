// Explain observed transport failures without changing route classification.
// Read only the chosen report: its nested attempt evidence repeats samples.
export function connectionDiagnostics(report){
 const rows=Array.isArray(report?.samples)?report.samples:[];
 const codes=new Set(['connection_failed','connection_timeout','tls_verification_failed','socks_greeting','socks_connect','socks_address']);
 const failed=rows.filter(x=>codes.has(x.code)&&!x.tls);
 const stages={local_socks:'اتصال به آزمایشگر محلی',socks_negotiation:'مذاکره با آزمایشگر',proxy_connect:'درخواست اتصال از پروکسی',target_tls:'برقراری TLS مقصد',http_exchange:'تبادل HTTP'};
 const kinds={connection_reset:'قطع اتصال',connection_refused:'اتصال رد شد',unexpected_eof:'پایان زودهنگام اتصال',tls_handshake:'خطای TLS',certificate_verification:'گواهی نامعتبر',timeout:'پایان مهلت',protocol_response:'پاسخ نامعتبر',io_error:'خطای ورودی/خروجی',unexpected_error:'خطای پیش‌بینی‌نشده'};
 const grouped=new Map();
 for(const x of failed){
  const stage=Object.hasOwn(stages,x.error_stage)?x.error_stage:'unspecified';
  const kind=Object.hasOwn(kinds,x.error_kind)?x.error_kind:'unspecified';
  const key=stage+'|'+kind;
  if(!grouped.has(key))grouped.set(key,{stage:stages[stage]||'در گزارش قدیمی ثبت نشده',kind:kinds[kind]||'در گزارش قدیمی ثبت نشده',samples:0});
  grouped.get(key).samples++;
 }
 return {sample_count:rows.length,failed_before_tls:failed.length,details:[...grouped.values()],explanation:failed.length&&failed.length===rows.length?'تمام نمونه‌ها پیش از تأیید TLS شکست خورده‌اند؛ مسیر Ads قابل مقایسه نیست. این نتیجه به معنی غیررزیدنتال بودن نیست.':failed.length?'بخشی از نمونه‌ها پیش از تأیید TLS شکست خورده‌اند؛ جزئیات هر مقصد را بررسی کنید.':'برای تشخیص مسیر، شواهد مقصدها و تصمیم ثبت‌شده را بررسی کنید.'};
}
