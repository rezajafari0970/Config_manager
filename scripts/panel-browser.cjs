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
 const c=await(await context.request.get(origin+'/api/conversions')).json();assert.equal(c.entries.length,2);assert.ok(c.entries.every(x=>x.core==='passed'&&x.connection==='not_run'));
 const response=await context.request.get(c.export_url);assert.equal(response.status(),200);assert.equal(response.headers()['content-type'],'text/plain; charset=utf-8');assert.equal(response.headers()['content-disposition'],'inline');const lines=(await response.text()).trim().split('\n');assert.equal(lines.length,2);for(const line of lines){const j=JSON.parse(line);assert.equal(j.remarks,'@shenvpn');assert.equal(j.inbounds[0].tag,'socks');assert.equal(j.outbounds.length,4)}
 // Navigate like a user: an API GET alone cannot detect forced downloads.
 const exportPage=await context.newPage(),downloads=[];exportPage.on('download',d=>downloads.push(d));
 const inlineResponse=await exportPage.goto(c.export_url,{waitUntil:'domcontentloaded'});assert.equal(inlineResponse.status(),200);assert.equal(await exportPage.evaluate(()=>document.contentType),'text/plain');assert.equal(await exportPage.locator('body').innerText(),await response.text());assert.equal(downloads.length,0);await exportPage.close();
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
 await page.goto(origin+'/setting');await page.waitForFunction(()=>document.querySelector('#probe-concurrency').value!=='');await page.setViewportSize({width:390,height:844});await noOverflow();
 if(phase==='1'){await page.locator('#request-timeout').fill('۱۹');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await wait(async()=>(await state()).settings.timeout_seconds===19);await page.locator('#probe-concurrency').fill('3');await page.locator('#probe-settings button').click();await wait(async()=>(await(await context.request.get(origin+'/api/conversions')).json()).settings.concurrency===3)}
 else{assert.equal(c.settings.concurrency,3);await page.locator('#retention-hours').fill('۰٫۰۰۲۵');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await wait(async()=>(await state()).stats.configs===0);const converted=await(await context.request.get(origin+'/api/conversions')).json();assert.equal(converted.entries.length,0);assert.equal(await(await context.request.get(converted.export_url)).text(),'');const emptySingle=await context.request.get(converted.single_export_url);assert.equal(emptySingle.status(),204);assert.equal(await emptySingle.text(),'')}
 await page.screenshot({path:dir+'/setting-mobile.png',fullPage:true});assert.deepEqual(errors,[]);console.log(JSON.stringify({phase,passed:true,browser:await browser.version(),scope:'synthetic UI only; core matrix is separate'}));
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
