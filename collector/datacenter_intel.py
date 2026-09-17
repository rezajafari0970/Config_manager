import json,socket,subprocess,urllib.parse
def jget(url):
 p=subprocess.run(['/usr/bin/curl','-fsS','--max-time','7',url],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
 try:return json.loads(p.stdout) if p.returncode==0 else {}
 except:return {}
def rdap(ip):return jget('https://rdap.org/ip/'+urllib.parse.quote(ip,safe=''))
def reverse(ip):
 try:return socket.gethostbyaddr(ip)[0]
 except:return None
def assess(meta):
 ip=meta.get('egress_ip');r=rdap(ip) if ip else {};ptr=reverse(ip) if ip else None;name=r.get('name') or '';handle=r.get('handle') or '';org=meta.get('network_org') or '';ev={'rdap_name':name,'rdap_handle':handle,'ptr':ptr,'org':org,'asn':meta.get('asn')}
 facility=None;confidence=0.0
 # PTR/RDAP are evidence, but provider/city alone never proves a physical facility.
 return {'facility':facility,'facility_confidence':confidence,'rdap_network':name or handle or None,'reverse_dns':ptr,'evidence':ev}