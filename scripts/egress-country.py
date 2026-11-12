#!/usr/bin/env python3
"""Local country-provider registry. Only transient, authenticated Ads peer IPs
enter lookups. No per-IP network calls, IP logs, or inference from entry/ASN.
"""
import hashlib, ipaddress, json, os, pathlib, re, sqlite3, time

VERSION = 'ads-country.v1'
MAX_AGE_MS = 45 * 86400000
CONFIG = '/etc/config-repair/egress-country.json'

class SQLiteRanges:
    def __init__(self, path):
        self.db = sqlite3.connect(pathlib.Path(path).resolve().as_uri()+'?mode=ro&immutable=1', uri=True)
        self.db.execute('PRAGMA query_only=ON')
    def lookup(self, ip):
        row = self.db.execute('SELECT finish,country FROM ranges WHERE family=? AND start<=? ORDER BY start DESC LIMIT 1', (ip.version, ip.packed)).fetchone()
        return row[1] if row and row[0] >= ip.packed else ''
    def close(self): self.db.close()

class MMDB:
    def __init__(self, path):
        import maxminddb
        self.reader = maxminddb.open_database(path)
    def lookup(self, ip):
        data = self.reader.get(str(ip)) or {}
        country = data.get('country', {})
        return data.get('country_code', '') or (country.get('iso_code', '') if isinstance(country, dict) else '')
    def close(self): self.reader.close()

DRIVERS = {'sqlite-ranges': SQLiteRanges, 'mmdb': MMDB}

class Registry:
    def __init__(self): self.key=None; self.providers=[]; self.error='unconfigured'; self.retry_at=0
    def close(self):
        for entry, driver in self.providers: driver.close()
        self.providers=[]
    def refresh(self, config=CONFIG):
        if time.monotonic()<self.retry_at:return
        try:
            p=pathlib.Path(config); st=p.stat(); key=(str(p), st.st_ino, st.st_mtime_ns, st.st_size)
            if key==self.key: return
            self.close(); self.key=None
            if st.st_size>16384: raise ValueError('manifest_limit')
            manifest=json.loads(p.read_text())
            if manifest.get('version')!=VERSION or not 1<=len(manifest.get('providers',[]))<=4: raise ValueError('manifest_invalid')
            seen=set()
            for entry in manifest['providers']:
                name=entry['name']
                if not re.fullmatch('[a-z0-9-]{1,48}',name) or name in seen: raise ValueError('provider_invalid')
                seen.add(name); path=pathlib.Path(entry['path'])
                if not path.is_absolute() or not path.is_file() or path.stat().st_size>256*1024*1024: raise ValueError('dataset_invalid')
                with path.open('rb') as f: digest=hashlib.file_digest(f,'sha256').hexdigest()
                if digest!=entry['sha256'] or not isinstance(entry['released_at'],int): raise ValueError('dataset_mismatch')
                self.providers.append((entry,DRIVERS[entry['driver']](path)))
            self.key=key; self.error=''
        except Exception:
            self.close(); self.key=None; self.error='country_provider_unavailable'; self.retry_at=time.monotonic()+30
    def lookup(self, value, at=None):
        at=int(time.time()*1000) if at is None else at
        result={'version':VERSION,'status':'unknown','country':'','at':at,'providers':[]}
        try:
            try:ip=ipaddress.ip_address(value)
            except ValueError:raise ValueError('invalid_source') from None
            if not ip.is_global or ip.is_multicast or ip.is_reserved: raise ValueError('nonpublic_source')
            if self.error or not self.providers: raise ValueError(self.error or 'unconfigured')
            countries=set()
            for entry,driver in self.providers:
                release=entry['released_at']
                if release>at+86400000 or at-release>MAX_AGE_MS: raise ValueError('country_data_stale')
                country=driver.lookup(ip)
                if not isinstance(country,str) or not re.fullmatch('[A-Z]{2}',country) or country in ('ZZ','XX','EU','AP'): raise ValueError('country_not_found')
                countries.add(country)
                result['providers'].append({'name':entry['name'],'dataset':entry['sha256'],'released_at':release,'country':country})
            if len(countries)!=1: raise ValueError('country_provider_disagreement')
            result.update(status='located',country=countries.pop())
        except Exception as error:
            allowed={'invalid_source','nonpublic_source','unconfigured','country_provider_unavailable','country_data_stale','country_not_found','country_provider_disagreement'}
            result['status']=str(error) if isinstance(error,ValueError) and str(error) in allowed else 'country_lookup_failed'
            result['country']=''
        return result
    def attach(self, rows, config=CONFIG):
        self.refresh(config); cache={}
        for row in rows:
            if not row.get('actual_ads') or not all(row.get(k) for k in ('echo','tls','same_stream')): continue
            value=row.get('ip','')
            if value not in cache:cache[value]=self.lookup(value)
            row['geo']=cache[value]
        return rows
