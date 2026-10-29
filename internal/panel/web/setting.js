import {clearAll} from './actions.js';
import {$,fa,clock,esc,icon,digit,interval,dateTime,countdown,toast,api,showResult,empty} from './core.js';
export default function Setting(ctx){
let loaded=false;
$('#settings-clear-configs').addEventListener('click',()=>clearAll('configs',ctx));
$('#settings-clear-sources').addEventListener('click',()=>clearAll('sources',ctx));
$('#settings-form')?.addEventListener('submit',async e=>{e.preventDefault();const button=e.submitter;button.disabled=true;try{const retention=Number(digit($('#retention-hours').value))*3600;const concurrency=Number(digit($('#concurrency').value));const timeout=Number(digit($('#request-timeout').value));if(!Number.isSafeInteger(retention)||retention<=0||!Number.isSafeInteger(concurrency)||concurrency<=0||!Number.isSafeInteger(timeout)||timeout<=0)throw new Error('مقادیر مثبت و معتبر وارد کنید.');await api('/api/settings',{method:'POST',body:{retention_seconds:retention,concurrency,timeout_seconds:timeout}});$('#settings-result').textContent='تنظیمات ذخیره شد.';await ctx.refresh();toast('تنظیمات اعمال شد.')}catch(error){toast(error.message)}finally{button.disabled=false}});
return {render(state){$('#effective-concurrency').textContent='ظرفیت فعلی: '+fa.format(state.worker.capacity||0);if(loaded)return;$('#retention-hours').value=state.settings.retention_seconds/3600;$('#concurrency').value=state.settings.concurrency;$('#request-timeout').value=state.settings.timeout_seconds;loaded=true},filters(){return {}}};
}
