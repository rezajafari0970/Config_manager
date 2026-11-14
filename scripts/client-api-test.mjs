import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
globalThis.document={querySelector:()=>({content:'synthetic-csrf'})};
const source=await readFile(new URL('../internal/panel/web/core.js',import.meta.url),'utf8');
const {api}=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
let timer,cleared=0,delay;
globalThis.setTimeout=(callback,ms)=>{timer=callback;delay=ms;return 42};
globalThis.clearTimeout=id=>{assert.equal(id,42);cleared++};
const options={method:'POST',body:{test:1}};
globalThis.fetch=async(path,request)=>{
 assert.equal(request.headers.get('X-CSRF-Token'),'synthetic-csrf');
 assert.equal(request.body,'{"test":1}');assert.equal(request.cache,'no-store');
 return {ok:true,json:async()=>({ok:1})};
};
assert.deepEqual(await api('/api/test',options),{ok:1});
assert.deepEqual(options.body,{test:1});assert.equal(delay,20000);
globalThis.fetch=async(_,{signal})=>new Promise((_,reject)=>signal.addEventListener('abort',()=>reject(new Error('aborted')),{once:true}));
const pending=api('/api/test');timer();
await assert.rejects(pending,/مهلت ارتباط/);
globalThis.fetch=async()=>({ok:true,json:async()=>({recovered:true})});
assert.deepEqual(await api('/api/test'),{recovered:true});
globalThis.fetch=async()=>({ok:false,json:async()=>({error:'fixture rejection'})});
await assert.rejects(api('/api/test'),/fixture rejection/);
assert.equal(cleared,4);
globalThis.fetch=async(_,request)=>{assert.equal('timeoutMs' in request,false);return {ok:true,json:async()=>true}};
for(const [timeoutMs,want] of [[120000,120000],[999999,120000],[0,1000],[NaN,20000]]){await api('/api/rescan',{timeoutMs});assert.equal(delay,want)}
assert.equal(cleared,8);
console.log('client API: bounded timeout, recovery, CSRF, immutable options and server errors passed');
