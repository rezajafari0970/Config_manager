from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
import time
class H(BaseHTTPRequestHandler):
 def do_GET(self):
  if self.path=='/slow': time.sleep(12); body=b''
  elif self.path=='/bad': body=b'not-a-config broken payload'
  elif self.path=='/large': body=b'x'*(6*1024*1024)
  else: body=b'vless://11111111-1111-1111-1111-111111111111@example.com:443#ok'
  try:self.send_response(200);self.end_headers();self.wfile.write(body)
  except:pass
 def log_message(self,*a):pass
ThreadingHTTPServer(('127.0.0.1',19090),H).serve_forever()
