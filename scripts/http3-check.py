#!/usr/bin/env python3
"""Full HTTP/3 DoH request via SOCKS5 UDP, with verified TLS and bounded I/O.
Requires pinned aioquic 1.3.0. No direct networking fallback.
"""
import base64,importlib.util,json,pathlib,socket,ssl,time,sys
from aioquic.quic.configuration import QuicConfiguration
from aioquic.quic.connection import QuicConnection
from aioquic.quic.events import HandshakeCompleted,ConnectionTerminated
from aioquic.h3.connection import H3Connection,H3_ALPN
from aioquic.h3.events import HeadersReceived,DataReceived
spec=importlib.util.spec_from_file_location('transport',pathlib.Path(__file__).with_name('transport-checks.py'))
transport=importlib.util.module_from_spec(spec);spec.loader.exec_module(transport)
def check(timeout=8):
 control,relay=transport.socks(3,'0.0.0.0',0,timeout)
 if relay[0] in ('0.0.0.0','::'):relay=('127.0.0.1',relay[1])
 dest=('1.1.1.1',443);ident,query=transport.dns_query()
 config=QuicConfiguration(is_client=True,alpn_protocols=H3_ALPN,server_name='cloudflare-dns.com',verify_mode=ssl.CERT_REQUIRED,idle_timeout=timeout)
 config.load_verify_locations(cafile='/etc/ssl/certs/ca-certificates.crt')
 q=QuicConnection(configuration=config);h3=H3Connection(q);deadline=time.monotonic()+timeout
 q.connect(dest,now=time.monotonic());sent=False;tls=False;status=b'';body=b'';stream=None
 with control,socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as udp:
  udp.connect(relay)
  while time.monotonic()<deadline:
   now=time.monotonic()
   while (event:=q.next_event()) is not None:
    if isinstance(event,ConnectionTerminated):return {'status':'failed','code':'quic_connection_terminated'}
    if isinstance(event,HandshakeCompleted):tls=event.alpn_protocol in H3_ALPN
    for item in h3.handle_event(event):
     if isinstance(item,HeadersReceived) and item.stream_id==stream:
      status=dict(item.headers).get(b':status',b'')
     if isinstance(item,DataReceived) and item.stream_id==stream:
      body+=item.data
      if len(body)>65536:return {'status':'failed','code':'response_limit'}
      if item.stream_ended:return {'status':'passed' if tls and status==b'200' and transport.dns_ok(body,ident) else 'failed','code':'http3_doh_response','tls':'verified' if tls else 'failed'}
   if tls and not sent:
    stream=q.get_next_available_stream_id();path=b'/dns-query?dns='+base64.urlsafe_b64encode(query).rstrip(b'=')
    h3.send_headers(stream_id=stream,headers=[(b':method',b'GET'),(b':scheme',b'https'),(b':authority',b'cloudflare-dns.com'),(b':path',path),(b'accept',b'application/dns-message')],end_stream=True);sent=True
   for data,_ in q.datagrams_to_send(now=now):udp.send(b'\x00\x00\x00'+transport.address(*dest)+data)
   timer=q.get_timer();wait=max(.001,min(.2,deadline-now,(timer-now) if timer else .2));udp.settimeout(wait)
   try:
    packet=udp.recv(65535)
    if len(packet)<10 or packet[:4]!=b'\x00\x00\x00\x01':continue
    if socket.inet_ntoa(packet[4:8])!=dest[0] or int.from_bytes(packet[8:10],'big')!=443:continue
    q.receive_datagram(packet[10:],dest,now=time.monotonic())
   except socket.timeout:pass
   if timer is not None and timer<=time.monotonic():q.handle_timer(now=time.monotonic())
 return {'status':'failed','code':'http3_timeout'}
if __name__=='__main__':
 try:result=check(float(sys.argv[1]) if len(sys.argv)>1 else 8)
 except Exception:result={'status':'failed','code':'http3_probe_error'}
 print(json.dumps(result))
