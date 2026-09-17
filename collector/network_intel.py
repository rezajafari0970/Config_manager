import json,time
from .db import connect
CDNS={'cloudflare':'Cloudflare','akamai':'Akamai','fastly':'Fastly','cloudfront':'Amazon CloudFront','cdn77':'CDN77','bunny':'Bunny CDN','gcore':'Gcore','imperva':'Imperva'}
HOSTERS=('hetzner','digitalocean','ovh','amazon','aws','google cloud','microsoft','azure','vultr','linode','leaseweb','contabo','oracle','choopa')
def classify(fp,meta):
 org=(meta.get('network_org') or '').strip();low=org.lower();cp=next((v for k,v in CDNS.items() if k in low),None);hosting=any(x in low for x in HOSTERS);facility=None;confidence=0.0
 evidence={'org':org,'asn':meta.get('asn'),'egress_ip':meta.get('egress_ip'),'rule':'organization/asn signals only'}
 r={'provider':org or None,'hosting':hosting,'cdn':bool(cp),'cdn_provider':cp,'facility':facility,'facility_confidence':confidence,'evidence':evidence}
 c=connect();c.execute('INSERT INTO network_intelligence(fingerprint,provider,hosting,cdn,cdn_provider,facility,facility_confidence,evidence,updated_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(fingerprint) DO UPDATE SET provider=excluded.provider,hosting=excluded.hosting,cdn=excluded.cdn,cdn_provider=excluded.cdn_provider,facility=excluded.facility,facility_confidence=excluded.facility_confidence,evidence=excluded.evidence,updated_at=excluded.updated_at',(fp,r['provider'],int(hosting),int(bool(cp)),cp,facility,confidence,json.dumps(evidence),time.time()));c.commit();c.close();return r