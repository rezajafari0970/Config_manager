import asyncio
from .db import init
from .engine import scheduler
async def main():
 init();await scheduler()
if __name__=='__main__':asyncio.run(main())