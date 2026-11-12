#!/usr/bin/env python3
import hashlib, importlib.util, json, pathlib, tempfile, time, unittest
HERE=pathlib.Path(__file__).parent
def load(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
country=load('country','egress-country.py');updater=load('update','egress-country-update.py')
class Countries(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup);self.root=pathlib.Path(self.temp.name)
  self.db=self.root/'ranges.sqlite'
  updater.build(b'8.8.8.0,8.8.8.255,TR\n2001:4860::,2001:4860:ffff:ffff:ffff:ffff:ffff:ffff,MO\n',self.db)
  self.entry=dict(name='fixture',driver='sqlite-ranges',path=str(self.db),sha256=hashlib.sha256(self.db.read_bytes()).hexdigest(),released_at=int(time.time()*1000)-86400000)
  self.config=self.root/'config.json';self.registry=country.Registry();self.addCleanup(self.registry.close)
  self.write([self.entry]);self.registry.refresh(self.config)
 def write(self,providers):self.config.write_text(json.dumps(dict(version=country.VERSION,providers=providers)))
 def test_families_boundaries_and_no_network(self):
  for ip,c in [('8.8.8.0','TR'),('8.8.8.255','TR'),('2001:4860::1','MO')]:self.assertEqual(self.registry.lookup(ip)['country'],c)
  for ip in ['8.8.7.255','8.8.9.0','127.0.0.1','not-an-ip']:
   r=self.registry.lookup(ip);self.assertEqual(r['country'],'');self.assertNotIn(ip,json.dumps(r))
 def test_stale_missing_and_corrupt_fail_closed(self):
  self.assertEqual(self.registry.lookup('8.8.8.8',self.entry['released_at']+country.MAX_AGE_MS+1)['country'],'')
  self.entry['sha256']='0'*64;self.write([self.entry]);self.registry.refresh(self.config)
  self.assertEqual(self.registry.lookup('8.8.8.8')['country'],'')
 def test_provider_conflict(self):
  db=self.root/'other.sqlite';updater.build(b'8.8.8.0,8.8.8.255,MO\n',db)
  other={**self.entry,'name':'other','path':str(db),'sha256':hashlib.sha256(db.read_bytes()).hexdigest()};self.write([self.entry,other]);self.registry.refresh(self.config)
  self.assertEqual(self.registry.lookup('8.8.8.8')['status'],'country_provider_disagreement')
 def test_only_authenticated_ads_attached(self):
  row=dict(ip='8.8.8.8',actual_ads=True,echo=True,tls=True,same_stream=True)
  rows=self.registry.attach([row,{**row,'actual_ads':False},{**row,'tls':False}],self.config)
  self.assertEqual(rows[0]['geo']['country'],'TR');self.assertNotIn('geo',rows[1]);self.assertNotIn('geo',rows[2]);self.assertNotIn('8.8.8.8',json.dumps(rows[0]['geo']))
 def test_bad_csv_rejected(self):
  for index,raw in enumerate([b'8.8.8.9,8.8.8.1,TR\n',b'8.8.8.0,8.8.8.9,TR\n8.8.8.8,8.8.8.10,TR\n',b'8.8.8.0,::1,TR\n']):
   with self.assertRaises(ValueError):updater.build(raw,self.root/str(index))
if __name__=='__main__':unittest.main()
