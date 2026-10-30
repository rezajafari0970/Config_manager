#!/usr/bin/env python3
"""Passwordless UI + isolated synthetic collector acceptance; never uses live sources."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parent.parent
WORK = ROOT / '.local' / 'p7-browser' / secrets.token_hex(6)
WORK.mkdir(parents=True, mode=0o700)
subprocess.run(['go', 'test', '-c', '-o', str(WORK / 'fixture-server'), './internal/panel'], cwd=ROOT, check=True)
ENV = dict(os.environ, PANEL_TEST_DIR=str(WORK), CONFIG_PANEL_BROWSER_FIXTURE_DIR=str(WORK))
RECEIPTS = []
for phase in ('1', '2'):
    with (WORK / ('server-' + phase + '.log')).open('w') as log:
        process = subprocess.Popen([str(WORK / 'fixture-server'), '-test.run=^TestBrowserFixtureServer$', '-test.v'], env=ENV, stdout=log, stderr=log)
        try:
            for _ in range(100):
                if process.poll() is not None:
                    raise RuntimeError('fixture exited before readiness')
                try:
                    with urllib.request.urlopen('http://127.0.0.1:18192/healthz', timeout=1) as response:
                        if response.status == 200:
                            break
                except OSError:
                    time.sleep(.1)
            else:
                raise RuntimeError('fixture not ready')
            result = subprocess.run(['node', str(ROOT / 'scripts' / 'panel-browser.cjs'), phase], cwd=ROOT, env=ENV, capture_output=True, text=True, timeout=150)
            if result.returncode:
                raise RuntimeError(result.stderr[-6000:])
            RECEIPTS.append(json.loads(result.stdout.strip()))
        finally:
            process.terminate()
            try:
                process.wait(timeout=20)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise RuntimeError('fixture shutdown failed')
        if process.returncode != 0:
            raise RuntimeError('fixture stopped with nonzero status')
receipt = {'phases': RECEIPTS, 'passed': True, 'tests':['input_bulk_dedup_status','jsonl_one_object_per_line','common_template','bank_routes_removed','mobile_and_desktop_no_overflow','restart_persistence','TTL_removes_conversions','conversion_settings','source_timeout']}
(WORK / 'receipt.json').write_text(json.dumps(receipt, indent=2))
print(json.dumps({'receipt': str(WORK / 'receipt.json'), 'passed': True}))
