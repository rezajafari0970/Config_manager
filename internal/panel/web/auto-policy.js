import {api} from './core.js';
// Serialize user changes; polling cannot overwrite unsaved intent. Retry conflicts
// against a fresh revision, with bounded network retries and visible failure.
export function autoPolicy({url,readPolicy,value,body,onPolicy,onStatus,onSaved}){
 let policy=null,dirty=false,saving=false,timer=null,revision=0,failures=0;
 function schedule(ms=500){clearTimeout(timer);timer=setTimeout(save,ms)}
 async function save(){
  if(!dirty||saving||!policy)return;
  const change=revision,wanted=value();saving=true;onStatus('در حال ذخیرهٔ خودکار…');
  try{
   const next=await api(url,{method:'POST',body:{revision:policy.revision,...body(wanted)}});
   policy=next;failures=0;if(change===revision)dirty=false;
   onStatus(dirty?'تغییر تازه در صف ذخیره':'ذخیره شد');onSaved();
  }catch(e){
   failures++;if(e.status===409){try{policy=await readPolicy()}catch{}}
   onStatus(failures<4?'ذخیره ناموفق؛ تلاش خودکار دوباره…':'ذخیره نشد؛ تغییرات هنوز اعمال نشده‌اند');
  }finally{saving=false;if(dirty&&failures<4)schedule(Math.min(4000,500*2**failures))}
 }
 return {change(){dirty=true;revision++;failures=0;onStatus('در انتظار ذخیرهٔ خودکار…');schedule()},update(next){if(!dirty&&!saving){policy=next;onPolicy(next)}}};
}
