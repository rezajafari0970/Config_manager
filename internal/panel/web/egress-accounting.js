import {$,fa} from './core.js';
export function renderAccounting(a){
 if(!a)return;
 const p=a.publication,q=a.queue,n=v=>fa.format(v);
 $('#egress-progress').textContent=n(p.eligible)+' رزیدنتال با تأیید معتبر · '+n(p.published)+' کانفیگ یکتا در خروجی · '+n(p.awaiting_proof)+' رزیدنتال در انتظار تمدید تأیید';
 $('#egress-inventory').textContent=n(a.current_identities)+' کانفیگ یکتای منابع فعلی · '+n(a.retained_outside_current)+' رزیدنتال نگهداری‌شده خارج از منابع فعلی؛ هر هویت فقط یک نمایندهٔ آزمون دارد.';
 $('#egress-queue').textContent=n(a.assessed_versions)+' کانفیگ یکتا بررسی‌شده · '+n(a.awaiting_first)+' بدون نتیجهٔ اولیه · '+n(q.running)+' در حال بررسی · '+n(q.due)+' آماده · '+n(q.scheduled)+' پیگیری زمان‌بندی‌شده · '+n(q.cached)+' نتیجهٔ ذخیره‌شده بدون تکرار · '+n(q.paused)+' متوقف';
 $('#egress-output-counts').textContent=n(p.published)+' ردیف خروجی = '+n(p.identities)+' هویت یکتا · '+n(p.country_filtered)+' خارج از انتخاب کشور · '+n(p.country_unknown)+' در انتظار تعیین معتبر کشور';
}
