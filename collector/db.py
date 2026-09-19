import sqlite3,pathlib
import os
DB=pathlib.Path(os.environ.get('CONFIG_MANAGER_DB',str(pathlib.Path(__file__).resolve().parent.parent/'data'/'collector.db')))
def connect():
 c=sqlite3.connect(DB,timeout=30,check_same_thread=False); c.row_factory=sqlite3.Row; c.execute('PRAGMA busy_timeout=30000'); c.execute('PRAGMA synchronous=NORMAL'); return c
def col(c,t,n,d):
 if n not in [x[1] for x in c.execute(f'PRAGMA table_info({t})')]: c.execute(f'ALTER TABLE {t} ADD COLUMN {n} {d}')
def init():
 c=connect();
 try:c.execute('PRAGMA journal_mode=WAL')
 except sqlite3.OperationalError:pass
 c.executescript('''CREATE TABLE IF NOT EXISTS sources(id INTEGER PRIMARY KEY,url TEXT UNIQUE NOT NULL,interval_sec INTEGER NOT NULL DEFAULT 10,enabled INTEGER NOT NULL DEFAULT 1,last_fetch REAL,last_ok REAL,next_fetch REAL,status TEXT DEFAULT 'new',http_status INTEGER,duration_ms INTEGER DEFAULT 0,total INTEGER DEFAULT 0,valid INTEGER DEFAULT 0,invalid INTEGER DEFAULT 0,duplicates INTEGER DEFAULT 0,unique_count INTEGER DEFAULT 0,error TEXT DEFAULT ''); CREATE TABLE IF NOT EXISTS configs(id INTEGER PRIMARY KEY,fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw TEXT NOT NULL,first_seen REAL,last_seen REAL); CREATE TABLE IF NOT EXISTS source_configs(source_id INTEGER,config_id INTEGER,last_seen REAL,PRIMARY KEY(source_id,config_id)); CREATE TABLE IF NOT EXISTS issues(id INTEGER PRIMARY KEY,source_id INTEGER,seen_at REAL,kind TEXT,code TEXT,raw TEXT,repairable INTEGER DEFAULT 0); CREATE TABLE IF NOT EXISTS quarantine(id INTEGER PRIMARY KEY,fingerprint TEXT NOT NULL,source_id INTEGER,kind TEXT NOT NULL,raw TEXT NOT NULL,reasons TEXT NOT NULL,first_seen REAL,last_seen REAL,hits INTEGER DEFAULT 1,UNIQUE(fingerprint,source_id)); CREATE TABLE IF NOT EXISTS warnings(id INTEGER PRIMARY KEY,fingerprint TEXT NOT NULL,source_id INTEGER,kind TEXT NOT NULL,code TEXT NOT NULL,message TEXT NOT NULL,raw TEXT NOT NULL,first_seen REAL,last_seen REAL,hits INTEGER DEFAULT 1,UNIQUE(fingerprint,source_id,code)); CREATE TABLE IF NOT EXISTS repairs(id INTEGER PRIMARY KEY,fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw_original TEXT NOT NULL,raw_repaired TEXT NOT NULL,changes TEXT NOT NULL,created_at REAL NOT NULL); CREATE TABLE IF NOT EXISTS proposals(id INTEGER PRIMARY KEY,kind TEXT NOT NULL,code TEXT NOT NULL,occurrences INTEGER NOT NULL,sample_raw TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'candidate',first_seen REAL,last_seen REAL,UNIQUE(kind,code)); CREATE TABLE IF NOT EXISTS intelligence_events(id INTEGER PRIMARY KEY,fingerprint TEXT,source_id INTEGER,kind TEXT,state TEXT,confidence REAL,engine_version TEXT,details TEXT,seen_at REAL); CREATE TABLE IF NOT EXISTS engine_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL,updated_at REAL NOT NULL);''')
 c.execute('''CREATE TABLE IF NOT EXISTS recovery_registry(id INTEGER PRIMARY KEY,original_fingerprint TEXT UNIQUE NOT NULL,original_kind TEXT NOT NULL,original_raw TEXT NOT NULL,candidate_fingerprint TEXT NOT NULL,candidate_kind TEXT NOT NULL,candidate_raw TEXT NOT NULL,class TEXT NOT NULL,confidence REAL NOT NULL,validator_ok INTEGER NOT NULL,xray_ok INTEGER NOT NULL,singbox_ok INTEGER NOT NULL,recoverable INTEGER NOT NULL,first_seen REAL NOT NULL,last_seen REAL NOT NULL,hits INTEGER DEFAULT 1)''')
 c.execute('''CREATE TABLE IF NOT EXISTS recovered_candidates(id INTEGER PRIMARY KEY,original_fingerprint TEXT UNIQUE NOT NULL,candidate_fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw TEXT NOT NULL,source TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'structurally_verified',first_seen REAL NOT NULL,last_seen REAL NOT NULL)''')
 c.execute('''CREATE TABLE IF NOT EXISTS test_candidates(id INTEGER PRIMARY KEY,fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw TEXT NOT NULL,origin TEXT NOT NULL,stage TEXT NOT NULL DEFAULT 'queued',upload_ok INTEGER,download_ok INTEGER,healthy INTEGER NOT NULL DEFAULT 0,created_at REAL NOT NULL,updated_at REAL NOT NULL)''')
 c.execute('''CREATE TABLE IF NOT EXISTS healthy_configs(id INTEGER PRIMARY KEY,fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw TEXT NOT NULL,remark TEXT,country_code TEXT,country_name TEXT,country_flag TEXT,city TEXT,region TEXT,asn TEXT,network_org TEXT,datacenter TEXT,egress_ip TEXT,cdn_state TEXT NOT NULL DEFAULT 'unknown',upload_ok INTEGER NOT NULL,download_ok INTEGER NOT NULL,first_healthy REAL NOT NULL,last_healthy REAL NOT NULL,next_check REAL NOT NULL)''')
 c.execute('''CREATE TABLE IF NOT EXISTS health_attempts(id INTEGER PRIMARY KEY,fingerprint TEXT NOT NULL,attempt INTEGER NOT NULL,phase TEXT NOT NULL,upload_ok INTEGER NOT NULL,download_ok INTEGER NOT NULL,external_ok INTEGER NOT NULL DEFAULT 0,server_traffic_ok INTEGER NOT NULL DEFAULT 0,providers TEXT,bytes_up INTEGER DEFAULT 0,bytes_down INTEGER DEFAULT 0,started_at REAL,finished_at REAL,result TEXT NOT NULL)''')
 c.execute('CREATE TABLE IF NOT EXISTS health_claims(fingerprint TEXT PRIMARY KEY,claimed_at REAL NOT NULL)')

 c.execute('''CREATE TABLE IF NOT EXISTS network_intelligence(fingerprint TEXT PRIMARY KEY,provider TEXT,hosting INTEGER,cdn INTEGER,cdn_provider TEXT,facility TEXT,facility_confidence REAL DEFAULT 0,evidence TEXT,updated_at REAL NOT NULL)''')

 for cname,ctype in [('rdap_network','TEXT'),('reverse_dns','TEXT')]:
  try:c.execute('ALTER TABLE network_intelligence ADD COLUMN '+cname+' '+ctype)
  except:pass
 c.execute('''CREATE TABLE IF NOT EXISTS throughput_samples(id INTEGER PRIMARY KEY,ts REAL NOT NULL,attempts INTEGER NOT NULL,healthy INTEGER NOT NULL,queued INTEGER NOT NULL)''')

 for cname,ctype in [('lane',"TEXT DEFAULT 'fast'"),('cost_ms','REAL DEFAULT 0')]:
  try:c.execute('ALTER TABLE test_candidates ADD COLUMN '+cname+' '+ctype)
  except:pass
 c.execute('CREATE INDEX IF NOT EXISTS idx_tc_stage_lane_updated ON test_candidates(stage,lane,updated_at,id)')
 c.execute('CREATE INDEX IF NOT EXISTS idx_ha_fp_finished ON health_attempts(fingerprint,finished_at)')
 c.execute('''CREATE TABLE IF NOT EXISTS profile_samples(id INTEGER PRIMARY KEY,ts REAL NOT NULL,fingerprint TEXT,kind TEXT,sandbox_ms REAL,probe_ms REAL,db_ms REAL,enrich_ms REAL,total_ms REAL,state TEXT)''')
 for n,d in [('raw_count','INTEGER DEFAULT 0'),('new_count','INTEGER DEFAULT 0'),('known_count','INTEGER DEFAULT 0'),('lifetime_seen','INTEGER DEFAULT 0'),('consecutive_errors','INTEGER DEFAULT 0'),('issue_count','INTEGER DEFAULT 0')]: col(c,'sources',n,d)
 c.commit(); c.close()
