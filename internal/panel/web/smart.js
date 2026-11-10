import {conversionCodes} from './diagnostics.js';
import {$,fa,esc,countdown,toast,api,verifiedJSON,empty} from './core.js';
export default function Smart(){
 const base='/'+document.body.dataset.page;
 let busy=false,page=0,signature='';
 const codes={...conversionCodes,smart_ads_main_unavailable:'مسیر اصلی عضو سالم ندارد؛ خروجی ادز تا آماده‌شدن آن منتشر نمی‌شود',smart_ads_unsupported_transport:'درایور این زنجیرهٔ ادز پشتیبانی نمی‌شود؛ خروجی مستقیم تولید نشده است',smart_ads_ambiguous_chain:'وابستگی مسیر ادز مبهم است و نیاز به بررسی دارد',smart_bootstrap_no_ip:'این ترکیب برای DNS تونلی به حداقل یک سرور IP مستقل نیاز دارد',smart_requires_source_routing:'این عضو به مسیریابی اختصاصی نیاز دارد و با قالب ثابت اسمارت قابل ترکیب نیست',smart_dependency_cycle:'وابستگی چرخشی میان خروجی‌های یک عضو وجود دارد',smart_core_validated:'هسته: پذیرفته شد',smart_core_rejected:'JSON ترکیبی توسط هسته رد شد؛ بررسی مجدد خودکار',smart_member_blob_unavailable:'دادهٔ یک عضو در حال بازیابی است',smart_profile_size_limit:'حجم این ترکیب از ظرفیت بررسی هسته بیشتر است',smart_missing_dependency:'وابستگی یکی از اعضا ناقص است'};
 async function refresh(){if(busy)return;busy=true;try{
 const requestedPage=page;const s=await api('/api'+base+'?page='+requestedPage);if(requestedPage!==page)return;$('#feature-status').textContent=s.availability?.available?'':(s.availability?.message||'');$('#smart-copy-all').disabled=false;$('#smart-copy-single').disabled=false;
 $('#smart-all-url').href=s.export_url;$('#smart-all-url').textContent=s.export_url;$('#smart-single-url').href=s.single_export_url;$('#smart-single-url').textContent=s.single_export_url;
 $('#smart-worker').textContent=!s.settings.enabled?'غیرفعال':s.worker.status==='running'?'ساخت خودکار فعال':'در انتظار سرویس اسمارت';
 $('#smart-counts').textContent=(base==='/smart-ads'?'اعضای درخواستی مسیر اصلی (به‌علاوه یک ادز): ':'اعضای درخواستی: ')+fa.format(s.settings.count)+(base==='/smart-ads'?' · اسپلش ادز واجد شرایط: ':' · ورودی یکتای واجد شرایط: ')+fa.format(s.stats.eligible)+(base==='/smart-ads'?' · مسیر اصلی واجد شرایط: '+fa.format(s.stats.main_eligible):'')+' · منابع با تأیید عمومی و اینستاگرام: '+fa.format(s.stats.sources)+' · آماده: '+fa.format(s.stats.ready)+' · منتظر خروجی معتبر: '+fa.format(s.stats.warming)+' · نیازمند بررسی: '+fa.format(s.stats.blocked);
 $('#smart-page').textContent=fa.format(page+1);$('#smart-prev').disabled=page===0;$('#smart-next').disabled=s.entries.length<100;
 const sig=JSON.stringify(s.entries);if(sig===signature)return;signature=sig;
 const host=$('#smart-list'),cached=new Map([...host.querySelectorAll('pre')].map(p=>[p.dataset.hash,{text:p.textContent,hidden:p.hidden}]));
 host.innerHTML=s.entries.length?s.entries.map(v=>`<article class="output-row"><div class="output-summary"><span class="protocol">${base==='/smart-ads'?'SMART ADS':base==='/smart-splash'?'SPLASH':'SMART'}</span><div class="output-meta"><b>اسمارت · ${esc(v.id.slice(0,10))}</b><span>${fa.format(v.count)} عضو · ${fa.format(v.sources)} منبع</span><span>${v.available?(v.refreshing?'آمادهٔ استفاده · به‌روزرسانی در پس‌زمینه':'هسته: پذیرفته شد'):v.state==='blocked'?esc(codes[v.code]||v.code):'در حال آماده‌سازی'}</span><span>اعتبار کش: ${countdown(v.expires)}</span></div>${v.available?`<div class="output-actions"><button class="secondary" data-smart-view="${v.id}">نمایش</button><button class="secondary" data-smart-copy="${v.id}">کپی JSON</button></div>`:''}</div><pre data-hash="${v.hash}" dir="ltr" hidden></pre></article>`).join(''):empty('اسمارت هنوز آماده نیست',s.availability?.message||'ساخت خودکار از کانفیگ‌های پذیرفته‌شدهٔ منابع انجام می‌شود.','output');
 for(const p of host.querySelectorAll('pre')){const v=cached.get(p.dataset.hash);if(v){p.textContent=v.text;p.hidden=v.hidden}}
 }catch(e){$('#feature-status').textContent=e.message+' تلاش مجدد خودکار انجام می‌شود.';return false}finally{busy=false}}
 for(const [button,link]of[['smart-copy-all','smart-all-url'],['smart-copy-single','smart-single-url']])$('#'+button).addEventListener('click',async()=>{if(!$('#'+link).getAttribute('href'))return;try{await navigator.clipboard.writeText($('#'+link).href);toast('لینک کپی شد.')}catch{toast('لینک را انتخاب و کپی کنید.')}});
 $('#smart-prev').addEventListener('click',()=>{if(page>0){page--;signature='';refresh()}});$('#smart-next').addEventListener('click',()=>{page++;signature='';refresh()});
 $('#smart-list').addEventListener('click',async e=>{const button=e.target.closest('[data-smart-view],[data-smart-copy]');if(!button)return;button.disabled=true;try{
 const pre=button.closest('article').querySelector('pre');if(button.dataset.smartView&&!pre.hidden){pre.hidden=true;return}
 const {hash,text}=await verifiedJSON(base+'/json/'+(button.dataset.smartView||button.dataset.smartCopy));if(button.dataset.smartCopy){await navigator.clipboard.writeText(text);toast('JSON اسمارت کپی شد.')}else{
 const id=button.dataset.smartView;
 // A periodic refresh can replace the row while its JSON is being fetched.
 const current=$('#smart-list').querySelector('[data-smart-view="'+CSS.escape(id)+'"]')?.closest('article')?.querySelector('pre');
 if(!current||current.dataset.hash!==hash)throw new Error('این ردیف تغییر کرده است؛ دوباره نمایش را بزنید.');
 current.textContent=text;current.dataset.hash=hash;current.hidden=false
}
 }catch(e){toast(e.message)}finally{button.disabled=false}});
 return {tasks:[refresh],render(){},filters(){return {}}};
}
