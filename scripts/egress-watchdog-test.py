#!/usr/bin/env python3
"""Fault injection against a private transient unit, never production."""
import json, os, pathlib, signal, subprocess, time
ROOT=pathlib.Path(__file__).resolve().parents[1]
unit="shen-egress-watchdog-test-"+str(os.getpid())
work=ROOT/".local"/unit
work.mkdir(mode=0o700)
def props():
    text=subprocess.check_output(["systemctl","show",unit,"-p","MainPID","-p","NRestarts","-p","ActiveState"],text=True)
    return dict(line.split("=",1) for line in text.splitlines())
def wait(predicate,timeout=30):
    until=time.monotonic()+timeout
    while time.monotonic()<until:
        p=props()
        if predicate(p):return p
        time.sleep(.2)
    raise RuntimeError("private watchdog test deadline")
try:
    command=["systemd-run","--quiet","--unit="+unit,"--service-type=notify",
       "-p","NotifyAccess=main","-p","WatchdogSec=10","-p","WatchdogSignal=SIGKILL",
       "-p","Restart=on-failure","-p","RestartSec=1","-p","StartLimitIntervalSec=0",
       "-p","MemoryMax=256M","-p","TasksMax=128","-p","TimeoutStopSec=10",
       str(ROOT/"bin/config-egress-worker"),"--socket",str(work/"absent.sock"),
       "--helper",str(ROOT/"scripts/egress-probe.py"),"--prefix","cgt",
       "--lock-file",str(work/"worker.lock"),"--workers","1"]
    subprocess.run(command,check=True)
    before=wait(lambda p:p["ActiveState"]=="active" and int(p["MainPID"])>0)
    pid=int(before["MainPID"])
    assert pathlib.Path("/proc/"+str(pid)+"/exe").resolve()==(ROOT/"bin/config-egress-worker").resolve()
    begin=time.monotonic()
    os.kill(pid,signal.SIGSTOP)
    after=wait(lambda p:p["ActiveState"]=="active" and int(p["MainPID"])>0 and int(p["MainPID"])!=pid and int(p["NRestarts"])>=1)
    receipt=dict(passed=True,seconds=round(time.monotonic()-begin,3),before=before,after=after,scope="isolated transient unit; SIGSTOP; watchdog SIGKILL and restart")
    (ROOT/".local/watchdog.json").write_text(json.dumps(receipt,indent=2))
    print(json.dumps(receipt))
finally:
    subprocess.run(["systemctl","stop",unit],check=False,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    subprocess.run(["systemctl","reset-failed",unit],check=False,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    subprocess.run(["python3",str(ROOT/"scripts/egress-probe.py"),"--prefix","cgt","--cleanup","--slot","0"],check=True)
