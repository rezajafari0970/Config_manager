import assert from 'node:assert/strict';
import fs from 'node:fs';
let src=fs.readFileSync(new URL('../internal/panel/web/auto-policy.js',import.meta.url),'utf8').replace("import {api} from './core.js';",'').replace('export function','function');
function harness(api){let seq=0,timers=new Map(),wanted=['TR'],display=null,status='',saved=0;
 const make=new Function('api','setTimeout','clearTimeout',src+';return autoPolicy;')(api,fn=>{timers.set(++seq,fn);return seq},id=>timers.delete(id));
 const ctl=make({url:'/policy',readPolicy:async()=>({revision:9}),value:()=>wanted.slice(),body:countries=>({countries}),onPolicy:p=>{display=p},onStatus:s=>{status=s},onSaved:()=>{saved++}});
 ctl.update({revision:1,countries:['US']});
 return {ctl,set:v=>{wanted=v;ctl.change()},run:()=>{const [k,f]=timers.entries().next().value;timers.delete(k);return f()},get pending(){return timers.size},get display(){return display},get status(){return status},get saved(){return saved}};
}
let release,calls=[];
const h=harness(async(u,{body})=>{calls.push(body);if(calls.length===1)return new Promise(r=>release=r);return {revision:3,countries:body.countries}});
h.set(['TR']);h.set(['SE']);assert.equal(h.pending,1);
const first=h.run();h.set(['DE']);h.ctl.update({revision:100,countries:['IR']});assert.equal(h.display.revision,1);
release({revision:2,countries:['SE']});await first;await h.run();assert.deepEqual(calls,[{revision:1,countries:['SE']},{revision:2,countries:['DE']}]);assert.equal(h.pending,0);assert.equal(h.status,'ذخیره شد');
let fail=0;const f=harness(async()=>{fail++;throw Object.assign(new Error('conflict'),{status:409})});f.set(['TR']);while(f.pending)await f.run();assert.equal(fail,4);assert.equal(f.saved,0);assert(f.status.includes('ذخیره نشد'));
f.ctl.update({revision:20,countries:['US']});assert.equal(f.display.revision,1);f.set(['GB']);assert.equal(f.pending,1);
console.log('automatic policy save: serialized latest intent, polling isolation, bounded retry and visible failure passed');
