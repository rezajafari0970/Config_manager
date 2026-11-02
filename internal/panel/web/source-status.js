// Receipt quality, current ownership and conversion eligibility are separate.
export function sourceStatus(s, conversion={}) {
 const statuses={queued:['در انتظار','wait'],fetching:['در حال دریافت','working'],error:['ناموفق','bad'],paused:['متوقف','']};
 if(!s.enabled)return statuses.paused;
 if(statuses[s.state])return statuses[s.state];
 if(s.state==='filtered'||(s.config_count===0&&s.suppressed_count>0))return ['ذخیره نشد: قبلاً حذف شده','wait'];
 if(s.config_count===0&&(['updated','unchanged'].includes(s.state)||s.content_state==='config'))return ['کانفیگ فعالی ذخیره نیست','wait'];
 if(s.suppressed_count>0&&s.config_count>0)return ['بخشی ذخیره شد؛ برخی قبلاً حذف شده‌اند','wait'];
 if(conversion.blocked>0)return ['برخی ورودی‌ها نیازمند تکمیل یا اصلاح‌اند','wait'];
 if(s.config_count>0&&conversion.total>0&&conversion.ready===0&&conversion.pending>0)return ['ذخیره شد؛ خروجی در حال بررسی','wait'];
 if(s.config_count>0&&conversion.total>0&&conversion.ready===conversion.total&&['partial','invalid'].includes(s.content_state))return ['خروجی‌های شناسایی‌شده آماده‌اند','good'];
 const content={config:['کانفیگ ذخیره شد','good'],partial:['بخشی از ورودی ایراد دارد','wait'],invalid:['قالب خراب','bad'],empty:['پاسخ خالی','wait'],unrecognized:['بدون کانفیگ قابل‌شناسایی','bad'],error_response:['پاسخ خطا / غیرکانفیگ','bad'],unverified:['قالب نیازمند بررسی','wait']};
 return content[s.content_state]||['بدون نتیجهٔ قابل تأیید','wait'];
}
