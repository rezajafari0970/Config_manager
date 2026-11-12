#!/usr/bin/env python3
"""Exact-profile, bounded real HTTPS A/B load; explicitly not an Android test."""
import argparse,concurrent.futures,hashlib,importlib.util,json,pathlib,secrets,subprocess,time
root=pathlib.Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser();p.add_argument('--baseline',required=True);p.add_argument('--candidate',required=True);p.add_argument('--output',required=True);p.add_argument('--core',required=True);args=p.parse_args()
assert hashlib.sha256(pathlib.Path(args.core).read_bytes()).hexdigest() in {'e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'}
out=pathlib.Path(args.output);out.mkdir(mode=0o700,parents=True,exist_ok=True)
s=importlib.util.spec_from_file_location('lab',root/'scripts/live-probe.py');lab=importlib.util.module_from_spec(s);s.loader.exec_module(lab)
def curl(box,url,expected,timeout=8,extra=()):
 args=[lab.IP,'netns','exec',box.name,'/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--silent','--show-error','--max-time',str(timeout),'--connect-timeout','7','--output','/dev/null','--write-out','%{json}',*extra,url]
 try:
  r=subprocess.run(args,capture_output=True,timeout=timeout+3)
  try:m=json.loads(r.stdout)
  except ValueError:m={}
  return {'passed':r.returncode==0 and m.get('ssl_verify_result')==0 and m.get('http_code')==expected,'curl_exit':r.returncode,'http_status':m.get('http_code'),'bytes':m.get('size_download',0),'ms':round(m.get('time_total',0)*1000),'tls':m.get('ssl_verify_result')}
 except subprocess.SubprocessError:return {'passed':False,'curl_exit':124}
def run(phase,path,slot):
 data=path.read_bytes();config=json.loads(data);report={'phase':phase,'sha256':hashlib.sha256(data).hexdigest(),'scope':'Linux server network, same exact profile; 8 MiB download deliberately rate-capped to 64 KiB/s to hold a session for about 128 seconds; not peak throughput','rounds':[],'android':'not_run'}
 with lab.Sandbox(args.core,secrets.token_hex(3),slot,prefix='cstf') as box:
  box.policy(lab.endpoints(config));p=box.launch(data);time.sleep(2)
  with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
   start=time.monotonic()
   download=pool.submit(curl,box,'https://speed.cloudflare.com/__down?bytes=8388608&nonce='+str(time.time_ns()),200,145,('--limit-rate','65536','--max-filesize','8388608'))
   for i in range(12):
    row={'elapsed_s':round(time.monotonic()-start,2),'core_alive':p.poll() is None,'checks':{}}
    for name,url,status in [('google','https://www.gstatic.com/generate_204',204),('instagram_https','https://www.instagram.com/robots.txt',200),('telegram_https','https://telegram.org/',200)]:
     row['checks'][name]=curl(box,url,status,8,('--max-filesize','2097152'))
    report['rounds'].append(row)
    (out/('sustained-'+phase+'.json')).write_text(json.dumps(report,indent=2))
    if i<11:time.sleep(max(0,start+(i+1)*10-time.monotonic()))
   report['download']=download.result();report['download']['passed']=report['download']['passed'] and report['download']['bytes']==8388608
   report['elapsed_s']=round(time.monotonic()-start,2)
 report['all_passed']=report['download']['passed'] and all(r['core_alive'] and all(x['passed'] for x in r['checks'].values()) for r in report['rounds'])
 (out/('sustained-'+phase+'.json')).write_text(json.dumps(report,indent=2))
 return {'phase':phase,'all_passed':report['all_passed'],'elapsed_s':report['elapsed_s'],'download':report['download'],'failed_requests':sum(not x['passed'] for r in report['rounds'] for x in r['checks'].values())}
with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
 jobs=[pool.submit(run,'baseline',pathlib.Path(args.baseline),8197),pool.submit(run,'candidate',pathlib.Path(args.candidate),8198)]
 for job in jobs:print(json.dumps(job.result()),flush=True)
