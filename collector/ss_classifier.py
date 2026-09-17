import re,urllib.parse
from .ss_deep import parse
def classify(raw):
 try:parse(raw);return {'class':'standard-ss','confidence':1.0}
 except:pass
 try:
  p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True);u=urllib.parse.unquote(p.username or '')
  is_uuid=bool(re.fullmatch(r'[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}',u));typ=(q.get('type',[''])[0] or '').lower();sec=(q.get('security',[''])[0] or '').lower()
  signals=is_uuid and typ in ('tcp','ws','grpc','xhttp','httpupgrade') and ('encryption' in q or 'headerType' in q or 'security' in q)
  if signals:return {'class':'vless-like-mislabeled','confidence':0.99,'suggested_scheme':'vless'}
 except:pass
 return {'class':'opaque-unknown','confidence':0.0}