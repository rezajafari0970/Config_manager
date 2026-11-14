import {$,api,esc,fa} from './core.js';
import {autoPolicy} from './auto-policy.js';
export function countryFilter(onSaved){
 let selected=new Set(),latest=null;
 const names=new Intl.DisplayNames(['fa'],{type:'region'}),label=c=>names.of(c)||c;
 const save=autoPolicy({url:'/api/egress/countries',readPolicy:async()=>(await api('/api/egress?brief=1')).country.policy,value:()=>[...selected],body:countries=>({countries}),onPolicy:p=>{selected=new Set(p.countries)},onStatus:s=>{$('#country-saved-count').textContent=s},onSaved});
 function draw(){if(!latest)return;const query=$('#country-search').value.trim().toLowerCase();
  $('#country-options').innerHTML=latest.codes.filter(c=>!query||(c+' '+label(c)).toLowerCase().includes(query)).sort((a,b)=>(latest.counts[b]||0)-(latest.counts[a]||0)||label(a).localeCompare(label(b),'fa')).map(c=>`<label class="country-option"><input type="checkbox" value="${c}" ${selected.has(c)?'checked':''}> ${esc(label(c))} <small dir="ltr">${c}</small> · ${fa.format(latest.counts[c]||0)}</label>`).join('');
  $('#country-selection').textContent=fa.format(selected.size)+' کشور انتخاب شده؛ ذخیرهٔ خودکار';
  $('#country-select-all').checked=selected.size===latest.codes.length;
  $('#country-select-all').indeterminate=selected.size>0&&selected.size<latest.codes.length;
 }
 $('#country-options').onchange=e=>{if(e.target.type!=='checkbox')return;e.target.checked?selected.add(e.target.value):selected.delete(e.target.value);save.change();draw()};
 $('#country-search').oninput=draw;
 $('#country-select-all').onchange=e=>{selected=new Set(e.target.checked?latest?.codes||[]:[]);save.change();draw()};
 return {label,update(data){latest=data;save.update(data.policy);$('#country-summary').textContent=fa.format(data.published)+' قابل انتشار · '+fa.format(data.known)+' با کشور مشخص · '+fa.format(data.shared)+' چندکشوری · '+fa.format(data.unknown)+' با کشور نامشخص';draw()}};
}
