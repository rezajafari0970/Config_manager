import {$,fa} from './core.js';
export function renderAccounting(a){
 if(!a)return;
 const p=a.publication,q=a.queue,n=v=>fa.format(v);
 $('#egress-progress').textContent=n(p.published)+' کانفیگ قابل انتشار بعد از فیلتر · '+n(p.eligible)+' رزیدنتال با تأیید معتبر فعلی · '+n(p.awaiting_proof)+' پذیرفته‌شده در انتظار تأیید تازه';
 $('#egress-inventory').textContent=n(a.current_versions)+' نسخهٔ خام در منابع جاری / '+n(a.current_identities)+' هویت یکتا · '+n(a.archive_versions)+' نسخهٔ نگهداری‌شده خارج از منابع جاری · '+n(a.live_versions)+' نسخهٔ زنده در کل مخزن';
 $('#egress-queue').textContent='وضعیت کل مخزن: '+n(q.running)+' در حال بررسی · '+n(q.due)+' آمادهٔ بررسی · '+n(q.scheduled)+' زمان‌بندی‌شده · '+n(q.paused)+' متوقف‌شده. '+n(a.awaiting_first)+' نسخه هنوز نتیجهٔ ثبت‌شدهٔ بررسی جاری ندارد.';
 $('#egress-output-counts').textContent='خروجی: '+n(p.published)+' ردیف / '+n(p.identities)+' هویت / '+n(p.distinct_templates)+' قالب با محتوای متفاوت · '+n(p.country_filtered)+' حذف‌شده فقط توسط فیلتر کشور · '+n(p.country_unknown)+' بدون تأیید تازهٔ کشور';
}
