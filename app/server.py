#!/usr/bin/env python3
import json, os, socket, subprocess, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse
ROOT='/root/Config_manager'; DATA=f'{ROOT}/data/servers.json'
os.makedirs(os.path.dirname(DATA), exist_ok=True)
if not os.path.exists(DATA): open(DATA,'w').write('[]\n')
def load():
    try: return json.load(open(DATA))
    except: return []
def sysinfo():
    def run(c):
        try:return subprocess.check_output(c,shell=True,text=True,stderr=subprocess.DEVNULL,timeout=3).strip()
        except:return 'n/a'
    return {'hostname':socket.gethostname(),'uptime':run('uptime -p'),'disk':run("df -h / | tail -1 | awk '{print $3\" / \"$2\" (\"$5\")\"}'"),'git':run('cd /root/Config_manager && git rev-parse --short HEAD')}
HTML='''<!doctype html><html lang="fa" dir="rtl"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Config Manager</title><style>body{font-family:system-ui;background:#0b1020;color:#eaf0ff;margin:0}.wrap{max-width:1100px;margin:auto;padding:28px}.top{display:flex;justify-content:space-between;align-items:center}.badge{background:#153a2b;color:#75f0aa;padding:8px 12px;border-radius:99px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:14px;margin:22px 0}.card{background:#151d32;border:1px solid #283653;border-radius:16px;padding:18px}.muted{color:#91a0bd}.big{font-size:24px;font-weight:700;margin-top:8px}table{width:100%;border-collapse:collapse}td,th{padding:12px;border-bottom:1px solid #283653;text-align:right}button{background:#5b7cfa;color:white;border:0;border-radius:10px;padding:10px 14px}h1{margin:0}</style></head><body><div class="wrap"><div class="top"><div><h1>Config Manager</h1><div class="muted">پنل مدیریت خودکار سرورها و کانفیگ‌ها</div></div><span class="badge">● Online</span></div><div id="cards" class="grid"></div><div class="card"><div class="top"><h2>سرورها</h2><button onclick="refresh()">بروزرسانی</button></div><table><thead><tr><th>نام</th><th>IP</th><th>وضعیت</th><th>نقش</th></tr></thead><tbody id="servers"></tbody></table></div></div><script>async function refresh(){let r=await fetch('/api/status'),d=await r.json();cards.innerHTML=`<div class=card><div class=muted>Hostname</div><div class=big>${d.system.hostname}</div></div><div class=card><div class=muted>Uptime</div><div class=big>${d.system.uptime}</div></div><div class=card><div class=muted>Disk</div><div class=big>${d.system.disk}</div></div><div class=card><div class=muted>Git</div><div class=big>${d.system.git}</div></div>`;servers.innerHTML=d.servers.map(x=>`<tr><td>${x.name}</td><td>${x.ip}</td><td>${x.status}</td><td>${x.role}</td></tr>`).join('')||'<tr><td colspan=4 class=muted>هنوز سروری ثبت نشده</td></tr>'}refresh();setInterval(refresh,15000)</script></body></html>'''
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        p=urlparse(self.path).path
        if p=='/api/status':
            b=json.dumps({'ok':True,'system':sysinfo(),'servers':load(),'time':int(time.time())},ensure_ascii=False).encode(); self.send_response(200);self.send_header('Content-Type','application/json; charset=utf-8');self.send_header('Content-Length',str(len(b)));self.end_headers();self.wfile.write(b)
        elif p=='/health':
            b=b'{"ok":true}';self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(b)
        else:
            b=HTML.encode();self.send_response(200);self.send_header('Content-Type','text/html; charset=utf-8');self.send_header('Content-Length',str(len(b)));self.end_headers();self.wfile.write(b)
    def log_message(self,fmt,*args): print('%s - %s'%(self.address_string(),fmt%args),flush=True)
ThreadingHTTPServer(('0.0.0.0',4040),H).serve_forever()
