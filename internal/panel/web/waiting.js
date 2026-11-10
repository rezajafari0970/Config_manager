import {poll} from './poll.js';
const status=document.querySelector('#status'),counts=document.querySelector('#counts'),check=document.querySelector('#check'),button=document.querySelector('#retry');
const url=new URL('status.json',location.href);
const fa=new Intl.NumberFormat('fa-IR');
const refresh=poll(async()=>{
 button.disabled=true;
 try{
  const r=await fetch(url,{cache:'no-store',headers:{Accept:'application/json'},signal:AbortSignal.timeout(10000)});
  if(!r.ok)throw new Error('status');
  const s=await r.json();
  if(s.available){location.reload();return}
  status.textContent=s.message;
  counts.textContent=`ورودی فعال: ${fa.format(s.inputs)} · بررسی ناموفق: ${fa.format(s.failed)} · صف بررسی: ${fa.format(s.pending)}`;
  check.textContent='وضعیت بررسی شد؛ بررسی بعدی خودکار است.';
 }catch{check.textContent='دریافت وضعیت ممکن نشد؛ تلاش مجدد خودکار انجام می‌شود.';return false}
 finally{button.disabled=false}
},{interval:15000,maxDelay:60000});
button.addEventListener('click',refresh);
