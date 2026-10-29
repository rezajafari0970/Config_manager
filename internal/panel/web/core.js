let toastTimer;
export const $=s=>document.querySelector(s);
export const csrf=$('meta[name="csrf-token"]').content;
export const fa=new Intl.NumberFormat('fa-IR');
export const clock=new Intl.DateTimeFormat('fa-IR',{hour:'2-digit',minute:'2-digit',second:'2-digit'});
export const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const icon=name=>`<svg aria-hidden="true"><use href="#i-${name}"></use></svg>`;
export const digit=v=>String(v).replace(/[۰-۹]/g,c=>String(c.charCodeAt(0)-1776)).replace(/[٠-٩]/g,c=>String(c.charCodeAt(0)-1632)).replace(/٫/g,'.');
export const unitLabel=unit=>({minutes:'دقیقه',seconds:'ثانیه',hours:'ساعت'}[unit]||unit);
export const interval=s=>fa.format(s.min)+(s.max!==s.min?' تا '+fa.format(s.max):'')+' '+unitLabel(s.unit);
export const dateTime=n=>n?clock.format(new Date(n)):'هنوز دریافت نشده';
export const countdown=n=>{const sec=Math.max(0,Math.ceil((n-Date.now())/1000));if(sec<60)return fa.format(sec)+' ثانیه';if(sec<3600)return fa.format(Math.ceil(sec/60))+' دقیقه';return fa.format(Math.ceil(sec/3600))+' ساعت'};
export function toast(text){$('#toast').textContent=text;$('#toast').classList.remove('hidden');clearTimeout(toastTimer);toastTimer=setTimeout(()=>$('#toast').classList.add('hidden'),4500)}
export async function api(path,options={}){const headers=new Headers(options.headers||{});if(options.method==='POST'){headers.set('X-CSRF-Token',csrf);if(typeof options.body!=='string'&&options.body!==undefined){headers.set('Content-Type','application/json');options.body=JSON.stringify(options.body)}}const response=await fetch(path,{...options,headers,credentials:'same-origin',cache:'no-store'});let data;try{data=await response.json()}catch{throw new Error('پاسخ سرور دریافت نشد؛ دوباره تلاش کنید.')}if(!response.ok)throw new Error(data.error||'عملیات انجام نشد.');return data}
export function showResult(selector,text,error=false){const box=$(selector);box.textContent=text;box.classList.remove('hidden');box.classList.toggle('error',error)}
export function empty(title,description,symbol){return `<div class="empty">${icon(symbol)}<h3>${esc(title)}</h3><p>${esc(description)}</p></div>`}
