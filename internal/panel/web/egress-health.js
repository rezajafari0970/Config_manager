import {$,api,fa,toast} from './core.js';
export function healthSettings(onSaved){
 let policy=null,dirty=false;
 $('#egress-health-minutes').oninput=()=>{dirty=true};
 $('#egress-health-save').onclick=async()=>{if(!policy)return;const b=$('#egress-health-save');b.disabled=true;try{policy=await api('/api/egress/lifecycle',{method:'POST',body:{revision:policy.revision,health_minutes:Number($('#egress-health-minutes').value)}});dirty=false;$('#egress-lifecycle-status').textContent='فاصلهٔ بررسی: '+fa.format(policy.health_minutes)+' دقیقه؛ ذخیره شد';onSaved()}catch(e){if(e.status===409){dirty=false;onSaved()}toast(e.message)}finally{b.disabled=false}};
 return {update(state){if(!dirty){policy=state.lifecycle;$('#egress-health-minutes').value=policy.health_minutes}$('#egress-coverage').textContent=fa.format(state.retained)+' کانفیگ پذیرفته‌شده در نگهداری مستقل · '+fa.format(state.coverage_missing)+' ورودی جاری بدون ثبت در صف';}};
}
