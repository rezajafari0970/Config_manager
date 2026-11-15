import {esc,fa} from './core.js';
const names={'doubleclick.v1':'DoubleClick','ads-serving.v1':'Google Ads Serving'};
export function profileSummary(x){
 if(!x)return '';
 return ' · '+fa.format(x.complete_tested_ads||0)+' با جداسازی تمام مقصدهای Ads آزمایش‌شده · '+fa.format(x.partial_tested_ads||0)+' با جداسازی پروفایل مشخص · '+fa.format(x.published||0)+' قابل انتشار پس از فیلتر کشور';
}
export function profileDetails(d){
 if(!d?.qualified_profiles?.length)return '';
 const coverage=d.ads_coverage==='ALL_TESTED'?'همهٔ مقصدهای Ads آزمایش‌شده جدا بودند':'پوشش جزئی Ads؛ مقصدهای دیگر ممکن است مشترک یا نامشخص باشند';
 return `<p>پروفایل تأییدشده: ${esc(d.qualified_profiles.map(x=>names[x]||x).join('، '))}</p><p class="hint">${esc(coverage)}</p><p class="hint">مقصدهای تأییدشده: <span dir="ltr">${esc((d.qualified_domains||[]).join(' · '))}</span></p>`;
}
