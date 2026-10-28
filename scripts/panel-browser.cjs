// Development acceptance only. Requires externally supplied, pinned playwright-core.
// No production URLs or credentials: the companion runner creates an isolated DB.
const fs=require('fs');
const assert=require('assert/strict');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE);
const dir=process.env.PANEL_TEST_DIR;
const origin='http://127.0.0.1:18192';
const phase=process.argv[2];
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
 try {
 const context=await browser.newContext({viewport:{width:1440,height:1000},...(phase==='2'?{storageState:dir+'/session.json'}:{})});
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto(origin);
 const noOverflow=async()=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,'horizontal overflow');
 if(phase==='1'){
  await page.getByLabel('رمز عبور',{exact:true}).fill('synthetic-browser-password');
  const [loginResponse]=await Promise.all([page.waitForResponse(r=>r.url()===origin+'/login' && r.request().method()==='POST'),page.getByRole('button',{name:'ورود به پنل',exact:true}).click()]);assert.equal(loginResponse.status(),303,await loginResponse.text().catch(()=>''));await page.waitForURL(origin+'/');
  await page.goto(origin+'/sources/new');
  await page.getByLabel('نام منبع',{exact:true}).fill('منبع آزمایشی');
  await page.getByLabel('لینک منبع',{exact:true}).fill('https://example.com/private-synthetic-token?key=fixture');
  await page.getByRole('button',{name:'ذخیرهٔ منبع',exact:true}).click();await page.waitForURL(origin+'/');
  assert.equal(await page.locator('.source-card').count(),1);
  assert.equal((await page.locator('body').innerText()).includes('private-synthetic-token'),false);
  await page.getByRole('link',{name:'ویرایش',exact:true}).click();
  assert.equal(await page.getByLabel('لینک منبع',{exact:true}).inputValue(),'https://example.com/private-synthetic-token?key=fixture');
  await page.getByLabel('نام منبع',{exact:true}).fill('منبع اصلی — نمونه');
  await page.locator('#min').fill('25');await page.locator('#max').fill('35');
  await page.getByRole('button',{name:'ذخیرهٔ منبع',exact:true}).click();await page.waitForURL(origin+'/');
  await page.getByRole('button',{name:'توقف',exact:true}).click();await page.waitForURL(origin+'/');
  assert.equal(await page.locator('.source-card .badge').innerText(),'متوقف');
  await noOverflow();await page.screenshot({path:dir+'/desktop.png',fullPage:true});
  await page.setViewportSize({width:390,height:844});await noOverflow();await page.screenshot({path:dir+'/mobile.png',fullPage:true});
  await page.getByRole('link',{name:'ویرایش',exact:true}).click();await noOverflow();await page.screenshot({path:dir+'/mobile-edit.png',fullPage:true});
  await context.storageState({path:dir+'/session.json'});
 }else{
  assert.equal(new URL(page.url()).pathname,'/','login lost after process restart');
  assert.equal(await page.locator('.source-card').count(),1);assert.equal(await page.locator('.source-card .badge').innerText(),'متوقف');
  assert.match(await page.locator('.source-card').innerText(),/25 تا 35/);
  await page.getByRole('button',{name:'فعال‌سازی',exact:true}).click();await page.waitForURL(origin+'/');
  assert.equal(await page.locator('.source-card .badge').innerText(),'فعال');
  await page.getByRole('link',{name:'حذف',exact:true}).click();
  await page.getByRole('button',{name:'بله، منبع حذف شود',exact:true}).click();await page.waitForURL(origin+'/');
  assert.equal(await page.locator('.source-card').count(),0);
  await page.getByRole('link',{name:'امنیت حساب',exact:true}).click();
  await page.getByLabel('رمز فعلی',{exact:true}).fill('synthetic-browser-password');
  await page.getByLabel('رمز جدید',{exact:true}).fill('synthetic-new-browser-password');
  await page.getByLabel('تکرار رمز جدید',{exact:true}).fill('synthetic-new-browser-password');
  await page.getByRole('button',{name:'تغییر رمز و خروج',exact:true}).click();await page.waitForURL(origin+'/login?changed=1');
  await page.getByLabel('رمز عبور',{exact:true}).fill('synthetic-new-browser-password');
  await page.getByRole('button',{name:'ورود به پنل',exact:true}).click();await page.waitForURL(origin+'/');
  await page.getByRole('button',{name:'خروج',exact:true}).click();await page.waitForURL(origin+'/login');
  await page.goto(origin+'/');await page.waitForURL(origin+'/login');
 }
 assert.deepEqual(errors,[]);console.log(JSON.stringify({phase,passed:true,browser:await browser.version()}));
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
