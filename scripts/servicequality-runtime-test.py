#!/usr/bin/env python3
"""Loopback-only Xray API test: isolate categories, preserve existing streams,
block new flows at cold start/revocation. Not an internet performance benchmark.
"""
import argparse, http.server, json, pathlib, socket, struct, subprocess, threading, time
from benchmarks.core import Session
from servicequality.profile import adapt

class Backend(http.server.BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def log_message(self,*args):pass
    def do_GET(self):
        body=self.server.label.encode();self.send_response(200);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
def exact(s,n):
    b=b''
    while len(b)<n:
        chunk=s.recv(n-len(b))
        if not chunk:raise EOFError()
        b+=chunk
    return b
def connect(port,name):
    s=socket.create_connection(('127.0.0.1',port),1);s.settimeout(1);s.sendall(b'\x05\x01\x00');assert exact(s,2)==b'\x05\x00'
    b=name.encode();s.sendall(b'\x05\x01\x00\x03'+bytes([len(b)])+b+struct.pack('!H',80));h=exact(s,4)
    if h[1]!=0:s.close();raise ConnectionError()
    n={1:4,4:16}.get(h[3]);n=n if n is not None else exact(s,1)[0];exact(s,n+2);return s
def get(s):
    s.sendall(b'GET / HTTP/1.1\r\nHost: fixture\r\nConnection: keep-alive\r\n\r\n');h=b''
    while not h.endswith(b'\r\n\r\n'):h+=exact(s,1)
    length=int(next(l.split(b':')[1] for l in h.split(b'\r\n') if l.lower().startswith(b'content-length:')))
    return exact(s,length).decode()
def new(port,name):
    with connect(port,name) as s:return get(s)
def blocked(port,name):
    try:new(port,name);return False
    except (OSError,EOFError,ConnectionError):return True
def main():
    p=argparse.ArgumentParser();p.add_argument('--core',required=True);p.add_argument('--output',required=True);a=p.parse_args()
    servers=[]
    for label in ['A','B']:
        srv=http.server.ThreadingHTTPServer(('127.0.0.1',0),Backend);srv.daemon_threads=True;srv.label=label
        threading.Thread(target=srv.serve_forever,daemon=True).start();servers.append(srv)
    b={'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True}}],
       'outbounds':[{'tag':'proxy-0'+str(i+1),'protocol':'freedom','settings':{'redirect':'127.0.0.1:'+str(s.server_port)}} for i,s in enumerate(servers)]+[{'tag':'block','protocol':'blackhole'}],
       'routing':{'balancers':[{'tag':'bootstrap-ip-balancer','selector':['proxy-01'],'strategy':{'type':'leastLoad'},'fallbackTag':'block'}],
                  'rules':[{'type':'field','network':'tcp,udp','balancerTag':'bootstrap-ip-balancer'}]},
       'burstObservatory':{'subjectSelector':['unused-fixture-'],'pingConfig':{'destination':'https://www.gstatic.com/generate_204','interval':'60s','sampling':1,'timeout':'1s'}}}
    c=adapt(b,api_port=39081);result={}
    try:
        with Session(a.core,c,pathlib.Path(a.output)/'runtime') as s:
            api=s.port_map[39081]
            def override(service,target):
                start=time.perf_counter_ns();p=subprocess.run([a.core,'api','bo','-s=127.0.0.1:'+str(api),'-t=1','-b','sq-'+service,target],capture_output=True,timeout=3)
                assert p.returncode==0,p.stderr.decode();return (time.perf_counter_ns()-start)/1e6
            result['cold_start_blocks']=blocked(s.port,'www.instagram.com');assert result['cold_start_blocks']
            override('instagram','proxy-01');old=connect(s.port,'www.instagram.com');assert get(old)=='A'
            result['switch_api_ms']=override('instagram','proxy-02')
            result['new_flow_B']=new(s.port,'www.instagram.com')=='B';assert result['new_flow_B']
            result['existing_flow_A']=get(old)=='A';assert result['existing_flow_A']
            result['other_service_isolated']=blocked(s.port,'telegram.org');assert result['other_service_isolated']
            override('telegram','proxy-01');result['telegram_A']=new(s.port,'telegram.org')=='A';assert result['telegram_A']
            override('instagram','block');result['revocation_blocks']=blocked(s.port,'www.instagram.com');assert result['revocation_blocks']
            result['revocation_does_not_replay_or_kill_old_flow']=get(old)=='A';old.close();assert result['revocation_does_not_replay_or_kill_old_flow']
            info=subprocess.run([a.core,'api','bi','--json','-s=127.0.0.1:'+str(api),'sq-instagram'],capture_output=True,timeout=3)
            result['readback']=info.stdout.decode();assert info.returncode==0
    finally:
        for srv in servers:srv.shutdown();srv.server_close()
    pathlib.Path(a.output,'result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
if __name__=='__main__':main()
