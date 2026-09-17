import os,time
STATE={'started':time.time(),'loops':0,'last_loop':0.0,'active':0,'completed':0}
def review_reason(s):
 if s['status']=='error': return 'خطا در دریافت: '+(s['error'] or 'خطای ارتباط نامشخص')
 if s['status']=='empty': return 'پاسخ دریافت شد اما هیچ کانفیگی پیدا نشد'
 if s['status']=='warning': return f"{s['issue_count']} مشکل ساختاری؛ کانفیگ سالمی در این Fetch وجود ندارد"
 return ''
