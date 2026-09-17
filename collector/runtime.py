import asyncio,os
MAX_CONCURRENCY=max(2,min(8,int(os.getenv('FETCH_CONCURRENCY','4'))))
semaphore=asyncio.Semaphore(MAX_CONCURRENCY)
def review_reason(s):
 if s['status']=='error': return 'خطا در دریافت: '+(s['error'] or 'خطای ارتباط نامشخص')
 if s['status']=='empty': return 'پاسخ دریافت شد اما هیچ کانفیگی پیدا نشد'
 if s['status']=='warning': return f"{s['issue_count']} مشکل ساختاری؛ کانفیگ سالمی در این Fetch وجود ندارد"
 return ''
