#!/usr/bin/env python3
"""Finite same-stream measurements. No redirects, ads, clicks or IP forwarding.
The error-page IP is a report by the authenticated Ads TLS peer, not a signed
Google diagnostic contract. The second HTTP authority is recorded separately.
"""
import asyncio,hashlib,ipaddress,json,pathlib,re,ssl,struct,sys,time,uuid
def target_registry(values=None):
 if values is None:
  values=json.loads((pathlib.Path(__file__).resolve().parents[1]/'internal/egress/targets.json').read_text())
 if not isinstance(values,list) or not 1<=len(values)<=32:raise ValueError('target_registry_invalid')
 seen=set()
 for t in values:
  if not isinstance(t,dict) or not re.fullmatch(r'[a-z0-9.-]{1,253}',t.get('domain','')) or t['domain'] in seen or t.get('role') not in ('ads','control') or t.get('driver') not in ('google-peer','cloudflare-trace','ipify','https-reachability'):raise ValueError('target_registry_invalid')
  seen.add(t['domain'])
 return values
try:DEFAULT_TARGETS=target_registry()
except FileNotFoundError:DEFAULT_TARGETS=[]
DOMAINS=[t['domain'] for t in DEFAULT_TARGETS if t['role']=='ads']
CONTROLS=[t['domain'] for t in DEFAULT_TARGETS if t['role']=='control']
TLS_CONTEXT=ssl.create_default_context();TLS_CONTEXT.set_alpn_protocols(['http/1.1'])
def digest(b):return hashlib.sha256(b).hexdigest()
def now():return int(time.time()*1000)
def authority(cert,name):
 for typ,value in cert.get('subjectAltName',[]):
  if typ=='DNS' and (value==name or value.startswith('*.') and name.count('.')==value.count('.') and name.endswith(value[1:])):return True
 return False
async def read_response(reader,close,head=False):
 header=await reader.readuntil(b'\r\n\r\n')
 if len(header)>32768:raise ValueError('header_limit')
 lines=header.decode('iso-8859-1').split('\r\n');status=int(lines[0].split()[1]);h={}
 for line in lines[1:]:
  if ':' in line:
   k,v=line.split(':',1);k=k.lower()
   if k in h and k in ('content-length','transfer-encoding','connection','location'):raise ValueError('duplicate_header')
   h[k]=v.strip()
 body=b''
 if head or status in (204,304):pass
 elif 100<=status<200:raise ValueError('interim_response')
 elif 'transfer-encoding' in h:
  if h['transfer-encoding'].lower()!='chunked' or 'content-length' in h:raise ValueError('ambiguous_framing')
  while True:
   line=await reader.readline()
   if len(line)>128:raise ValueError('chunk_limit')
   n=int(line.split(b';')[0],16)
   if n<0 or len(body)+n>65536:raise ValueError('body_limit')
   if n==0:
    trailers=0
    while True:
     line=await reader.readline();trailers+=len(line)
     if trailers>4096 or not line:raise ValueError('trailers_limit')
     if line==b'\r\n':break
    break
   body+=await reader.readexactly(n)
   if await reader.readexactly(2)!=b'\r\n':raise ValueError('chunk_framing')
 elif 'content-length' in h:
  n=int(h['content-length'])
  if not 0<=n<=65536:raise ValueError('body_limit')
  body=await reader.readexactly(n)
 elif close:
  while True:
   b=await reader.read(min(8192,65537-len(body)))
   if not b:break
   body+=b
   if len(body)>65536:raise ValueError('body_limit')
 else:raise ValueError('persistent_framing_missing')
 return status,h,body
def socks_target(domain,control_ipv4=None):
 if domain=='one.one.one.one':return b'\x04'+ipaddress.ip_address('2606:4700:4700::1111').packed
 host=domain.encode('ascii');return b'\x03'+bytes([len(host)])+host
async def measure(domain,variant,wave,timeout,control_ipv4=None,target=None):
 target=target or next((t for t in DEFAULT_TARGETS if t['domain']==domain),None)
 if target is None:raise ValueError('target_driver_missing')
 nonce=uuid.uuid4().hex;writer=None
 row=dict(domain=domain,variant=variant,wave=wave,at=now(),transport='tcp',nonce=nonce,connection=nonce,ip='',family=0,tls=False,same_stream=False,echo=False,actual_ads=False,code='unavailable',certificate='',response='',first_status=0,echo_status=0,diagnostic_authority=False,request_bytes=0,response_bytes=0,response_complete=False)
 try:
  async with asyncio.timeout(timeout):
   reader,writer=await asyncio.open_connection('127.0.0.1',10808,limit=32769)
   writer.write(b'\x05\x01\x00');await writer.drain()
   if await reader.readexactly(2)!=b'\x05\x00':raise ValueError('socks_greeting')
   address=socks_target(domain,control_ipv4);writer.write(b'\x05\x01\x00'+address+struct.pack('!H',443));await writer.drain()
   version,rep,_,typ=await reader.readexactly(4)
   if version!=5 or rep:raise ValueError('socks_connect')
   n={1:4,4:16}.get(typ)
   if typ==3:n=(await reader.readexactly(1))[0]
   if n is None:raise ValueError('socks_address')
   await reader.readexactly(n+2)
   await writer.start_tls(TLS_CONTEXT,server_hostname=domain,ssl_handshake_timeout=timeout)
   tls=writer.get_extra_info('ssl_object');row.update(tls=True,certificate=digest(tls.getpeercert(binary_form=True)))
   async def get(host,path,close,method="GET"):
    request=(method+' '+path+' HTTP/1.1\r\nHost: '+host+'\r\nUser-Agent: Shen-Egress-Audit/3\r\nCache-Control: no-cache\r\nAccept-Encoding: identity\r\nConnection: '+('close' if close else 'keep-alive')+'\r\n\r\n').encode()
    writer.write(request);await writer.drain();row['request_bytes']+=len(request)
    result=await read_response(reader,close,method=="HEAD");row['response_bytes']+=len(result[2]);return result
   driver=target['driver']
   if driver=='google-peer':
    status,h,b=await get(domain,'/robots.txt',False);row.update(first_status=status,actual_ads=target['role']=='ads')
    if not 200<=status<500:raise ValueError('target_http_status')
    if h.get('connection','').lower()=='close':raise ValueError('ads_connection_closed')
    row['diagnostic_authority']=authority(tls.getpeercert(),'www.google.com')
    status,h,b=await get('www.google.com','/sorry/index?shen_probe='+nonce,True);row['echo_status']=status
    values=re.findall(rb'IP address:\s*([^<\s]+)',b)
    if len(values)!=1 or nonce.encode() not in b:raise ValueError('ads_echo_unavailable')
    value=values[0].decode('ascii');row['code']='ads_peer_reported' if target['role']=='ads' else 'google_control_peer_reported'
   elif driver in ('cloudflare-trace','ipify'):
    path=('/cdn-cgi/trace?shen_probe=' if driver=='cloudflare-trace' else '/?format=json&shen_probe=')+nonce
    status,h,b=await get(domain,path,True);row['first_status']=status
    if status!=200:raise ValueError('control_http_status')
    if driver=='cloudflare-trace':
     fields=dict(x.split('=',1) for x in b.decode().splitlines() if '=' in x)
     if fields.get('h')!=domain:raise ValueError('control_authority')
     value=fields['ip']
    else:value=json.loads(b)['ip']
    row['code']='control_echo'
   else:
    status,h,b=await get(domain,'/',True,'HEAD')
    row.update(first_status=status,same_stream=True,response=digest(b),response_complete=True,code='reachable_no_source')
    return row
   ip=ipaddress.ip_address(value)
   if not ip.is_global or ip.is_multicast or ip.is_reserved:raise ValueError('nonpublic_echo')
   row.update(ip=str(ip),family=ip.version,same_stream=True,echo=True,response=digest(b),response_complete=True)
 except (TimeoutError,asyncio.TimeoutError):row['code']='connection_timeout'
 except ssl.SSLCertVerificationError:row['code']='tls_verification_failed'
 except ValueError as e:row['code']=str(e) if re.fullmatch('[a-z_]{1,48}',str(e)) else 'invalid_response'
 except Exception:row['code']='connection_failed'
 finally:
  if writer:writer.close()
 return row

async def session(req):
 targets=target_registry(req.get('targets'));limit=asyncio.Semaphore(12)
 async def run(t,n):
  async with limit:return await measure(t['domain'],req['variant'],n,req.get('timeout',3),req.get('control_ipv4'),t)
 async def wave(n,selected,delay=0):
  await asyncio.sleep(delay)
  return await asyncio.gather(*(run(t,n) for t in selected))
 start=req.get('start_wave',1);waves=req.get('waves',3)
 if not 1<=waves<=3:raise ValueError('wave_limit')
 if req.get('adaptive') and waves==3:
  groups=[await wave(start,targets),await wave(start+1,targets)]
  # Repeating an unsupported source observation cannot reveal an IP. Retain
  # both completed reachability observations and report their limited scope.
  complete=[{x['domain'] for x in g if x.get('echo')} for g in groups]
  required=[t for t in targets if t.get('required')]
  missing=[t for t in required if t['domain'] not in complete[0]&complete[1]]
  # Recovery must remeasure the comparison destinations in the SAME wave.
  # Repeating only a failed Ads request leaves its recovered observation unpaired.
  # Reachability-only destinations cannot supply source evidence by retrying.
  if missing:groups.append(await wave(start+2,required))
 elif req.get('sequential'):
  groups=[]
  for i in range(waves):groups.append(await wave(start+i,targets))
 else:groups=await asyncio.gather(*(wave(start+i,targets,i*.15) for i in range(waves)))
 return [row for rows in groups for row in rows]
if __name__=='__main__':print(json.dumps(asyncio.run(session(json.load(sys.stdin)))))
