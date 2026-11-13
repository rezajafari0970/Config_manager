'use strict';
const assert=require('assert/strict'),{chromium}=require(process.env.PLAYWRIGHT_MODULE);
(async()=>{const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
try{const page=await browser.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
for(const width of [390,1440]){
 await page.setViewportSize({width,height:900});await page.goto(process.env.PANEL_TEST_ORIGIN+'/egress');await page.waitForFunction(()=>document.querySelector('#egress-progress').textContent.includes('رزیدنتال'));
 assert.equal(await page.locator('#egress-links a').count(),4);await page.screenshot({path:process.env.PANEL_TEST_DIR+'/egress-before-'+width+'.png',fullPage:true});if(!await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth))console.error(await page.evaluate(()=>[...document.querySelectorAll('*')].filter(e=>e.getBoundingClientRect().width>innerWidth).map(e=>({tag:e.tagName,cls:e.className,id:e.id,width:e.getBoundingClientRect().width})).slice(0,12)));assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 await page.locator('#egress-diagnostics').check();await page.waitForFunction(()=>document.querySelectorAll('#egress-list article').length>0);
 await page.locator('#egress-list details').first().locator('summary').click();await page.waitForFunction(()=>document.querySelector('#egress-list details[open] pre')?.textContent.trim().startsWith('{'));await page.waitForTimeout(3500);assert.equal(await page.locator('#egress-list details[open]').count(),1);assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 const state=await(await page.request.get(process.env.PANEL_TEST_ORIGIN+'/api/egress?view=diagnostics')).json();assert(state.total>0);assert(state.entries.every(x=>x.hash.length===64));
 const a=state.accounting,p=a.publication;assert.equal(a.at,state.now);assert.equal(a.live_versions,state.total);assert.equal(Object.values(a.queue).reduce((x,y)=>x+y,0),a.live_versions);assert.equal(a.awaiting_first+a.assessed_versions,a.live_versions);assert.equal(a.current_live_versions+a.archive_versions,a.live_versions);assert.equal(p.retained,p.eligible+p.awaiting_proof);assert.equal(p.eligible,p.country_unknown+p.country_filtered+p.published);assert.equal(p.published,state.country.published);assert.equal(a.coverage_missing,0);assert.equal(a.staged_coverage_missing,0);assert((await page.locator('#egress-inventory').textContent()).includes('نسخه'));assert((await page.locator('#egress-output-counts').textContent()).includes('هویت'));assert(state.throughput.unique_definitive_10m<=state.throughput.unique_10m);
 await page.screenshot({path:process.env.PANEL_TEST_DIR+'/egress-'+width+'.png',fullPage:true});
}
if(process.env.PANEL_COUNTRY_TEST==='1'){
 await page.goto(process.env.PANEL_TEST_ORIGIN+'/egress');await page.waitForFunction(()=>document.querySelectorAll('#country-options input').length===250);
 assert.equal(await page.locator('#egress-health-minutes').inputValue(),'15');
 await page.locator('#egress-health-minutes').fill('5');await page.locator('#egress-health-save').click();await page.waitForFunction(()=>document.querySelector('#egress-lifecycle-status').textContent.includes('ذخیره شد'));
 const original=(await(await page.request.get(process.env.PANEL_TEST_ORIGIN+'/api/egress')).json()).country.policy;
 await page.locator('#country-none').click();await page.locator('#country-save').click();
 await page.waitForFunction(()=>!document.querySelector('#country-selection').textContent.includes('ذخیره نشده'));
 assert((await page.locator('#country-saved-count').textContent()).includes('کانفیگ'));
 await page.reload();await page.waitForFunction(()=>document.querySelectorAll('#country-options input').length===250);
 assert.equal(await page.locator('#country-options input:checked').count(),0);
 assert((await page.locator('body').textContent()).includes('وجود حداقل یک کشور انتخاب‌شده'));
 const state=await(await page.request.get(process.env.PANEL_TEST_ORIGIN+'/api/egress')).json();assert.equal(state.country.match_mode,'any');
 for(const url of Object.values(state.links))assert.equal((await page.request.get(url)).status(),503);
 await page.locator('#country-options input[value="TR"]').check();
 await page.waitForTimeout(3500);assert.equal(await page.locator('#country-options input:checked').count(),1);
 await page.locator('#country-save').click();await page.waitForFunction(()=>!document.querySelector('#country-selection').textContent.includes('ذخیره نشده'));
 const after=await(await page.request.get(process.env.PANEL_TEST_ORIGIN+'/api/egress')).json();assert.deepEqual(after.country.policy.countries,['TR']);
 const restored=await page.evaluate(async body=>{const r=await fetch('/api/egress/countries',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.querySelector('meta[name="csrf-token"]').content},body:JSON.stringify(body)});return r.status},{revision:after.country.policy.revision,countries:original.countries});assert.equal(restored,200);
}
assert.deepEqual(errors,[]);console.log(JSON.stringify({passed:true,mobile:390,desktop:1440,errors}));
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
