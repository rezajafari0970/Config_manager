const {chromium}=require(process.env.PLAYWRIGHT_MODULE);
const fs=require('fs');
(async()=>{
 const dir=process.env.CONFIG_RESIDENTIAL_BROWSER_FIXTURE_DIR;
 const {origin}=JSON.parse(fs.readFileSync(dir+'/public.json','utf8'));
 const browser=await chromium.launch({executablePath:process.env.CHROME_PATH,headless:true,args:['--no-sandbox']});
 const context=await browser.newContext({permissions:['clipboard-read','clipboard-write']});
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const sizes=[];
 for(const width of [390,1440]){
  await page.setViewportSize({width,height:900});await page.goto(origin+'/residential');
  await page.waitForFunction(()=>document.querySelector('#res-list .config-card'));
  if(await page.locator('#res-links a').count()!==6)throw Error('missing links');
  const core=page.locator('#res-list details').filter({has:page.locator('summary',{hasText:'سازگاری هسته'})}).first();
  await core.locator('summary').click();
  if(!(await core.innerText()).includes('xray-26.5.9')||!(await core.innerText()).includes('HEALTHY'))throw Error('compatibility diagnostics missing');
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
  if(overflow)throw Error('horizontal overflow '+width);
  await page.screenshot({path:dir+'/residential-'+width+'.png',fullPage:true});sizes.push(width);
 }
 const state=await(await page.request.get(origin+'/api/residential')).json();
 if(!state.links['raw.txt']||state.links['raw.zip'])throw Error('raw format not migrated');
 if(!state.progress||state.total!==3)throw Error('missing progress');
 if(state.eligible_count!==2||state.ads_different.eligible_count!==1)throw Error('fixture count mismatch');
 const generations=[];
 for(const [name,url] of Object.entries(state.links)){
  const response=await page.request.get(url+'?generation='+state.generation);
  if(response.status()!==200)throw Error('export '+name);
  if(name==='raw.txt'&&(!response.headers()['content-type'].startsWith('text/plain')||response.headers()['content-disposition']!=='inline'))throw Error('raw not inline');generations.push(response.headers()['x-residential-generation']);
 }
 if(generations.some(x=>x!==state.generation))throw Error('generation mismatch');
 for(const [name,url] of Object.entries(state.ads_different.links)){
  const response=await page.request.get(url+'?generation='+state.ads_different.generation);
  if(response.status()!==200||response.headers()['x-residential-generation']!==state.ads_different.generation||!response.headers()['x-evidence-class'].includes('not-residential'))throw Error('Ads export separation '+name);
 }
 await page.locator('[data-copy="raw.txt"][data-pool="ads"]').click();
 if(await page.evaluate(()=>navigator.clipboard.readText())!==state.ads_different.links['raw.txt'])throw Error('Ads copy mismatch');
 await page.locator('[data-copy="configs.jsonl"][data-pool="res"]').click();
 if(await page.evaluate(()=>navigator.clipboard.readText())!==state.links['configs.jsonl'])throw Error('copy mismatch');
 await page.locator('.config-card').filter({has:page.locator('strong',{hasText:'رزیدنتال تأییدشده'})}).first().locator('[data-res-action="recheck"]').click();
 await page.waitForFunction(()=>document.querySelector('#res-count').textContent.includes('۱'));
 page.once('dialog',d=>d.accept());await page.locator('[data-res-action="delete"]').first().click();
 await page.waitForFunction(()=>document.querySelector('#res-list').textContent.includes('حذف تأییدشده'));
 if(errors.length)throw Error(errors.join('\n'));
 await browser.close();console.log(JSON.stringify({passed:true,evidence:'synthetic-fixture-only',widths:sizes,copy:true,three_output_generation:true,ads_outputs_separate:true,recheck_quarantine:true,manual_delete:true,page_errors:errors}));
})().catch(e=>{console.error(e);process.exit(1)});