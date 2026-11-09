// Independent, bounded feature loops. Slow or failed siblings cannot stall us.
export function poll(task, {interval=2000, maxDelay=15000}={}) {
 let pending, timer, failures=0, stopped=false;
 async function run() {
  if(stopped||document.hidden)return;
  if(pending)return pending;
  clearTimeout(timer);
  pending=(async()=>{
   try { failures=(await Promise.resolve().then(task))===false?failures+1:0; }
   catch { failures++; }
   finally {
    pending=null;
    if(!stopped)timer=setTimeout(run,Math.min(maxDelay,interval*2**Math.min(failures,4)));
   }
  })();
  return pending;
 }
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)run()});
 addEventListener('pagehide',()=>{stopped=true;clearTimeout(timer)});
 addEventListener('pageshow',()=>{stopped=false;run()});
 queueMicrotask(run);
 return run;
}
