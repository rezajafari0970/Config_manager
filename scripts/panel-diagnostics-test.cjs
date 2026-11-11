'use strict';
const assert=require('assert/strict');
module.exports=async function(context,origin){
 const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 let updates=0;
 await page.route('**/api/state*',async route=>{
  const response=await route.fetch(),data=await response.json();
  if(data.sources.length){
   const s=data.sources[0];s.state='updated';s.error_code='';s.enabled=true;s.config_count=2;s.content_state='config';
   data.conversions[s.id]={total:2,ready:0,connected:0,pending:0,blocked:2,repaired:0,issues:{ipv6_only_probe_pending:1,transport_verification_incomplete:1}};
   s.fetch_count=++updates;
  }
  await route.fulfill({response,json:data});
 });
 await page.goto(origin+'/input');
 await page.locator('[data-conversion-issue="ipv6_only_probe_pending"]').waitFor({state:'attached'});
 const issue=page.locator('[data-conversion-issue="ipv6_only_probe_pending"]').first();
 await issue.evaluate(el=>{el.closest('.source-row').open=true});
 assert.match(await issue.textContent(),/محیط آزمون/);
 await issue.locator('summary').click();
 const initial=updates;
 await page.waitForFunction(()=>document.querySelector('[data-conversion-issue="ipv6_only_probe_pending"]').open);
 for(let i=0;updates<=initial&&i<100;i++)await page.waitForTimeout(100);
 assert(updates>initial,'no independent refresh');
 assert.equal(await issue.getAttribute('open'),'','issue expansion lost after polling');
 await page.unroute('**/api/state*');
 const list=await(await context.request.get(origin+'/api/conversions')).json();
 const item=list.entries.find(e=>e.candidate_hash);assert(item);
 let phase='valid',observed=0;
 await page.route('**/api/conversion/evidence/*',async route=>{
  observed++;
  const data={raw_sha256:item.hash,candidate_sha256:item.candidate_hash,tested:item.tested,checks:{https:'passed',dns_tcp:'passed',dns_udp:'passed',quic_vn:'failed',android:'not_run'}};
  if(phase==='stale')data.candidate_sha256='0'.repeat(64);
  await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(data)});
 });
 await page.goto(origin+'/output');
 const button=page.locator('[data-evidence="'+item.hash+'"]');await button.waitFor();
 await button.click();
 const detail=button.locator('xpath=following-sibling::div[1]');
 await detail.waitFor({state:'visible'});
 assert.match(await detail.textContent(),/HTTPS با گواهی معتبرموفق/);
 assert.match(await detail.textContent(),/پاسخ QUIC روی UDP\/443ناموفق/);
 assert.match(await detail.textContent(),/اجرای واقعی Androidاجرا نشده/);
 assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 await page.reload();await button.waitFor();phase='stale';
 await button.click();
 assert.equal(await detail.isVisible(),false,'stale evidence rendered');
 assert(observed>=2);
 assert.deepEqual(errors,[]);
 await page.close();
};
