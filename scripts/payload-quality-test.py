#!/usr/bin/env python3
import importlib.util,json,pathlib,subprocess,tempfile,unittest
from types import SimpleNamespace
from unittest.mock import patch
s=importlib.util.spec_from_file_location('quality',pathlib.Path(__file__).with_name('payload-quality.py'));q=importlib.util.module_from_spec(s);s.loader.exec_module(q)
class QualityTests(unittest.TestCase):
 def test_http_200_is_not_completed_payload(self):
  good={'ssl_verify_result':0,'http_code':200,'size_download':q.DOWNLOAD}
  self.assertTrue(q.accepted(0,good,{200},'size_download',q.DOWNLOAD))
  for rc,row in [(28,good),(0,{**good,'size_download':q.DOWNLOAD-1}),(0,{**good,'ssl_verify_result':20}),(0,{**good,'http_code':302}),(0,{**good,'http_code':429})]:
   self.assertFalse(q.accepted(rc,row,{200},'size_download',q.DOWNLOAD))
 def test_upload_full_size_ack_and_tls(self):
  with tempfile.TemporaryDirectory() as folder:
   p=pathlib.Path(folder)/'upload';p.write_bytes(b'x'*q.UPLOAD)
   for rc,size,ok in [(0,q.UPLOAD,True),(28,q.UPLOAD,False),(0,65536,False)]:
    metrics={'ssl_verify_result':0,'http_code':200,'size_upload':size}
    with patch.object(q.subprocess,'run',return_value=SimpleNamespace(returncode=rc,stdout=json.dumps(metrics).encode())) as run:
     r=q.request('https://speed.cloudflare.com/__up',20,payload=p)
     self.assertEqual(r['status']=='passed',ok)
     args=run.call_args.args[0];self.assertIn('socks5h://127.0.0.1:10808',args);self.assertIn('--noproxy',args);self.assertNotIn('--insecure',args);self.assertNotIn('-k',args);self.assertNotIn('--location',args)
 def test_postload_failure_revokes_quality(self):
  e={'payload_version':q.VERSION,'payload_download':'passed','payload_upload':'passed','payload_recheck':'passed','payload_download_bytes':str(q.DOWNLOAD),'payload_upload_bytes':str(q.UPLOAD)}
  self.assertTrue(q.verified(e))
  for key,value in [('payload_version','old'),('payload_download','not_run'),('payload_upload','failed'),('payload_recheck','failed'),('payload_upload_bytes','131072')]:
   self.assertFalse(q.verified({**e,key:value}))
 def test_timeout_cannot_be_success(self):
  with patch.object(q.subprocess,'run',side_effect=subprocess.TimeoutExpired('curl',20)):
   self.assertEqual(q.request('https://www.gstatic.com/generate_204',20)['status'],'failed')
if __name__=='__main__':unittest.main()
