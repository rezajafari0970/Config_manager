from .db import connect
from .system_pressure import snapshot as pressure
def state():
 c=connect();q=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='queued'").fetchone()[0];retry=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='retry_wait'").fetchone()[0];c.close();p=pressure();cpu=p['psi']['cpu'].get('avg10',0);return {'queue':q,'retry':retry,'cpu_psi':cpu,'health_priority':q>500 or retry>50 or cpu>35}
def allow_background():return not state()['health_priority']