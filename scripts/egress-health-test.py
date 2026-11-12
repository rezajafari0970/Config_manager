#!/usr/bin/env python3
import importlib.util,pathlib,unittest
spec=importlib.util.spec_from_file_location('health',pathlib.Path(__file__).with_name('egress-health.py'));h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
class Health(unittest.TestCase):
 def test_full_transfer_only(self):
  good=dict(curl_exit=0,tls_verified=True,http_status=200,bytes=65536)
  self.assertTrue(h.passed(good))
  for patch in [dict(curl_exit=28),dict(tls_verified=False),dict(http_status=403),dict(bytes=65535)]:self.assertFalse(h.passed({**good,**patch}))
if __name__=='__main__':unittest.main()
