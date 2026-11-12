'use strict';
const assert=require('assert/strict'),{chromium}=require(process.env.PLAYWRIGHT_MODULE);
(async()=>{const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
try{const page=await browser.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
for(const width of [390,1440]){
 await page.setViewportSize({width,height:900});await page.goto(process.env.PANEL_TEST_ORIGIN+'/egress');await page.waitForFunction(()=>document.querySelector('#egress-progress').textContent.includes('رزیدنتال'));
 assert.equal(await page.locator('#egress-links a').count(),4);await page.screenshot({path:process.env.PANEL_TEST_DIR+'/egress-before-'+width+'.png',fullPage:true});if(!await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth))console.error(await page.evaluate(()=>[...document.querySelectorAll('*')].filter(e=>e.getBoundingClientRect().width>innerWidth).map(e=>({tag:e.tagName,cls:e.className,id:e.id,width:e.getBoundingClientRect().width})).slice(0,12)));assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 await page.locator('#egress-diagnostics').check();await page.waitForFunction(()=>document.querySelectorAll('#egress-list article').length>0);
 await page.locator('#egress-list details').first().locator('summary').click();assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 const state=await(await page.request.get(process.env.PANEL_TEST_ORIGIN+'/api/egress?view=diagnostics')).json();assert(state.total>0);assert(state.entries.every(x=>x.hash.length===64));
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
