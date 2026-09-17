import hashlib,json,pathlib,threading,time
P=pathlib.Path('/root/Config_manager/data/canary_tuning.json');L=threading.Lock();S={'baseline':{'n':0,'ms':0,'ok':0},'canary':{'n':0,'ms':0,'ok':0}}
def group(fp):return 'canary' if int(hashlib.sha256(fp.encode()).hexdigest()[:8],16)%100<5 else 'baseline'
def limits(g):
 if g=='baseline':return 1,6
 try:d=json.loads(P.read_text());return tuple(d.get('candidate_limits',[1,5]))
 except:return 1,5
def record(g,ms,ok):
 with L:
  x=S[g];x['n']+=1;x['ms']+=ms;x['ok']+=int(ok);pathlib.Path('/root/Config_manager/data/canary_results.json').write_text(json.dumps(S))