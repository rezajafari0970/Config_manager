// Receipt quality, current ownership and conversion eligibility are separate.
export function sourceStatus(s, conversion={}) {
 const statuses={queued:['در انتظار','wait'],fetching:['در حال دریافت','working'],error:['ناموفق','bad'],paused:['متوقف','']};
 if(!s.enabled)return statuses.paused;
 if(s.error_code==='source_count_drop_pending'||s.refresh_reason==='source_count_drop_pending')return ['افت تعداد پاسخ؛ در حال بررسی دوباره','wait'];
 if(s.refresh_reason&&s.state==='queued')return ['در انتظار دریافت تازه','wait'];
 if(s.state==='error'&&s.error_code==='processing_timeout')return ['دریافت شد؛ پردازش نیازمند تکرار است','wait'];
 if(s.state==='error'&&s.config_count>0)return ['خطای دریافت تازه؛ ورودی‌های قبلی موجودند','wait'];
 if(statuses[s.state])return statuses[s.state];
 if(s.state==='filtered'||(s.config_count===0&&s.suppressed_count>0))return ['ذخیره نشد: قبلاً حذف شده','wait'];
 if(s.config_count===0&&(['updated','unchanged'].includes(s.state)||s.content_state==='config'))return [s.stored_count>0?'ورودی‌های قبلی دیگر موجود نیستند؛ دریافت تازه لازم است':'ورودی یکتای موجود ندارد','wait'];
 if(s.suppressed_count>0&&s.config_count>0)return ['بخشی ذخیره شد؛ برخی قبلاً حذف شده‌اند','wait'];
 if(s.config_count>0&&s.disabled_count>=s.config_count)return ['ورودی‌ها ذخیره‌اند؛ خروجی متوقف است','wait'];
 if(conversion.blocked>0)return ['دریافت شد؛ برخی خروجی‌ها نیازمند بررسی‌اند','wait'];
 if(s.config_count>0&&conversion.total>0&&conversion.ready===0&&conversion.pending>0)return ['ذخیره شد؛ خروجی در حال بررسی','wait'];
 if(s.config_count>0&&conversion.total>0&&conversion.ready===conversion.total&&['partial','invalid'].includes(s.content_state))return ['خروجی‌های شناسایی‌شده آماده‌اند','good'];
 const content={config:['کانفیگ ذخیره شد','good'],partial:['بخشی از ورودی ایراد دارد','wait'],invalid:['قالب خراب','bad'],empty:['پاسخ خالی','wait'],unrecognized:['بدون کانفیگ قابل‌شناسایی','bad'],error_response:['پاسخ خطا / غیرکانفیگ','bad'],unverified:['قالب نیازمند بررسی','wait']};
 return content[s.content_state]||['بدون نتیجهٔ قابل تأیید','wait'];
}

// Never infer receipt dedup from the legacy `duplicates` field: that field also
// counts identities already stored by an earlier request or another source.
export function receiptCounts(s) {
 const unique=Number.isInteger(s.receipt_unique)&&s.receipt_unique>=0?s.receipt_unique:null;
 const added=Number.isInteger(s.receipt_new)&&s.receipt_new>=0?s.receipt_new:null;
 return {unique, repeated:unique===null?null:Math.max(0,s.received_count-unique),
  added, existing:unique===null||added===null?null:Math.max(0,unique-added-(s.suppressed_count||0))};
}
