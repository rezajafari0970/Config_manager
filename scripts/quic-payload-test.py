#!/usr/bin/env python3
import unittest
from unittest.mock import patch
from benchmarks.quic_transfer import accepted,transfer,DOWNLOAD,UPLOAD
class QUICPayload(unittest.TestCase):
 def test_full_h3_only(self):
  good={'http_version':'3','http_code':200,'ssl_verify_result':0,'size_download':DOWNLOAD,'size_upload':UPLOAD}
  self.assertTrue(accepted(0,good,'size_download',DOWNLOAD));self.assertTrue(accepted(0,good,'size_upload',UPLOAD))
  for patchrow in [{'http_version':'2'},{'http_code':403},{'ssl_verify_result':20},{'size_download':DOWNLOAD-1}]:
   self.assertFalse(accepted(0,{**good,**patchrow},'size_download',DOWNLOAD))
  self.assertFalse(accepted(28,good,'size_upload',UPLOAD))
 def test_invalid_destinations_never_open_sockets(self):
  for address in ['127.0.0.1','10.0.0.1','::1','invalid','169.254.169.254']:
   with patch('benchmarks.quic_transfer.socket.socket',side_effect=AssertionError('must not dial')):
    with self.assertRaises(ValueError):transfer(10808,address)
 def test_resource_bounds(self):
  for port,timeout in [(0,20),(65536,20),(10808,0),(10808,46)]:
   with self.assertRaises(ValueError):transfer(port,'1.1.1.1',timeout=timeout)
if __name__=='__main__':unittest.main()
