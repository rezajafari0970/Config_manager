#!/usr/bin/env python3
"""Atomic, independently scheduled DB-IP Lite updater; CC BY 4.0 attribution
is shown in the panel. Download via verified TLS, validate the publisher's raw
CSV SHA1 and all ranges, then pin the immutable local SQLite file with SHA256.
"""
import argparse, csv, datetime, fcntl, gzip, hashlib, io, json, os, pathlib, re, sqlite3, tempfile, urllib.request, urllib.parse
PAGE='https://db-ip.com/db/download/ip-to-country-lite'
HEADERS={'User-Agent':'Shen-Config-Country/1.0','Referer':PAGE}

def download(url, maximum):
    with urllib.request.urlopen(urllib.request.Request(url,headers=HEADERS),timeout=45) as response:
        if urllib.parse.urlsplit(response.url).scheme!='https' or urllib.parse.urlsplit(response.url).hostname not in ('db-ip.com','download.db-ip.com'):raise ValueError('redirect_host')
        data=response.read(maximum+1)
    if len(data)>maximum:raise ValueError('download_limit')
    return data

def build(raw, path):
    import ipaddress
    db=sqlite3.connect(path); db.execute('CREATE TABLE ranges(family INTEGER,start BLOB,finish BLOB,country TEXT,PRIMARY KEY(family,start)) WITHOUT ROWID')
    previous={};rows=[];count=0
    try:
        for record in csv.reader(io.StringIO(raw.decode('utf-8'))):
            if len(record)!=3:raise ValueError('csv_columns')
            start,finish=map(ipaddress.ip_address,record[:2]);country=record[2]
            if start.version!=finish.version or int(start)>int(finish) or int(start)<=previous.get(start.version,-1):raise ValueError('overlapping_ranges')
            if not re.fullmatch('[A-Z]{2}',country):raise ValueError('country_code')
            previous[start.version]=int(finish);rows.append((start.version,start.packed,finish.packed,country));count+=1
            if count>2000000:raise ValueError('record_limit')
            if len(rows)>=4096:db.executemany('INSERT INTO ranges VALUES(?,?,?,?)',rows);rows=[]
        db.executemany('INSERT INTO ranges VALUES(?,?,?,?)',rows);db.commit()
        if db.execute('PRAGMA quick_check').fetchone()[0]!='ok':raise ValueError('database_integrity')
    finally:db.close()
    return count

def update(directory, config):
    os.umask(0o077);directory=pathlib.Path(directory);directory.mkdir(mode=0o700,parents=True,exist_ok=True)
    config=pathlib.Path(config);config.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
    with (directory/'update.lock').open('a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        html=download(PAGE,2*1024*1024).decode()
        links=re.findall(r'https://download\.db-ip\.com/free/dbip-country-lite-(\d{4}-\d{2})\.csv\.gz',html)
        month=max(links);url='https://download.db-ip.com/free/dbip-country-lite-'+month+'.csv.gz'
        hashes=re.findall(r'\b[a-f0-9]{40}\b',html[max(0,html.index(url)-5000):html.index(url)])
        if not hashes:raise ValueError('publisher_checksum_missing')
        released=int(datetime.datetime.strptime(month,'%Y-%m').replace(tzinfo=datetime.timezone.utc).timestamp()*1000)
        current=int(datetime.datetime.now(datetime.timezone.utc).timestamp()*1000)
        if released>current+86400000 or current-released>45*86400000:raise ValueError('publisher_release_stale')
        if config.exists():
            existing=json.loads(config.read_text())
            for entry in existing.get('providers',[]):
                p=pathlib.Path(entry.get('path',''))
                if entry.get('name')=='dbip-country-lite' and entry.get('publisher_sha1')==hashes[-1] and p.is_file():
                    with p.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
                    if digest==entry.get('sha256'):print(json.dumps({'status':'current','release':month}));return
        compressed=download(url,24*1024*1024)
        with gzip.GzipFile(fileobj=io.BytesIO(compressed)) as stream:raw=stream.read(128*1024*1024+1)
        if len(raw)>128*1024*1024 or hashlib.sha1(raw).hexdigest()!=hashes[-1]:raise ValueError('publisher_checksum_mismatch')
        with tempfile.TemporaryDirectory(prefix='build-',dir=directory) as work:
            temp=pathlib.Path(work)/'country.sqlite';count=build(raw,temp)
            if count<100000:raise ValueError('incomplete_dataset')
            with temp.open('rb') as f:sha=hashlib.file_digest(f,'sha256').hexdigest()
            destination=directory/('dbip-'+month+'-'+sha[:16]+'.sqlite');os.replace(temp,destination)
            entry=dict(name='dbip-country-lite',lineage='dbip',driver='sqlite-ranges',path=str(destination.resolve()),sha256=sha,released_at=released,publisher_sha1=hashes[-1],source_url=url,license='CC-BY-4.0')
            providers=[]
            if config.exists():providers=[p for p in json.loads(config.read_text()).get('providers',[]) if p.get('name')!='dbip-country-lite']
            manifest=json.loads(config.read_text()) if config.exists() else {'version':'ads-country.v2'}
            manifest['providers']=[entry]+providers
            staged=config.with_name(config.name+'.new');staged.write_text(json.dumps(manifest,indent=2));os.chmod(staged,0o600);os.replace(staged,config)
        # Old immutable datasets are small and retained for in-flight readers.
        print(json.dumps({'status':'updated','release':month,'records':count,'sha256':sha}))

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--directory',default='/var/lib/config-repair/country');p.add_argument('--config',default='/etc/config-repair/egress-country.json');a=p.parse_args();update(a.directory,a.config)
