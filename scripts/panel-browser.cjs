'use strict';
const assert=require('assert/strict');const fs=require('fs');const crypto=require('crypto');const {chromium}=require(process.env.PLAYWRIGHT_MODULE);
const dir=process.env.PANEL_TEST_DIR,origin='http://127.0.0.1:18192',phase=process.argv[2];
const uri='vless://00000000-0000-4000-8000-000000000001@vpn.example:443?security=tls#Original%20Label\r\n';
const custom='{ "outbounds" : [{"protocol":"vless","settings":{"address":"vpn.example","port":443,"id":"00000000-0000-4000-8000-000000000002"}}], "stats" : [], "opaque" : 9007199254740993 }\r\n';
(async()=>{const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});try{
 const context=await browser.newContext({viewport:{width:1440,height:1100}});const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto(origin+'/input');assert.equal(await page.locator('input[type=password]').count(),0);
 const state=async()=>{const response=await context.request.get(origin+'/api/state');assert.equal(response.status(),200);return response.json()};
 const waitFor=async predicate=>{const deadline=Date.now()+20000;while(Date.now()<deadline){const s=await state();if(predicate(s))return s;await new Promise(r=>setTimeout(r,150))}throw new Error('state timeout')};
 const noOverflow=async()=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,'horizontal overflow');
 if(phase==='1'){
  await page.locator('#bulk-links').fill('https://example.com/Api/Verde/trun.php\nhttps://example.com/Api/Rosa/trun.php\nhttps://example.com/Api/Broken/trun.php\nhttps://example.com/Api/Verde/trun.php');
  await page.locator('#bulk-interval').fill('۲ تا ۳');await page.getByRole('button',{name:'افزودن لینک‌ها',exact:true}).click();
  const s=await waitFor(s=>s.stats.sources===3&&s.stats.configs===2&&s.sources.some(x=>x.name==='Broken'&&x.state==='error'));
  assert.deepEqual(s.sources.map(x=>x.name).sort(),['Broken','Rosa','Verde']);assert.equal(s.settings.retention_seconds,7200);assert.ok(s.sources.every(x=>x.min===2&&x.max===3));
  await page.waitForFunction(()=>document.querySelectorAll('.source-row .badge.good').length===2&&document.querySelectorAll('.source-row .badge.bad').length===1);
  const verde=page.locator('.source-row').filter({has:page.locator('.source-name strong',{hasText:'Verde'})});await verde.locator('summary').click();
  await noOverflow();await page.screenshot({path:dir+'/input-desktop.png',fullPage:true});
  await page.setViewportSize({width:390,height:844});await noOverflow();await page.screenshot({path:dir+'/input-mobile.png',fullPage:true});
  await page.getByRole('button',{name:'باز کردن منو',exact:true}).click();await page.screenshot({path:dir+'/drawer-mobile.png',fullPage:false,animations:'disabled'});await page.locator('#nav-drawer').getByRole('link',{name:/Output/}).click();await page.waitForURL(origin+'/output');
  await page.waitForFunction(()=>document.querySelectorAll('.output-row').length===2);await noOverflow();
  const data=await state();for(const item of data.outputs){const response=await context.request.get(origin+'/output/raw/'+item.id);const body=await response.body();assert.equal(crypto.createHash('sha256').update(body).digest('hex'),item.hash);assert.equal(body.toString(),item.kind==='json'?custom:uri)}
  await page.locator('.output-row').first().getByRole('button',{name:'نمایش',exact:true}).click();await page.waitForFunction(()=>[...document.querySelectorAll('.output-row pre')].some(p=>!p.classList.contains('hidden')));await page.screenshot({path:dir+'/output-mobile.png',fullPage:true});
  const response=await context.request.get(origin+'/input/raw/'+data.sources.find(x=>x.name==='Verde').id);assert.equal((await response.body()).toString(),uri+uri+custom);
  await page.goto(origin+'/setting');await page.waitForFunction(()=>document.querySelector('#retention-hours').value==='2');await noOverflow();await page.screenshot({path:dir+'/setting-mobile.png',fullPage:true});
  await page.setViewportSize({width:1440,height:1100});await page.screenshot({path:dir+'/setting-desktop.png',fullPage:true});
  await page.goto(origin+'/input');await page.waitForFunction(()=>document.querySelectorAll('.source-row').length===3);
  const rosa=page.locator('.source-row').filter({has:page.locator('.source-name strong',{hasText:'Rosa'})});await rosa.locator('summary').click();await rosa.getByRole('button',{name:'ویرایش',exact:true}).click();await page.locator('#edit-name').fill('Rosa Edited');await page.locator('#edit-interval').fill('۴ تا ۵');await page.locator('#edit-unit').selectOption('seconds');await page.getByRole('button',{name:'ذخیرهٔ تغییرات',exact:true}).click();await waitFor(s=>s.sources.some(x=>x.name==='Rosa Edited'&&x.min===4&&x.max===5&&x.unit==='seconds'));
  // Pause every source through the same public API, preserving bytes for restart/expiry.
  const token=await page.locator('meta[name=csrf-token]').getAttribute('content');for(const source of (await state()).sources){const r=await context.request.post(origin+'/api/sources/'+source.id+'/toggle',{headers:{Origin:origin,'X-CSRF-Token':token},data:{revision:source.revision}});assert.equal(r.status(),200)}
  await waitFor(s=>s.sources.every(x=>!x.enabled)&&s.stats.configs===2);
 }else{
  const initial=await waitFor(s=>s.stats.sources===3&&s.stats.configs===2);assert.ok(initial.sources.every(x=>!x.enabled));assert.ok(initial.sources.some(x=>x.name==='Rosa Edited'&&x.min===4));
  await page.goto(origin+'/output');await page.waitForFunction(()=>document.querySelectorAll('.output-row').length===2);const [download]=await Promise.all([page.waitForEvent('download'),page.getByRole('link',{name:'دانلود همه',exact:true}).click()]);await download.saveAs(dir+'/configs-raw.zip');
  await page.goto(origin+'/setting');await page.locator('#retention-hours').fill('۰٫۰۰۲۵');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await waitFor(s=>s.settings.retention_seconds===9&&s.stats.configs===0);
  await page.locator('#retention-hours').fill('2');await page.getByRole('button',{name:'ذخیرهٔ تنظیمات',exact:true}).click();await waitFor(s=>s.settings.retention_seconds===7200&&s.stats.configs===0);
  await page.goto(origin+'/input');await page.waitForFunction(()=>document.querySelectorAll('.source-row').length===3);const broken=page.locator('.source-row').filter({has:page.locator('.source-name strong',{hasText:'Broken'})});await broken.locator('summary').click();await broken.getByRole('button',{name:'حذف',exact:true}).click();await page.locator('#confirm-dialog').getByRole('button',{name:'حذف منبع',exact:true}).click();await waitFor(s=>s.stats.sources===2);
 }
 assert.deepEqual(errors,[]);console.log(JSON.stringify({phase,passed:true,browser:await browser.version()}));
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
