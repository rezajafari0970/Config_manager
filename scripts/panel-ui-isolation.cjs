'use strict';
const assert=require('assert/strict');
module.exports=async(context,origin)=>{
 const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 const waits=[];let heavyReads=0,posts=0,conflict=false,postRevisions=[];
 await page.route('**/api/state?*',r=>r.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'synthetic summary outage'})}));
 await page.route('**/api/conversions*',r=>{heavyReads++;return r.continue()});
 await page.route('**/api/conversion/settings',async r=>{await new Promise(resolve=>waits.push(resolve));await r.continue()});
 await page.route('**/api/smart/settings',async r=>{
  if(r.request().method()==='POST'){
   posts++;postRevisions.push(r.request().postDataJSON().revision);conflict=true;
   return r.fulfill({status:posts===1?409:200,contentType:'application/json',body:posts===1?' {"error":"synthetic conflict"}':'true'});
  }
  return r.fulfill({contentType:'application/json',body:JSON.stringify({settings:{enabled:true,count:posts>1?23:10,revision:conflict?9:8}})});
 });
 try{
  await page.goto(origin+'/setting');
  await page.waitForFunction(()=>!document.querySelector('#smart-member-count').disabled);
  assert.equal(await page.locator('#smart-member-count').inputValue(),'10');
  assert.equal(await page.locator('#probe-enabled').isDisabled(),true);
  assert.equal(await page.locator('#probe-settings').getAttribute('aria-busy'),'true');
  assert.equal(heavyReads,0,'settings scanned entire conversion list');
  await page.locator('#smart-member-count').fill('23');
  // Failed shared state and a pending sibling cannot stop settings or saves.
  await page.locator('#smart-settings button').click();
  await page.waitForFunction(()=>document.querySelector('#smart-settings [role=status]').textContent.includes('جای دیگری'));
  assert.equal(await page.locator('#smart-member-count').inputValue(),'23');
  await page.locator('#smart-settings button').click();
  await page.waitForFunction(()=>document.querySelector('#smart-settings [role=status]').textContent.includes('ذخیره شد'));
  assert.deepEqual(postRevisions,[8,9]);assert.equal(posts,2);
  waits.splice(0).forEach(resolve=>resolve());await page.unroute('**/api/conversion/settings');
  await page.waitForFunction(()=>!document.querySelector('#probe-enabled').disabled);
  assert.equal(await page.locator('#probe-enabled').isChecked(),true);
  assert.equal(heavyReads,0);
  await page.unroute('**/api/smart/settings');
  // Output and Smart listings load even while summary fails; links stay
  // disabled until their owning API has supplied an actual URL.
  for(const [path,api,copy,link,view,list] of [
   ['/output','**/api/conversions?*','#copy-single','#single-url','[data-view]','#output-list'],
   ['/smart','**/api/smart?*','#smart-copy-single','#smart-single-url','[data-smart-view]','#smart-list']
  ]){
   let release,entered;const started=new Promise(r=>entered=r),held=new Promise(r=>release=r);
   await page.route(api,async route=>{entered();await held;await route.continue()});
   await page.goto(origin+path);await started;
   assert.equal(await page.locator(copy).isDisabled(),true);assert.equal(await page.locator(link).getAttribute('href'),null);
   release();await page.unroute(api);await page.waitForFunction(selector=>!document.querySelector(selector).disabled,copy);
   assert((await page.locator(link).getAttribute('href')).startsWith(origin));
   await page.waitForSelector(view);
   await page.route('**'+path+'/json/*',async route=>{await page.evaluate(selector=>{const list=document.querySelector(selector);list.innerHTML=list.innerHTML},list);await route.continue()});
   await page.locator(view).first().click();await page.waitForFunction(selector=>[...document.querySelectorAll(selector+' pre')].some(p=>!p.hidden&&p.textContent.startsWith('{')),list);
   await page.unroute('**'+path+'/json/*');
  }
  assert.deepEqual(errors,[]);
 }finally{waits.forEach(r=>r());await page.close()}
};
