import {$,api,fa} from './core.js';
import {autoPolicy} from './auto-policy.js';
export function healthSettings(onSaved){
 const save=autoPolicy({url:'/api/egress/lifecycle',readPolicy:async()=>(await api('/api/egress?brief=1')).lifecycle,value:()=>Number($('#egress-health-minutes').value),body:health_minutes=>({health_minutes}),onPolicy:p=>{if(document.activeElement!==$('#egress-health-minutes'))$('#egress-health-minutes').value=p.health_minutes},onStatus:s=>{$('#egress-lifecycle-status').textContent=s},onSaved});
 $('#egress-health-minutes').oninput=()=>{const n=Number($('#egress-health-minutes').value);if(Number.isInteger(n)&&n>=1&&n<=60)save.change();else $('#egress-lifecycle-status').textContent='فاصله باید عدد صحیح بین ۱ و ۶۰ باشد'};
 return {update(state){save.update(state.lifecycle);$('#egress-coverage').textContent=fa.format(state.retained)+' رزیدنتال در نگهداری خودکار · '+fa.format(state.coverage_missing)+' هویت جاری بدون ثبت در صف';}};
}
