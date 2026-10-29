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
WORK = ROOT / '.local' / 'p3-browser' / secrets.token_hex(6)
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
with zipfile.ZipFile(WORK / 'configs-raw.zip') as archive:
    assert len(archive.namelist()) == 2
    for name in archive.namelist():
        raw = archive.read(name)
        assert raw.endswith(b'\r\n')
        if name.endswith('.json'):
            assert b'"stats" : []' in raw and b'9007199254740993' in raw
receipt = {'phases': RECEIPTS, 'passed': True, 'tests': ['no_login', 'bulk_import', 'automatic_names', 'Persian_intervals', 'independent_source_status', 'raw_SHA256', 'original_response_bytes', 'global_deduplication', 'edit', 'pause', 'delete', 'restart_persistence', 'fixed_TTL', 'ZIP_exact_bytes', 'mobile_drawer', '1440_and_390_no_overflow']}
(WORK / 'receipt.json').write_text(json.dumps(receipt, indent=2))
print(json.dumps({'receipt': str(WORK / 'receipt.json'), 'passed': True}))
