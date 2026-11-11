#!/usr/bin/env python3
"""Closed-network qualification of the generated relay, on both target cores.
Synthetic credentials only. Separate server process: final client bytes unchanged.
TCP + UDP + UDP/443 echo; TLS/HTTP3/Android are separate evidence dimensions.
"""
import importlib.util,json,pathlib,secrets,subprocess,sys,tempfile,time
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('lab',HERE/'live-probe.py')
lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
def run():
    report={}
    for version,sha in [('26.10.31','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'),('26.11.9','e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d')]:
        core=pathlib.Path('/opt/config-repair/core-'+version+'/xray');assert lab.digest(core.read_bytes())==sha
        header={'type':'http','request':{'version':'1.1','method':'GET','path':['/'],'headers':{'Host':['fast.com']}}}
        user='11111111-1111-4111-8111-111111111111'
        client={'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'auth':'noauth','udp':True}}],'outbounds':[{'tag':'proxy','protocol':'vless','settings':{'address':'203.0.113.81','port':24443,'id':user,'encryption':'none'},'streamSettings':{'network':'tcp','security':'none','tcpSettings':{'header':header}}},{'tag':'block','protocol':'blackhole','settings':{}}],'routing':{'rules':[{'type':'field','network':'tcp,udp','outboundTag':'proxy'}]}}
        client['log']={'loglevel':'debug'}
        compiled=subprocess.run([str(HERE.parent/'.local/config-compat')],input=json.dumps(client).encode(),capture_output=True,check=True).stdout
        server={'inbounds':[{'tag':'fixture','listen':'203.0.113.81','port':24443,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':user}]},'streamSettings':{'network':'tcp','security':'none','tcpSettings':{'header':{'type':'http'}}}}],'outbounds':[{'protocol':'freedom','settings':{}}]}
        server['log']={'loglevel':'debug'}
        server['outbounds'][0]['settings']={'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445'}]}
        with lab.Sandbox(str(core),secrets.token_hex(3),8200,prefix='ctf') as box:
            box.net([lab.IP,'addr','add','203.0.113.81/32','dev','lo'])
            box.net([lab.NFT,'-f','-'],input=b'flush ruleset\ntable inet guard { chain out { type filter hook output priority 0; policy drop; oifname "lo" accept; }; chain in { type filter hook input priority 0; policy drop; iifname "lo" accept; }; }\n')
            # Trusted synthetic server is bounded by the same cgroup/netns. Config
            # has no user input; client still uses the capability-free mount.
            path=box.work/'fixture.json';path.write_text(json.dumps(server))
            prefix=[lab.IP,'netns','exec',box.name]
            check=box.launch(compiled,True);check.wait(timeout=10);assert check.returncode==0,box.logs()
            box.stop()
            fixture=subprocess.Popen(['/usr/bin/python3',str(HERE/'live-probe.py'),'--sandbox-launch',str(box.cg)]+prefix+[str(core),'run','-c',str(path)],stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
            echo=subprocess.Popen(prefix+[sys.executable,str(HERE/'traffic-policy-test.py'),'--echo'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            try:
                process=box.launch(compiled)
                for _ in range(50):
                    assert process.poll() is None,box.logs()
                    ready=box.net([sys.executable,'-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'])
                    if ready.stdout.strip()==b'0':break
                    time.sleep(.05)
                try:response=box.net([sys.executable,str(HERE/'traffic-policy-test.py'),'--exercise'],timeout=20)
                except subprocess.CalledProcessError as e:
                    fixture.terminate();logs=fixture.communicate(timeout=3)[0]
                    raise RuntimeError(e.stderr.decode()+box.logs().decode()+logs.decode()) from None
                tests=json.loads(response.stdout);assert all(tests.values()),tests
                report[version]={'config_sha256':lab.digest(compiled),'runtime_sha256':lab.digest((box.work/'config.json').read_bytes()),'core':'passed',**tests}
            finally:
                for p in [fixture,echo]:
                    p.terminate()
                    try:p.wait(timeout=3)
                    except subprocess.TimeoutExpired:p.kill();p.wait()
    print(json.dumps({'driver':'legacy-vless-tcp-loopback.v1','passed':True,'tests':report,'https':'separate_test_required','http3':'not_run','android':'not_run'}))
if __name__=='__main__':run()
