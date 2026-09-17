import json,tempfile,subprocess,time,os,socket
from .health_settings import load
from .vless_adapter import xray as vx
from .vmess_adapter import xray as mx
from .trojan_adapter import xray as tx
from .ss_adapter import xray as sx
from .sandbox_profiler import add as prof
ADAPTERS={'vless':vx,'vmess':mx,'trojan':tx,'ss':sx}
def free_port():
 s=socket.socket();s.bind(('127.0.0.1',0));p=s.getsockname()[1];s.close();return p
def build(kind,raw,port):
 if kind.startswith('json-'):o=json.loads(raw)
 elif kind in ADAPTERS:o=json.loads(ADAPTERS[kind](raw))
 else:raise ValueError('unsupported-sandbox-kind')
 o.setdefault('log',{})['loglevel']='none';o['inbounds']=[{'listen':'127.0.0.1','port':port,'protocol':'socks','settings':{'udp':True}}];return o
def start(kind,raw):
 try:
  a=time.time();port=free_port();prof('port_ms',(time.time()-a)*1000);a=time.time();cfg=build(kind,raw,port);prof('build_ms',(time.time()-a)*1000);a=time.time();f=tempfile.NamedTemporaryFile('w',suffix='.json',delete=False);json.dump(cfg,f);f.close();prof('write_ms',(time.time()-a)*1000)
 except Exception as e:return {'ok':False,'error':'build:'+type(e).__name__}
 a=time.time();p=subprocess.Popen(['/usr/local/bin/xray','run','-c',f.name],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,text=True,start_new_session=True);prof('spawn_ms',(time.time()-a)*1000);a=time.time();deadline=time.time()+load()['startup_timeout_ms']/1000
 while time.time()<deadline:
  if p.poll() is not None:break
  s=socket.socket();s.settimeout(.05)
  try:s.connect(('127.0.0.1',port));s.close();prof('ready_wait_ms',(time.time()-a)*1000);prof('ok',1);return {'ok':True,'process':p,'file':f.name,'port':port,'started':time.time()}
  except:time.sleep(.03)
 prof('ready_wait_ms',(time.time()-a)*1000);prof('fail',1);err=(p.stderr.read()[-500:] if p.poll() is not None else 'startup-timeout');stop({'process':p,'file':f.name});return {'ok':False,'error':err}
def stop(h):
 p=h.get('process');
 if p and p.poll() is None:p.terminate();
 try:p.wait(timeout=1)
 except:
  if p:p.kill()
 try:os.unlink(h.get('file',''))
 except:pass