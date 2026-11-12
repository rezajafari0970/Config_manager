import {$,api,esc,fa,toast} from './core.js';
export function countryFilter(onSaved){
 let policy=null,dirty=false,selected=new Set(),latest=null;
 const names=new Intl.DisplayNames(['fa'],{type:'region'});
 const label=c=>names.of(c)||c;
 function draw(){if(!latest)return;const query=$('#country-search').value.trim().toLowerCase();
  $('#country-options').innerHTML=latest.codes.filter(c=>!query||(c+' '+label(c)).toLowerCase().includes(query)).sort((a,b)=>(latest.counts[b]||0)-(latest.counts[a]||0)||label(a).localeCompare(label(b),'fa')).map(c=>`<label class="country-option"><input type="checkbox" value="${c}" ${selected.has(c)?'checked':''}> ${esc(label(c))} <small dir="ltr">${c}</small> · ${fa.format(latest.counts[c]||0)}</label>`).join('');
  $('#country-selection').textContent=fa.format(selected.size)+' کشور انتخاب شده'+(dirty?' · تغییرات ذخیره نشده':'');
 }
 $('#country-options').onchange=e=>{if(e.target.type!=='checkbox')return;e.target.checked?selected.add(e.target.value):selected.delete(e.target.value);dirty=true;draw()};
 $('#country-search').oninput=draw;
 $('#country-all').onclick=()=>{selected=new Set(latest?.codes||[]);dirty=true;draw()};
 $('#country-none').onclick=()=>{selected.clear();dirty=true;draw()};
 $('#country-save').onclick=async()=>{if(!policy)return;const b=$('#country-save');b.disabled=true;try{policy=await api('/api/egress/countries',{method:'POST',body:{revision:policy.revision,countries:[...selected]}});dirty=false;$('#country-saved-count').textContent=fa.format(policy.published)+' کانفیگ پس از ذخیرهٔ فیلتر قابل انتشار بود';toast('فیلتر ذخیره شد؛ '+fa.format(policy.published)+' کانفیگ قابل انتشار');onSaved()}catch(e){if(e.status===409){dirty=false;onSaved();toast('انتخاب در جای دیگری تغییر کرده؛ وضعیت تازه را بررسی و دوباره انتخاب کنید')}else toast(e.message)}finally{b.disabled=false;draw()}};
 return {label,update(data){latest=data;if(!dirty){policy=data.policy;selected=new Set(policy.countries)}$('#country-summary').textContent=fa.format(data.published)+' قابل انتشار · '+fa.format(data.known)+' با کشور مشخص · '+fa.format(data.shared)+' چندکشوری · '+fa.format(data.unknown)+' با کشور نامشخص';draw()}};
}
