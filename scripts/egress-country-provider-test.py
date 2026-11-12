#!/usr/bin/env python3
import importlib.util,io,json,pathlib,tempfile,time,unittest
from unittest.mock import patch
HERE=pathlib.Path(__file__).parent
def load(name,file):
 s=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
u=load('providers_update','egress-country-providers-update.py');g=load('country','egress-country.py')
class Providers(unittest.TestCase):
 def test_cidr_import_sorts_and_checks_boundaries(self):
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d)/'db.sqlite'
   n=u.build_cidr(io.StringIO('network,country_code\n2001:4860::/32,MO\n8.8.8.0/24,TW\n1.1.1.0/24,TR\n'),p);self.assertEqual(n,3)
   driver=g.SQLiteRanges(p)
   try:
    import ipaddress
    self.assertEqual(driver.lookup(ipaddress.ip_address('8.8.8.255')),'TW');self.assertEqual(driver.lookup(ipaddress.ip_address('8.8.9.0')),'')
   finally:driver.close()
 def test_cidr_overlap_and_wrong_fields_rejected(self):
  for raw in ['network,country_code\n8.8.8.0/24,TR\n8.8.8.0/25,TW\n','wrong,country_code\n8.8.8.0/24,TR\n','network,country_code\n8.8.8.1/24,TR\n']:
   with tempfile.TemporaryDirectory() as d,self.assertRaises(Exception):u.build_cidr(io.StringIO(raw),pathlib.Path(d)/'db')
 def test_both_refresh_failures_retain_original_provider_pins_and_age(self):
  with tempfile.TemporaryDirectory() as d:
   root=pathlib.Path(d);config=root/'manifest.json';old={'version':'ads-country.v2','geojs':True,'providers':[{'name':'dbip-country-lite','lineage':'dbip','path':'/private/db','sha256':'a'*64,'released_at':123}]};config.write_text(json.dumps(old))
   with patch.object(u.dbip,'update',side_effect=TimeoutError()),patch.object(u,'iplocate',side_effect=TimeoutError()),patch('builtins.print'):
    u.update(root,config)
   now=json.loads(config.read_text());self.assertEqual(now,old);self.assertTrue((root/'update-status.json').exists())
 def test_unknown_remote_addresses_cannot_be_substituted(self):
  class Response:
   def __enter__(self):return self
   def __exit__(self,*a):pass
   def read(self,n):return b'[{"ip":"1.1.1.1","country":"MO"}]'
  class Opener:
   def open(self,*a,**k):return Response()
  r=g.Registry()
  with patch.object(g.urllib.request,'build_opener',return_value=Opener()):self.assertEqual(r.remote_lookup(['8.8.8.8']),{})
  self.assertEqual(len(r.cache),0)
 def test_remote_failure_has_bounded_backoff(self):
  r=g.Registry()
  with patch.object(g.urllib.request,'build_opener',side_effect=TimeoutError()) as request:
   self.assertEqual(r.remote_lookup(['8.8.8.8']),{});self.assertEqual(r.remote_lookup(['8.8.8.8']),{});self.assertEqual(request.call_count,1)
  self.assertGreater(r.remote_retry,time.monotonic())
if __name__=='__main__':unittest.main()
