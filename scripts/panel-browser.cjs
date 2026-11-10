'use strict';
const assert=require('assert/strict');const {chromium}=require(process.env.PLAYWRIGHT_MODULE);
const dir=process.env.PANEL_TEST_DIR,origin='http://127.0.0.1:18192',phase=process.argv[2];
(async()=>{const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});try{
 const context=await browser.newContext({viewport:{width:390,height:844}}),page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 const state=async()=>(await context.request.get(origin+'/api/state')).json();
 const wait=async fn=>{for(let n=0;n<150;n++){if(await fn())return;await new Promise(r=>setTimeout(r,100))}throw new Error('condition timed out')};
 const noOverflow=async()=>assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'horizontal overflow');
 await page.goto(origin+'/input');assert.equal(await page.locator('input[type=password]').count(),0);assert.equal((await context.request.get(origin+'/assets/diagnostics.js')).status(),200);
 if(phase==='1'){
  await page.locator('#bulk-links').fill('https://example.com/Api/Verde/trun.php\nhttps://example.com/Api/Rosa/trun.php\nhttps://example.com/Api/Broken/trun.php\nhttps://example.com/Api/Verde/trun.php');await page.locator('#bulk-interval').fill('۲ تا ۳');await page.getByRole('button',{name:'افزودن لینک‌ها',exact:true}).click();await wait(async()=>{const s=await state();return s.stats.sources===3&&s.stats.configs===2});
  const s=await state();assert.deepEqual(s.sources.map(x=>x.name).sort(),['Broken','Rosa','Verde']);assert.equal(s.sources.find(x=>x.name==='Verde').content_state,'partial');assert.equal(s.sources.find(x=>x.name==='Broken').state,'error');await noOverflow();await page.screenshot({path:dir+'/input-mobile.png',fullPage:true});
  await page.getByRole('button',{name:'باز کردن منو',exact:true}).click();await page.locator('#nav-drawer').getByRole('link',{name:/Output/}).click();
 }else{const s=await state();assert.equal(s.stats.sources,3);assert.equal(s.settings.timeout_seconds,19);await page.goto(origin+'/output')}
 await page.waitForFunction(()=>document.querySelectorAll('#output-list [data-view]').length===2);assert.equal(await page.locator('#bank-list').count(),0);await noOverflow();
 const c=await(await context.request.get(origin+'/api/conversions')).json();assert.equal(c.entries.length,2);assert.ok(c.entries.every(x=>x.core==='passed'&&x.connection==='passed'));
 const response=await context.request.get(c.export_url);assert.equal(response.status(),200);assert.equal(response.headers()['content-type'],'text/plain; charset=utf-8');assert.equal(response.headers()['content-disposition'],'inline');const lines=(await response.text()).trim().split('\n');assert.equal(lines.length,2);for(const line of lines){const j=JSON.parse(line);assert.equal(j.remarks,'@shenvpn');assert.equal(j.inbounds[0].tag,'socks');assert.equal(j.outbounds.length,4)}
 // Navigate like a user: an API GET alone cannot detect forced downloads.
 const exportPage=await context.newPage(),downloads=[];exportPage.on('download',d=>downloads.push(d));
 const inlineResponse=await exportPage.goto(c.export_url,{waitUntil:'domcontentloaded'});assert.equal(inlineResponse.status(),200);assert.equal(await exportPage.evaluate(()=>document.contentType),'text/plain');assert.equal(await exportPage.locator('body').innerText(),await response.text());assert.equal(downloads.length,0);await exportPage.close();
 const rawPage=await context.newPage(),rawDownloads=[];rawPage.on('download',d=>rawDownloads.push(d));
 assert.equal(await page.locator('#raw-all-url').getAttribute('href'),origin+'/output/raw.txt');
 const rawResponse=await rawPage.goto(origin+'/output/raw.txt');assert.equal(rawResponse.status(),200);assert.equal(rawResponse.headers()['content-disposition'],'inline');assert.equal(rawResponse.headers()['x-config-count'],'2');assert.equal(await rawPage.locator('body').innerText(),(await rawResponse.text()).replace(/\r\n/g,'\n').replace(/\r/g,'\n'));assert.equal(rawDownloads.length,0);await rawPage.close();
 // Verify the new link as a real navigation, including a full two-item round.
 assert.equal(await page.locator('#single-url').getAttribute('href'),c.single_export_url);
 const onePage=await context.newPage(),singleDownloads=[];onePage.on('download',d=>singleDownloads.push(d));
 const single=[];
 for(let i=0;i<3;i++){
  const r=await onePage.goto(c.single_export_url,{waitUntil:'domcontentloaded'});assert.equal(r.status(),200);
  assert.equal(r.headers()['content-type'],'text/plain; charset=utf-8');assert.equal(r.headers()['content-disposition'],'inline');
  const text=await r.text();assert.equal(await onePage.locator('body').innerText(),text);assert.equal(text.trim().split('\n').length,1);assert.equal(JSON.parse(text).remarks,'@shenvpn');
  single.push({hash:r.headers()['x-content-sha256'],cycle:Number(r.headers()['x-export-cycle'])});
 }
 assert.equal(singleDownloads.length,0);await onePage.close();
 assert.notEqual(single[0].hash,single[1].hash);
 if(phase==='1'){assert.equal(single[0].cycle,1);assert.equal(single[1].cycle,1);assert.equal(single[2].cycle,2);assert.equal(single[2].hash,single[0].hash)}
 else{assert.equal(single[0].cycle,2);assert.equal(single[1].cycle,3);assert.equal(single[2].cycle,3);assert.notEqual(single[1].hash,single[2].hash)}
 // Separate source exports: Verde owns two configs, Rosa one shared config.
 const sourceList=await(await context.request.get(origin+'/api/export/sources')).json();
 assert.equal(sourceList.sources.length,3);const verde=sourceList.sources.find(s=>s.name==='Verde'),rosa=sourceList.sources.find(s=>s.name==='Rosa'),broken=sourceList.sources.find(s=>s.name==='Broken');
 const own=async source=>(await(await context.request.get(source.export_url)).text()).trim().split('\n').filter(Boolean);
 const vl=await own(verde),rl=await own(rosa);assert.equal(vl.length,2);assert.equal(rl.length,1);assert.ok(vl.includes(rl[0]));assert.deepEqual(await own(broken),[]);
 await page.locator('#export-sources summary').click();await page.waitForFunction(()=>document.querySelectorAll('[data-source-panel]').length===3);
 await page.locator('#export-source-search').fill('Rosa');await page.waitForFunction(()=>document.querySelectorAll('#export-source-list article').length===1);
 const scopedPage=await context.newPage();await scopedPage.goto(origin+rosa.panel_url);await scopedPage.waitForFunction(()=>document.querySelectorAll('#output-list [data-view]').length===1);
 assert.equal(await scopedPage.locator('#jsonl-url').getAttribute('href'),rosa.export_url);assert.equal(await scopedPage.locator('#single-url').getAttribute('href'),rosa.single_export_url);
 assert.ok((await scopedPage.locator('#output-scope-title').textContent()).includes('Rosa'));await scopedPage.locator('[data-view]').first().click();await scopedPage.waitForFunction(()=>[...document.querySelectorAll('#output-list pre')].some(p=>!p.hidden));assert.equal(await scopedPage.locator('#output-list pre').innerText(),rl[0]);
 assert.ok(await scopedPage.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));await scopedPage.close();
 const turns=[];for(let i=0;i<3;i++){const r=await context.request.get(verde.single_export_url);const text=await r.text();assert.ok(vl.includes(text.trim()));turns.push({hash:r.headers()['x-content-sha256'],cycle:Number(r.headers()['x-export-cycle'])})}
 if(phase==='1'){assert.equal(turns[0].cycle,1);assert.equal(turns[1].cycle,1);assert.equal(turns[2].cycle,2)}else{assert.equal(turns[0].cycle,2);assert.equal(turns[1].cycle,3);assert.equal(turns[2].cycle,3)}
 for(let i=0;i<turns.length;i++)for(let j=i+1;j<turns.length;j++)if(turns[i].cycle===turns[j].cycle)assert.notEqual(turns[i].hash,turns[j].hash);
 const rr=await context.request.get(rosa.single_export_url);assert.equal((await rr.text()).trim(),rl[0]);assert.equal(Number(rr.headers()['x-export-cycle']),Number(phase));
 const sourceViewer=await context.newPage(),sourceDownloads=[];sourceViewer.on('download',d=>sourceDownloads.push(d));const sourceResponse=await sourceViewer.goto(verde.export_url,{waitUntil:'domcontentloaded'});assert.equal(await sourceViewer.locator('body').innerText(),await sourceResponse.text());assert.equal(sourceDownloads.length,0);await sourceViewer.close();
 await page.locator('#export-source-search').fill('');await page.waitForFunction(()=>document.querySelectorAll('#export-source-list article').length===3);
 assert.equal((await context.request.get(origin+'/api/bank')).status(),404);assert.equal((await context.request.get(origin+'/assets/bank.js')).status(),404);
 await page.locator('[data-view]').first().click();await page.waitForFunction(()=>[...document.querySelectorAll('pre')].some(p=>!p.hidden&&p.textContent.startsWith('{')));await noOverflow();await page.screenshot({path:dir+'/output-mobile.png',fullPage:true});
 await page.setViewportSize({width:1440,height:1000});await noOverflow();await page.screenshot({path:dir+'/output-desktop.png',fullPage:true});

 await require('./panel-ui-isolation.cjs')(context,origin);
 // Smart exports have a durable independent head round and stay inline.
 await page.goto(origin+'/smart');await wait(async()=>{const x=await(await context.request.get(origin+'/api/smart')).json();return x.stats.ready===2});
 assert((await page.locator('.output-panel').innerText()).includes('leastLoad'));assert(!(await page.locator('.output-panel').innerText()).includes('leastPing'));
 const smart=await(await context.request.get(origin+'/api/smart')).json();assert.equal(smart.settings.count,phase==='1'?10:50);
 await wait(async()=>await page.locator('#smart-all-url').getAttribute('href')===smart.export_url);
 assert.equal(await page.locator('#smart-all-url').getAttribute('href'),smart.export_url);
 const smartPage=await context.newPage(),smartDownloads=[];smartPage.on('download',d=>smartDownloads.push(d));
 const allSmart=await smartPage.goto(smart.export_url);assert.equal(allSmart.headers()['content-disposition'],'inline');assert.equal(await smartPage.evaluate(()=>document.contentType),'text/plain');
 const smartLines=(await allSmart.text()).trim().split('\n');assert.equal(smartLines.length,2);assert.equal(new Set(smartLines).size,2);
 for(const line of smartLines){
  const j=JSON.parse(line);assert.equal(j.remarks,'@shenvpn-smart-parallel-v6b');assert.deepEqual(Object.keys(j),['remarks','log','dns','inbounds','outbounds','routing','burstObservatory','policy','stats','metrics']);assert.deepEqual(j.log,{loglevel:'none',access:'none',error:'none',dnsLog:false});assert(j.routing.balancers[1].selector.includes(j.outbounds[0].tag));assert(j.outbounds.some(x=>x.tag==='block'&&x.protocol==='blackhole'));assert.equal(j.dns.servers.length,4);
  assert.deepEqual(j.dns.servers.map(d=>d.address),['https://8.8.8.8/dns-query','https://1.1.1.1/dns-query','https://1.1.1.1/dns-query','https://8.8.8.8/dns-query']);for(const d of j.dns.servers){assert.equal(d.timeoutMs,2500)}
  for(const d of j.dns.servers.slice(0,2)){assert.equal(d.skipFallback,true);assert.equal(d.tag,'dns-bootstrap');assert.deepEqual(d.domains,['full:vpn.example'])}
  assert.equal(j.routing.balancers.length,2);assert.equal(j.routing.rules.length,4);assert.equal(j.routing.balancers[0].fallbackTag,'block');assert.equal(j.routing.balancers[1].selector.length,1);
  const bootstrap=j.outbounds.find(x=>x.tag===j.routing.balancers[1].selector[0]);assert.equal(bootstrap.settings.address||bootstrap.settings.vnext[0].address,'192.0.2.1');assert.equal(j.outbounds.filter(x=>x.tag.startsWith('proxy-')).length,2);assert.equal(j.inbounds[0].settings.udp,true);assert.equal(j.metrics.listen,'127.0.0.1:11111');
 }

 const smartTurns=[];for(let i=0;i<3;i++){const r=await smartPage.goto(smart.single_export_url);assert.equal(r.status(),200);assert.equal(r.headers()['content-disposition'],'inline');assert.equal((await r.text()).trim().split('\n').length,1);smartTurns.push({id:r.headers()['x-smart-first-id'],cycle:Number(r.headers()['x-export-cycle'])})}
 assert.equal(smartDownloads.length,0);await smartPage.close();
 assert.deepEqual(smartTurns.map(x=>x.cycle),phase==='1'?[1,1,2]:[2,3,3]);
 for(let i=0;i<3;i++)for(let j=i+1;j<3;j++)if(smartTurns[i].cycle===smartTurns[j].cycle)assert.notEqual(smartTurns[i].id,smartTurns[j].id);
 // Re-render during the request: the viewer must update the current DOM row.
 await page.route('**/smart/json/*',async route=>{await page.evaluate(()=>{const list=document.querySelector('#smart-list');list.innerHTML=list.innerHTML});await route.continue()});
 await page.locator('[data-smart-view]').first().click();await page.waitForFunction(()=>[...document.querySelectorAll('#smart-list pre')].some(p=>!p.hidden&&p.textContent.startsWith('{')));
 await noOverflow();await page.screenshot({path:dir+'/smart-desktop.png',fullPage:true});await page.setViewportSize({width:390,height:844});await noOverflow();await page.screenshot({path:dir+'/smart-mobile.png',fullPage:true});
 if(phase==='2'){
  await page.goto(origin+'/input');
  const post=async(path,body)=>page.evaluate(async({path,body})=>{const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.querySelector('meta[name="csrf-token"]').content},body:JSON.stringify(body)});if(!r.ok)throw new Error('POST '+r.status);return r.json()},{path,body});
  const before=await state(),rosa=before.sources.find(x=>x.name==='Rosa');
  const original=before.outputs.find(x=>x.sources===2);assert.ok(original);
  await post('/api/configs/batch',{action:'delete',ids:[original.id]});
  await post('/api/sources/'+rosa.id+'/run',{revision:rosa.revision});
  await wait(async()=>{const r=(await state()).sources.find(x=>x.id===rosa.id);return r.state==='filtered'&&r.suppressed_count===1&&r.config_count===0&&r.duplicates===0});
  const row=page.locator('.source-row[data-id="'+rosa.id+'"]');await page.waitForFunction(id=>document.querySelector('.source-row[data-id="'+id+'"] .badge')?.textContent.includes('قبلاً حذف'),rosa.id);
  assert.equal(await row.locator('.badge.good').count(),0);await row.locator('summary').click();await row.getByRole('button',{name:'بازیابی و دریافت دوباره',exact:true}).click();
  await wait(async()=>{const r=(await state()).sources.find(x=>x.id===rosa.id);return r.config_count===1&&r.suppressed_count===0&&r.state==='updated'});
  await wait(async()=>(await(await context.request.get(origin+'/api/conversions')).json()).entries.length===2);
  assert.equal((await state()).stats.configs,2);await noOverflow();await page.screenshot({path:dir+'/restored-source.png',fullPage:true});
 }
 if(phase==='2'){
  const pendingState=await(await context.request.get(origin+'/api/smart-ads')).json();
  const waiting=await context.newPage();waiting.on('pageerror',e=>errors.push(e.message));
  const unavailable=await waiting.goto(pendingState.single_export_url);assert.equal(unavailable.status(),503);
  assert.equal(await waiting.evaluate(()=>document.contentType),'text/html');
  assert.ok((await waiting.locator('#status').innerText()).length>20);
  assert.ok(await waiting.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await waiting.screenshot({path:dir+'/waiting-mobile.png',fullPage:true});
  const machine=await context.request.get(pendingState.single_export_url,{headers:{Accept:'application/json'}});assert.equal(machine.status(),503);assert(machine.headers()['content-type'].startsWith('text/plain'));
  await page.goto(origin+'/input');await page.waitForSelector('[data-action=splash]');if(await page.locator('#import-panel').getAttribute('open')===null)await page.locator('#import-panel summary').click();await page.locator('#bulk-links').fill('https://example.com/Api/Splash/trun.php');await page.locator('#bulk-splash').check();await page.getByRole('button',{name:'افزودن لینک‌ها',exact:true}).click();
  const splashState=async()=>(await(await context.request.get(origin+'/api/smart-splash')).json());
  await wait(async()=>{const s=await splashState();return s.stats.ready===2&&s.entries.every(v=>v.count===2&&!v.refreshing)});const src=(await state()).sources.find(s=>s.name==='Splash');assert(src.splash);
  const row=page.locator('.source-row[data-id="'+src.id+'"]'),toggle=row.locator('[data-action=splash]');await wait(async()=>await toggle.getAttribute('aria-pressed')==='true');await noOverflow();
  await page.goto(origin+'/smart-splash');await wait(async()=>await page.locator('[data-smart-view]').count()===2);await noOverflow();
  const sp=await splashState();assert.equal(sp.settings.count,7);assert.equal((await(await context.request.get(origin+'/api/smart')).json()).stats.ready,2);
  const all=(await(await context.request.get(sp.export_url)).text()).trim().split('\n');assert.equal(all.length,2);
  for(const l of all){const j=JSON.parse(l);assert.equal(j.outbounds.filter(o=>o.tag.startsWith('proxy-')).length,2);assert(!l.includes('vpn.example'));assert(l.includes('192.0.2.71')&&l.includes('192.0.2.72'))}
  const one=await context.newPage();const downloads=[];one.on('download',d=>downloads.push(d));const oneResponse=await one.goto(sp.single_export_url);assert.equal(oneResponse.status(),200);assert.equal((await oneResponse.text()).trim().split('\n').length,1);assert.equal(downloads.length,0);await one.close();
  await page.locator('[data-smart-view]').first().click();await page.waitForFunction(()=>[...document.querySelectorAll('#smart-list pre')].some(p=>!p.hidden));await page.screenshot({path:dir+'/splash-mobile.png',fullPage:true});
  const adsState=async()=>(await(await context.request.get(origin+'/api/smart-ads')).json());
  await wait(async()=>(await adsState()).stats.ready===2);
  // The waiting page may already have recovered automatically before this step.
  if(await waiting.evaluate(()=>document.contentType!=='text/plain')){
   try{await waiting.locator('#retry').click({timeout:2000})}catch(error){if(await waiting.evaluate(()=>document.contentType!=='text/plain'))throw error}
  }
  await waiting.waitForFunction(()=>document.contentType==='text/plain');
  assert.equal(JSON.parse(await waiting.locator('body').innerText()).remarks,'@shenvpn-smart-ads');await waiting.close();
  for(const width of [390,1440]){
   await page.setViewportSize({width,height:900});await page.goto(origin+'/smart-ads');await wait(async()=>await page.locator('[data-smart-view]').count()===2);await noOverflow();
   const ad=await adsState();assert.equal(ad.settings.count,3);assert.equal(await page.locator('#smart-all-url').getAttribute('href'),ad.export_url);
   const lines=(await(await context.request.get(ad.export_url)).text()).trim().split('\n');assert.equal(lines.length,2);
   for(const l of lines){const j=JSON.parse(l);assert.equal(j.remarks,'@shenvpn-smart-ads');assert.equal(j.outbounds.filter(x=>x.tag==='ads-01').length,1);assert(j.outbounds.find(x=>x.tag==='ads-01').streamSettings.sockopt.dialerProxy==='main-chain-dispatch')}
   await page.locator('[data-smart-view]').first().click();await page.waitForFunction(()=>[...document.querySelectorAll('#smart-list pre')].some(p=>!p.hidden));
   await page.screenshot({path:dir+'/smart-ads-'+width+'.png',fullPage:true});
  }
  const ad=await adsState(),adsPage=await context.newPage(),adsDownloads=[];adsPage.on('download',d=>adsDownloads.push(d));
  const adsTurns=[];for(let i=0;i<3;i++){const r=await adsPage.goto(ad.single_export_url);assert.equal(r.status(),200);assert.equal(JSON.parse((await r.text()).trim()).remarks,'@shenvpn-smart-ads');adsTurns.push([r.headers()['x-export-cycle'],r.headers()['x-smart-profile-id']])}
  assert.equal(adsDownloads.length,0);for(let i=0;i<adsTurns.length;i++)for(let k=i+1;k<adsTurns.length;k++)if(adsTurns[i][0]===adsTurns[k][0])assert.notEqual(adsTurns[i][1],adsTurns[k][1]);assert(Number(adsTurns[2][0])>Number(adsTurns[0][0]));await adsPage.close();
  await page.goto(origin+'/input');const button=page.locator('.source-row[data-id="'+src.id+'"] [data-action=splash]');await button.click();await wait(async()=>(await splashState()).stats.ready===0);await wait(async()=>(await(await context.request.get(origin+'/api/smart')).json()).stats.ready===4);
  await wait(async()=>await button.getAttribute('aria-pressed')==='false');await button.click();await wait(async()=>(await splashState()).stats.ready===2);await wait(async()=>(await(await context.request.get(origin+'/api/smart')).json()).stats.ready===2);
 }
 await page.goto(origin+'/setting');await page.waitForFunction(()=>document.querySelector('#probe-concurrency').value!=='');await page.setViewportSize({width:390,height:844});await noOverflow();
 if(phase==='1'){await page.waitForFunction(()=>document.querySelector('#smart-ads-member-count').value!=='');await page.locator('#smart-ads-member-count').fill('۳');await page.locator('#smart-ads-settings button').click();await wait(async()=>(await(await context.request.get(origin+'/api/smart-ads')).json()).settings.count===3)}
 if(phase==='1'){await page.waitForFunction(()=>document.querySelector('#smart-splash-member-count').value!=='');await page.locator('#smart-splash-member-count').fill('۷');await page.locator('#smart-splash-settings button').click();await wait(async()=>(await(await context.request.get(origin+'/api/smart-splash')).json()).settings.count===7)}
 if(phase==='1'){await page.locator('#smart-member-count').fill('۵۰');await page.locator('#smart-settings button').click();await wait(async()=>(await(await context.request.get(origin+'/api/smart')).json()).settings.count===50);await wait(async()=>(await(await context.request.get(origin+'/api/smart')).json()).stats.ready===2)}
 if(phase==='1'){await page.locator('#retention-hours').fill('۱۰');await page.locator('#request-timeout').fill('۱۹');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await wait(async()=>(await state()).settings.timeout_seconds===19);assert.equal((await state()).settings.retention_seconds,36000);await page.locator('#probe-health_ttl_seconds').fill('۱۰');await page.locator('#probe-concurrency').fill('3');await page.locator('#probe-settings button').click();await wait(async()=>(await(await context.request.get(origin+'/api/conversions')).json()).settings.concurrency===3)}
 else{assert.equal(c.settings.concurrency,3);assert.equal(c.settings.health_ttl_seconds,36000);assert.equal(await page.locator('#probe-health_ttl_seconds').inputValue(),'10');const retainedBefore=(await state()).stats.configs;require('fs').writeFileSync(dir+'/upstream-offline','fixture');await page.locator('#retention-hours').fill('۰٫۰۰۲۵');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await wait(async()=>(await state()).stats.configs===0);const converted=await(await context.request.get(origin+'/api/conversions')).json();assert.equal(converted.entries.length,0);assert.equal(await(await context.request.get(converted.export_url)).text(),'');const emptySingle=await context.request.get(converted.single_export_url);assert.equal(emptySingle.status(),204);assert.equal(await emptySingle.text(),'');require('fs').unlinkSync(dir+'/upstream-offline');await wait(async()=>(await state()).stats.configs===retainedBefore);await wait(async()=>(await(await context.request.get(origin+'/api/conversions')).json()).entries.filter(x=>x.available).length===retainedBefore)}
 await page.screenshot({path:dir+'/setting-mobile.png',fullPage:true});assert.deepEqual(errors,[]);console.log(JSON.stringify({phase,passed:true,browser:await browser.version(),scope:'synthetic UI only; core matrix is separate'}));
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
