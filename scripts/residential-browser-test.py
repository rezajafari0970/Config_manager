#!/usr/bin/env python3
import json,os,pathlib,subprocess,tempfile,time
root=pathlib.Path(__file__).resolve().parents[1]
work=pathlib.Path(tempfile.mkdtemp(prefix='p28-residential-browser-',dir=root/'.local'))
subprocess.run(['go','test','-c','-o',str(work/'fixture'),'./internal/panel'],cwd=root,check=True)
env=dict(os.environ,CONFIG_RESIDENTIAL_BROWSER_FIXTURE_DIR=str(work))
with (work/'server.log').open('w') as log:
    process=subprocess.Popen([str(work/'fixture'),'-test.run=^TestResidentialBrowserFixture$'],env=env,stdout=log,stderr=log)
    try:
        for _ in range(100):
            if (work/'public.json').exists():break
            if process.poll() is not None:raise RuntimeError('fixture failed')
            time.sleep(.1)
        result=subprocess.run(['node',str(root/'scripts/residential-browser.cjs')],env=env,capture_output=True,text=True,timeout=90)
        if result.returncode:raise RuntimeError(result.stderr[-3000:])
        receipt=json.loads(result.stdout);receipt['screenshots']=str(work)
        (root/'.local/p28-residential-browser.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt))
    finally:process.terminate();process.wait(timeout=20)
