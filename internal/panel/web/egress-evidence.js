import {api} from './core.js';
// Bounded receipt cache; only the matching row version receives a response.
export function egressEvidence(container){
 const cache=new Map(),pending=new Map(),queue=[];let active=0;
 const key=d=>[d.dataset.evidence,d.dataset.tested,d.dataset.revision].join(':');
 function schedule(work){return new Promise((resolve,reject)=>{if(queue.length>=64){reject(new Error("درخواست‌های شواهد در حال پردازش‌اند؛ دوباره تلاش می‌شود."));return}queue.push({work,resolve,reject});drain()})}
 function drain(){while(active<2&&queue.length){const t=queue.shift();active++;Promise.resolve().then(t.work).then(t.resolve,t.reject).finally(()=>{active--;drain()})}}
 async function show(d){
  if(!d.open)return;const k=key(d),hash=d.dataset.evidence;
  try{
   if(!cache.has(k)){
    if(!pending.has(k))pending.set(k,schedule(async()=>{
     const state=await api('/api/egress?view=diagnostics&hash='+encodeURIComponent(hash));
     const x=state.entries.find(x=>x.hash===hash&&String(x.tested)===d.dataset.tested&&String(x.revision)===d.dataset.revision);
     if(!x)throw new Error('شواهد تغییر کرده‌اند؛ در به‌روزرسانی بعدی دوباره دریافت می‌شوند.');
     const text=JSON.stringify({scope:x.decision?.scope,report_version:x.report?.version,coverage_complete:x.decision?.coverage_complete,evidence_profile:x.decision?.evidence_profile,services:x.decision?.services,general_payload_status:x.report?.general_health,code:x.report?.code,checks:x.report?.checks,control_sources:x.decision?.control_sources,attempts:x.report?.attempts,general_payload:x.report?.health,attempt_evidence:x.report?.evidence,samples:x.report?.samples},null,2);
     cache.set(k,text);while(cache.size>32)cache.delete(cache.keys().next().value);return text;
    }).finally(()=>pending.delete(k)));
    const text=await pending.get(k);if(d.isConnected&&key(d)===k)d.querySelector('pre').textContent=text;
   }else d.querySelector('pre').textContent=cache.get(k);
  }catch(e){if(d.isConnected&&key(d)===k)d.querySelector('pre').textContent=e.message||'دریافت شواهد ناموفق بود.'}
 }
 container.addEventListener('toggle',e=>{if(e.target.matches('details[data-evidence]')&&e.target.open)show(e.target)},true);
 return {
  capture:()=>new Set([...container.querySelectorAll('details[data-evidence][open]')].map(d=>d.dataset.evidence)),
  restore(open){for(const d of container.querySelectorAll('details[data-evidence]'))if(open.has(d.dataset.evidence)){d.open=true;show(d)}}
 };
}
