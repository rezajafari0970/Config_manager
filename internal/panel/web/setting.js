import {clearAll} from './actions.js';
import {$,fa,digit} from './core.js';
import {bindSettings} from './settings-form.js';
export default function Setting(ctx){
 const number=id=>{const n=Number(digit($('#'+id).value));if(!Number.isSafeInteger(n)||n<1)throw new Error('مقادیر باید عدد صحیح مثبت باشند.');return n};
 const tasks=[];
 for(const prefix of ['smart','smart-splash','smart-ads'])tasks.push(bindSettings({
  form:$('#'+prefix+'-settings'),path:'/api/'+prefix+'/settings',
  read:()=>({enabled:$('#'+prefix+'-enabled').checked,count:number(prefix+'-member-count')}),
  write:s=>{$('#'+prefix+'-enabled').checked=s.enabled;$('#'+prefix+'-member-count').value=s.count}
 }));
 const fields=['concurrency','timeout_seconds','retest_seconds','health_ttl_seconds'];
 tasks.push(bindSettings({form:$('#probe-settings'),path:'/api/conversion/settings',
  read:()=>Object.fromEntries([['enabled',$('#probe-enabled').checked],...fields.map(k=>[k,number('probe-'+k)])]),
  write:s=>{$('#probe-enabled').checked=s.enabled;for(const k of fields)$('#probe-'+k).value=s[k]},
  onData:s=>{$('#probe-status').textContent=s.worker.status==='running'?'تبدیل خودکار فعال است':'در انتظار آماده‌شدن سرویس تبدیل'}
 }));
 tasks.push(bindSettings({form:$('#settings-form'),path:'/api/settings',
  read:()=>{const retention=Number(digit($('#retention-hours').value))*3600;if(!Number.isSafeInteger(retention)||retention<1)throw new Error('مدت نگهداری معتبر وارد کنید.');return {retention_seconds:retention,concurrency:number('concurrency'),timeout_seconds:number('request-timeout')}},
  write:s=>{$('#retention-hours').value=s.retention_seconds/3600;$('#concurrency').value=s.concurrency;$('#request-timeout').value=s.timeout_seconds},
  onData:s=>{$('#effective-concurrency').textContent='ظرفیت فعلی: '+fa.format(s.capacity)}
 }));
 $('#settings-clear-configs').addEventListener('click',()=>clearAll('configs',ctx));
 $('#settings-clear-sources').addEventListener('click',()=>clearAll('sources',ctx));
 return {tasks,render(){},filters(){return {}}};
}
