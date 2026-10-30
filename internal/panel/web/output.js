import {conversionCodes as codes} from './diagnostics.js';
import {$,fa,esc,dateTime,countdown,toast,api,empty} from './core.js';
export default function Output(ctx){
 let page=0,busy=false,signature='',url='',singleURL='';
 const source=new URLSearchParams(location.search).get('source')||'',sourceQuery=source?'&source='+encodeURIComponent(source):'';
 $('#all-output').hidden=!source;
 let sourcePage=0,sourceBusy=false,sourceSignature='',sourceRefreshAt=0,sourceTimer;

 async function refresh(){if(busy)return;busy=true;try{const s=await api('/api/conversions?page='+page+sourceQuery);$('#output-scope-title').textContent='کانفیگ‌های '+s.source_name;url=s.export_url;singleURL=s.single_export_url;$('#single-url').href=singleURL;$('#single-url').textContent=singleURL;$('#jsonl-url').href=url;$('#jsonl-url').textContent=url;$('#conversion-page').textContent=fa.format(page+1);$('#conversion-prev').disabled=page===0;$('#conversion-next').disabled=s.entries.length<100;
 const sig=JSON.stringify(s.entries);if(sig===signature)return;signature=sig;
 const host=$('#output-list'),cached=new Map([...host.querySelectorAll('pre')].map(p=>[p.dataset.hash,{text:p.textContent,hidden:p.hidden}]));
 host.innerHTML=s.entries.length?s.entries.map(c=>{const connected=c.connection==='passed'&&c.valid_until>Date.now();const core=c.core==='passed'?'هسته: پذیرفته شد':c.core==='failed'?'هسته: رد شد':c.state==='running'?'در حال تبدیل و بررسی':c.state==='queued'?'در صف بررسی':'هسته: آزمون اجرا نشد';const connection=connected?'اتصال HTTPS تأیید شد':c.connection==='partial'?'اتصال محدود':c.connection==='failed'?'اتصال برقرار نشد':'اتصال تأیید نشده';return `<article class="output-row"><div class="output-summary"><span class="protocol">${esc(c.kind)}</span><div class="output-meta"><b dir="ltr">@shenvpn · ${esc(c.id.slice(0,10))}</b><span>${core} · ${connection}</span><span>حذف تا ${countdown(c.expires)} دیگر · بررسی: ${dateTime(c.tested)}</span>${!c.available?`<span class="hint">${esc(codes[c.code]||c.code||'تبدیل خودکار در انتظار اجراست')}</span>`:''}</div>${c.available?`<div class="output-actions"><button class="secondary" data-view="${c.id}" data-hash="${c.candidate_hash}">نمایش</button><button class="secondary" data-copy="${c.id}" data-hash="${c.candidate_hash}">کپی JSON</button></div>`:''}</div><pre dir="ltr" data-hash="${c.candidate_hash}" hidden></pre></article>`}).join(''):empty('هنوز کانفیگی موجود نیست','لینک‌ها را در Input اضافه کنید؛ تبدیل و بررسی خودکار انجام می‌شود.','output');
 for(const p of host.querySelectorAll('pre')){const v=cached.get(p.dataset.hash);if(v){p.textContent=v.text;p.hidden=v.hidden}}
 }catch(e){toast(e.message)}finally{busy=false}}
 $('#copy-jsonl').addEventListener('click',async()=>{try{await navigator.clipboard.writeText(url);toast('لینک خروجی کپی شد.')}catch{toast('کپی انجام نشد؛ لینک را انتخاب کنید.')}});
 $('#copy-single').addEventListener('click',async()=>{try{await navigator.clipboard.writeText(singleURL);toast('لینک تک‌کانفیگ کپی شد.')}catch{toast('کپی انجام نشد؛ لینک را انتخاب کنید.')}});
 $('#conversion-prev').addEventListener('click',()=>{if(page>0){page--;signature='';refresh()}});$('#conversion-next').addEventListener('click',()=>{page++;signature='';refresh()});
 $('#output-list').addEventListener('click',async e=>{const b=e.target.closest('[data-view],[data-copy]');if(!b)return;b.disabled=true;try{const p=b.closest('article').querySelector('pre');if(b.dataset.view&&!p.hidden){p.hidden=true;return}const r=await fetch('/output/json/'+(b.dataset.view||b.dataset.copy)+(source?'?source='+encodeURIComponent(source):''),{cache:'no-store'});if(!r.ok)throw new Error('این خروجی تغییر کرده یا منقضی شده است.');const bytes=await r.arrayBuffer();const hash=[...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))].map(v=>v.toString(16).padStart(2,'0')).join('');if(hash!==r.headers.get('X-Content-SHA256'))throw new Error('دریافت کامل نشد.');const text=new TextDecoder('utf-8',{fatal:true}).decode(bytes);if(b.dataset.copy){await navigator.clipboard.writeText(text);toast('JSON تک‌خطی کپی شد.')}else{p.textContent=text;p.dataset.hash=hash;p.hidden=false}}catch(e){toast(e.message)}finally{b.disabled=false}});
 async function refreshSources(force=false){
  if(sourceBusy||(!force&&(!$('#export-sources').open||Date.now()-sourceRefreshAt<5000)))return;
  sourceBusy=true;sourceRefreshAt=Date.now();
  const requestedPage=sourcePage,query=$('#export-source-search').value;
  try{
   const result=await api('/api/export/sources?'+new URLSearchParams({page:requestedPage,q:query}));
   if(requestedPage!==sourcePage||query!==$('#export-source-search').value){sourceRefreshAt=0;return}
   $('#export-source-page').textContent=fa.format(sourcePage+1);$('#export-source-prev').disabled=sourcePage===0;$('#export-source-next').disabled=!result.has_more;
   const sig=JSON.stringify(result.sources);if(sig===sourceSignature)return;sourceSignature=sig;
   $('#export-source-list').innerHTML=result.sources.length?result.sources.map(s=>`<article class="export-source-row" data-source="${esc(s.id)}"><div><strong>${esc(s.name)}</strong><span class="hint">${s.enabled?'فعال':'غیرفعال'} · ${fa.format(s.config_count)} کانفیگ دریافت‌شده</span></div><div class="row-actions"><a class="button secondary" data-source-panel href="${esc(s.panel_url)}">مشاهده</a><a class="button secondary" data-source-all href="${esc(s.export_url)}">خروجی همه</a><a class="button secondary" data-source-single href="${esc(s.single_export_url)}">خروجی تکی</a></div></article>`).join(''):empty('منبعی پیدا نشد','لینک‌ها را در Input اضافه کنید.','input');
  }catch(e){toast(e.message)}finally{sourceBusy=false}
 }
 $('#export-sources').addEventListener('toggle',()=>{if($('#export-sources').open)refreshSources(true)});
 $('#export-source-search').addEventListener('input',()=>{clearTimeout(sourceTimer);sourceTimer=setTimeout(()=>{sourcePage=0;refreshSources(true)},250)});
 $('#export-source-prev').addEventListener('click',()=>{if(sourcePage>0){sourcePage--;refreshSources(true)}});
 $('#export-source-next').addEventListener('click',()=>{sourcePage++;refreshSources(true)});
 return {render(){refresh();refreshSources()},filters(){return {}}};
}
