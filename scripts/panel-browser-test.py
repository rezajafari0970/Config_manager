#!/usr/bin/env python3
"""Isolated CRUD/auth/restart/browser acceptance. Requires node, Chrome, playwright-core."""
import base64
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
WORK = ROOT / '.local' / 'panel-browser' / secrets.token_hex(6)
WORK.mkdir(parents=True, mode=0o700)
CONFIG = WORK / 'panel.json'
CONFIG.write_text(json.dumps({'listen': '127.0.0.1:18192', 'public_origin': 'http://127.0.0.1:18192', 'database': str(WORK / 'data' / 'panel.db'), 'master_key': base64.b64encode(secrets.token_bytes(32)).decode()}))
CONFIG.chmod(0o600)
PASSWORD = WORK / 'password'
PASSWORD.write_text('synthetic-browser-password')
PASSWORD.chmod(0o600)
COMMAND = [str(ROOT / 'bin' / 'config-panel'), '-config', str(CONFIG)]
subprocess.run(COMMAND + ['-init-admin', '-password-file', str(PASSWORD)], check=True, capture_output=True)
ENV = dict(os.environ, PANEL_TEST_DIR=str(WORK))
RECEIPTS = []
for phase in ('1', '2'):
    with (WORK / ('server-' + phase + '.log')).open('w') as log:
        process = subprocess.Popen(COMMAND, stdout=log, stderr=log)
        try:
            for _ in range(100):
                if process.poll() is not None:
                    raise RuntimeError('panel exited before readiness')
                try:
                    with urllib.request.urlopen('http://127.0.0.1:18192/healthz', timeout=1) as response:
                        if response.status == 200:
                            break
                except OSError:
                    time.sleep(.1)
            else:
                raise RuntimeError('panel did not become ready')
            result = subprocess.run(['node', str(ROOT / 'scripts' / 'panel-browser.cjs'), phase], cwd=ROOT, env=ENV, capture_output=True, text=True, timeout=120)
            if result.returncode:
                raise RuntimeError(result.stderr[-5000:])
            RECEIPTS.append(json.loads(result.stdout.strip()))
        finally:
            process.terminate()
            try:
                process.wait(timeout=20)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise RuntimeError('graceful shutdown failed')
        if process.returncode != 0:
            raise RuntimeError('panel shutdown returned nonzero')
(WORK / 'receipt.json').write_text(json.dumps({'phases': RECEIPTS, 'tests': ['CRUD', 'pause_resume', 'redaction', '1440px_390px_no_overflow', 'session_and_source_survive_process_restart', 'password_change', 'logout'], 'passed': True}, indent=2))
print(json.dumps({'receipt': str(WORK / 'receipt.json'), 'passed': True}))
