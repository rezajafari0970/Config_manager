#!/usr/bin/env python3
"""Provider-isolated, atomic country refresh. Failures preserve last-good files."""
import argparse,csv,datetime,fcntl,hashlib,importlib.util,io,ipaddress,json,os,pathlib,re,sqlite3,tempfile,urllib.request,urllib.parse,zipfile
HERE=pathlib.Path(__file__).parent
spec=importlib.util.spec_from_file_location('dbip_update',HERE/'egress-country-update.py');dbip=importlib.util.module_from_spec(spec);spec.loader.exec_module(dbip)
HOSTS={'api.github.com','raw.githubusercontent.com','media.githubusercontent.com'}
def download(url,maximum):
 class Redirect(urllib.request.HTTPRedirectHandler):
  def redirect_request(self,req,fp,code,msg,headers,newurl):
   p=urllib.parse.urlsplit(newurl)
   if p.scheme!='https' or p.hostname not in HOSTS:raise ValueError('redirect_host')
   return super().redirect_request(req,fp,code,msg,headers,newurl)
 p=urllib.parse.urlsplit(url)
 if p.scheme!='https' or p.hostname not in HOSTS:raise ValueError('source_host')
 opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),Redirect())
 with opener.open(urllib.request.Request(url,headers={'User-Agent':'Shen-Ads-Country/2.0'}),timeout=40) as r:data=r.read(maximum+1)
 if len(data)>maximum:raise ValueError('download_limit')
 return data

def build_cidr(stream,path):
 db=sqlite3.connect(path);db.execute('CREATE TABLE ranges(family INTEGER,start BLOB,finish BLOB,country TEXT,PRIMARY KEY(family,start)) WITHOUT ROWID');db.execute('PRAGMA cache_size=-8192')
 count=0;rows=[]
 try:
  reader=csv.DictReader(stream)
  if not {'network','country_code'}.issubset(reader.fieldnames or []):raise ValueError('csv_fields')
  for r in reader:
   n=ipaddress.ip_network(r['network'],strict=True);c=r['country_code']
   if not re.fullmatch('[A-Z]{2}',c):raise ValueError('country_code')
   rows.append((n.version,n.network_address.packed,n.broadcast_address.packed,c));count+=1
   if count>4000000:raise ValueError('record_limit')
   if len(rows)>=4096:db.executemany('INSERT INTO ranges VALUES(?,?,?,?)',rows);rows=[]
  db.executemany('INSERT INTO ranges VALUES(?,?,?,?)',rows);db.commit()
  previous={}
  for family,start,finish in db.execute('SELECT family,start,finish FROM ranges ORDER BY family,start'):
   if family in previous and start<=previous[family]:raise ValueError('overlapping_ranges')
   previous[family]=finish
  if db.execute('PRAGMA quick_check').fetchone()[0]!='ok':raise ValueError('integrity')
 finally:db.close()
 return count

def save_manifest(config,m):
 staged=config.with_name(config.name+'.new');staged.write_text(json.dumps(m,indent=2));os.chmod(staged,0o600);os.replace(staged,config)
def iplocate(directory,config):
 api='https://api.github.com/repos/iplocate/ip-address-databases/commits?path=ip-to-country/ip-to-country.csv.zip&per_page=1'
 commit=json.loads(download(api,262144))[0];sha=commit['sha']
 if not re.fullmatch('[a-f0-9]{40}',sha):raise ValueError('commit')
 released=int(datetime.datetime.fromisoformat(commit['commit']['committer']['date'].replace('Z','+00:00')).timestamp()*1000)
 now=int(datetime.datetime.now(datetime.timezone.utc).timestamp()*1000)
 if released>now+86400000 or now-released>45*86400000:raise ValueError('publisher_stale')
 relative='/iplocate/ip-address-databases/'+sha+'/ip-to-country/ip-to-country.csv.zip'
 pointer=download('https://raw.githubusercontent.com'+relative,1024).decode()
 match=re.fullmatch(r'version https://git-lfs.github.com/spec/v1\noid sha256:([a-f0-9]{64})\nsize (\d+)\n?',pointer)
 if not match or int(match[2])>32*1024*1024:raise ValueError('lfs_pointer')
 m=json.loads(config.read_text()) if config.exists() else {'providers':[]}
 for e in m['providers']:
  path=pathlib.Path(e.get('path',''))
  if e.get('name')=='iplocate-country' and e.get('publisher_sha256')==match[1] and path.is_file():
   with path.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
   if digest==e.get('sha256'):return {'status':'current','commit':sha}
 raw=download('https://media.githubusercontent.com/media'+relative,32*1024*1024)
 if len(raw)!=int(match[2]) or hashlib.sha256(raw).hexdigest()!=match[1]:raise ValueError('publisher_checksum')
 with tempfile.TemporaryDirectory(prefix='iplocate-build-',dir=directory) as folder:
  path=pathlib.Path(folder)/'country.sqlite'
  with zipfile.ZipFile(io.BytesIO(raw)) as archive:
   members=[x for x in archive.infolist() if not x.is_dir() and x.filename.endswith('.csv')]
   if len(members)!=1 or members[0].file_size>256*1024*1024:raise ValueError('archive_limit')
   with archive.open(members[0]) as stream:count=build_cidr(io.TextIOWrapper(stream,encoding='utf-8-sig'),path)
  if count<100000:raise ValueError('incomplete_dataset')
  with path.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
  dest=directory/('iplocate-'+digest[:16]+'.sqlite');os.replace(path,dest)
  entry=dict(name='iplocate-country',lineage='iplocate',driver='sqlite-ranges',path=str(dest.resolve()),sha256=digest,released_at=released,publisher_sha256=match[1],source_url='https://github.com/iplocate/ip-address-databases',commit=sha,license='CC-BY-SA-4.0')
  m['providers']=[x for x in m['providers'] if x['name']!='iplocate-country']+[entry];save_manifest(config,m)
 return {'status':'updated','records':count,'commit':sha,'sha256':digest}
def update(directory,config):
 os.umask(0o077);directory=pathlib.Path(directory);directory.mkdir(parents=True,exist_ok=True,mode=0o700);config=pathlib.Path(config);config.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
 status={}
 try:dbip.update(directory,config);status['dbip']='ready'
 except Exception:status['dbip']='refresh_failed_last_good_retained'
 with (directory/'update.lock').open('a') as lock:
  fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
  try:status['iplocate']=iplocate(directory,config)
  except Exception as e:status['iplocate']={'status':'refresh_failed_last_good_retained','code':str(e) if isinstance(e,ValueError) and re.fullmatch('[a-z_]{1,40}',str(e)) else type(e).__name__}
  m=json.loads(config.read_text()) if config.exists() else {'providers':[]}
  m.update(version='ads-country.v2',geojs=True)
  for e in m['providers']:e.setdefault('lineage','dbip' if e['name']=='dbip-country-lite' else e['name'])
  if m['providers']:save_manifest(config,m)
  status['checked_at']=int(datetime.datetime.now(datetime.timezone.utc).timestamp()*1000)
  p=directory/'update-status.json';tmp=p.with_suffix('.new');tmp.write_text(json.dumps(status));os.replace(tmp,p)
  # Keep current and one previous dataset per provider; open immutable readers
  # retain descriptors. Never remove any path referenced by the manifest.
  active={x.get('path') for x in m['providers']}
  for prefix in ('dbip-','iplocate-'):
   files=sorted(directory.glob(prefix+'*.sqlite'),key=lambda x:x.stat().st_mtime,reverse=True)
   for p in files[2:]:
    if str(p.resolve()) not in active:p.unlink()
 print(json.dumps(status))
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--directory',default='/var/lib/config-repair/country');p.add_argument('--config',default='/etc/config-repair/egress-country.json');a=p.parse_args();update(a.directory,a.config)
