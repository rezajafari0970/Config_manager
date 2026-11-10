import {$,api,esc,fa,dateTime,toast} from './core.js';
export default function(){
 let current=null;
 const titles={'configs.jsonl':'همه با قالب @shenvpn · JSONL','next.json':'تک‌کانفیگ چرخشی · JSON','raw.zip':'همهٔ کانفیگ‌های خام · ZIP'};
 const labels={ACCEPTED:'تأیید در محدودهٔ آزمون',PARTIAL:'پوشش ناقص',UNKNOWN:'شواهد ناکافی',UNSUPPORTED:'درایور یا قابلیت پشتیبانی نشده',QUARANTINED:'قرنطینه؛ خارج از انتشار',NON_RESIDENTIAL:'شاهد غیررزیدنتال',DELETED:'حذف تأییدشده'};
 $('#res-links').addEventListener('click',async e=>{const b=e.target.closest('[data-copy]');if(!b||!current)return;try{await navigator.clipboard.writeText(current.links[b.dataset.copy]);toast('لینک کپی شد')}catch{toast('کپی انجام نشد؛ لینک را انتخاب کنید.')}});
 $('#res-list').addEventListener('click',async e=>{const b=e.target.closest('[data-res-action]');if(!b)return;const action=b.dataset.resAction;if(action==='delete'&&!confirm('این کانفیگ از کل سیستم حذف شود؟'))return;b.disabled=true;try{await api('/api/residential/manage',{method:'POST',body:{hash:b.dataset.hash,action}});await refresh();toast('وضعیت به‌روزرسانی شد')}catch(error){toast(error.message)}finally{b.disabled=false}});
 async function refresh(){
  try{const state=await api('/api/residential'+location.search);current=state;
   $('#res-count').textContent=fa.format(state.eligible_count)+' عضو قابل انتشار';
   $('#res-state').textContent=(state.worker.status==='running'?'پردازش خودکار فعال است. ':'پردازش در دسترس نیست؛ خروجی فقط با شواهد معتبر ارائه می‌شود. ')+(state.trust_configured?'':'روش قابل اتکای اثبات خروجی و منابع تخصصی نوع IP هنوز پیکربندی نشده‌اند؛ هیچ نتیجهٔ نامطمئنی رزیدنتال منتشر نمی‌شود.');
   $('#res-links').innerHTML=Object.entries(state.links).map(([format,url])=>`<div class="subscription-box"><label>${esc(titles[format])}</label><div class="sub-link"><a href="${esc(url)}" dir="ltr">${esc(url)}</a><button class="secondary" data-copy="${esc(format)}">کپی لینک</button></div></div>`).join('');
   $('#res-policy').textContent=JSON.stringify({version:state.policy.domains_version,domains:state.policy.domains,families:state.policy.families,transports:state.policy.transports,repeats:state.policy.repeats,evidence_ttl_seconds:state.policy.evidence_ttl_seconds},null,2);
   $('#res-updated').textContent='آخرین دریافت وضعیت: '+dateTime(state.now);
   $('#res-metrics').textContent=`آزمون سرور · p50: ${fa.format(state.metrics.p50_ms)} ms · p95: ${fa.format(state.metrics.p95_ms)} ms · کانفیگ در دقیقه: ${fa.format(state.metrics.configs_per_minute_last_5m)}`;
   $('#res-generation').textContent='Generation: '+state.generation;
   $('#res-list').innerHTML=state.entries.length?state.entries.map(x=>`<article class="config-card"><strong>${esc(labels[x.state]||x.state)}</strong><p dir="ltr">${esc(x.reason)}</p><p>سلامت: ${esc(x.decision.health||'UNKNOWN')} · خروجی: ${esc(x.decision.egress||'UNKNOWN')} · شبکه: ${esc(x.decision.network||'Unknown')}</p><p>پوشش: ${fa.format(x.decision.covered||0)} از ${fa.format(x.decision.required||0)} · آزمون: ${esc(dateTime(x.tested))}</p><a class="button secondary" href="/api/residential/evidence/${esc(x.hash)}" target="_blank" rel="noopener">شواهد و علت تصمیم</a>${x.state==='DELETED'?'':`<div class="row-actions"><button class="secondary" data-res-action="recheck" data-hash="${esc(x.hash)}">آزمون دوباره</button><button class="secondary" data-res-action="disable" data-hash="${esc(x.hash)}">توقف انتشار</button><button class="danger" data-res-action="delete" data-hash="${esc(x.hash)}">حذف از کل سیستم</button></div>`}</article>`).join(''):'<p class="hint">هنوز ورودی برای ارزیابی ثبت نشده است.</p>';
  }catch(error){$('#res-state').textContent=error.message;return false}
 }
 return {filters:()=>({}),render(){},tasks:[refresh]};
}
