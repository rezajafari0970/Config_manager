import {api,toast} from './core.js';

// Shared form driver. Polling preserves drafts; conflicts require explicit
// resubmission with a refreshed revision, never a silent overwrite.
export function bindSettings({form,path,read,write,onData=()=>{}}){
 const controls=[...form.querySelectorAll('input,button[type="submit"],button:not([type])')];
 const status=document.createElement('p');status.className='hint';status.setAttribute('role','status');form.prepend(status);
 let loaded=false,dirty=false,saving=false,pending,revision,lastRead=0;
 const lock=()=>{for(const c of controls)c.disabled=!loaded||saving;form.setAttribute('aria-busy',String(!loaded||saving))};
 const note=text=>{status.textContent=text};
 note('در حال دریافت تنظیمات…');lock();
 form.addEventListener('input',()=>{dirty=true});form.addEventListener('change',()=>{dirty=true});
 async function load(force=false){
  if(pending)return pending;
  if(saving&&!force)return;
  if(!force&&loaded&&Date.now()-lastRead<5000)return;
  pending=(async()=>{
   try{
    const data=await api(path);if(!dirty||force||!loaded)revision=data.settings.revision;onData(data);
    if(!dirty){write(data.settings);note('تنظیمات دریافت شد.');}
    loaded=true;lastRead=Date.now();return true;
   }catch(e){note(e.message+' تلاش مجدد خودکار انجام می‌شود.');return false}
   finally{pending=null;lock()}
  })();
  return pending;
 }
 form.addEventListener('submit',async e=>{
  e.preventDefault();if(!loaded||saving)return;
  saving=true;lock();
  try{
   if(pending)await pending;
   const body=read();if(revision!==undefined)body.revision=revision;
   await api(path,{method:'POST',body});dirty=false;
   await load(true);note('تنظیمات ذخیره شد.');toast('تنظیمات ذخیره شد.');
  }catch(e){
   if(e.status===409){dirty=true;const ok=await load(true);if(ok)note('تنظیمات در جای دیگری تغییر کرده است. مقادیر شما حفظ شد؛ برای اعمال آن‌ها دوباره ذخیره کنید.');}
   else note(e.message);
  }finally{saving=false;lock()}
 });
 return load;
}
