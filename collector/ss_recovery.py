import tempfile,subprocess,os
from .ss_classifier import classify
from .validator import validate,hard_errors
from .vless_adapter import xray,singbox
from .singbox_core import check
def candidate(raw):return 'vless://'+raw.split('://',1)[1]
def assess(raw):
 cl=classify(raw)
 if cl['class']!='vless-like-mislabeled':return {'recoverable':False,'reason':'not-candidate'}
 v=candidate(raw);issues=validate('vless',v)
 if hard_errors(issues):return {'recoverable':False,'reason':'validator-hard','issues':[x['code'] for x in issues]}
 f=tempfile.NamedTemporaryFile('w',delete=False,suffix='.json');f.write(xray(v));f.close()
 try:p=subprocess.run(['/usr/bin/timeout','2s','/usr/local/bin/xray','run','-test','-c',f.name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=2.5);xo=p.returncode==0
 except:xo=False
 finally:os.unlink(f.name)
 so=check(singbox(v))['ok'];return {'recoverable':xo and so,'validator_ok':True,'xray_ok':xo,'singbox_ok':so,'candidate':v if xo and so else None}