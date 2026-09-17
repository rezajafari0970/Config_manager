import sqlite3, pathlib
DB=pathlib.Path(__file__).resolve().parent.parent/'data'/'collector.db'
def connect():
 c=sqlite3.connect(DB,timeout=10,check_same_thread=False); c.row_factory=sqlite3.Row; c.execute('PRAGMA journal_mode=WAL'); return c
def init():
 c=connect(); c.executescript('''CREATE TABLE IF NOT EXISTS sources(id INTEGER PRIMARY KEY,url TEXT UNIQUE NOT NULL,interval_sec INTEGER NOT NULL DEFAULT 10,enabled INTEGER NOT NULL DEFAULT 1,last_fetch REAL,last_ok REAL,next_fetch REAL,status TEXT DEFAULT 'new',http_status INTEGER,duration_ms INTEGER DEFAULT 0,total INTEGER DEFAULT 0,valid INTEGER DEFAULT 0,invalid INTEGER DEFAULT 0,duplicates INTEGER DEFAULT 0,unique_count INTEGER DEFAULT 0,error TEXT DEFAULT ''); CREATE TABLE IF NOT EXISTS configs(id INTEGER PRIMARY KEY,fingerprint TEXT UNIQUE NOT NULL,kind TEXT NOT NULL,raw TEXT NOT NULL,first_seen REAL,last_seen REAL); CREATE TABLE IF NOT EXISTS source_configs(source_id INTEGER,config_id INTEGER,last_seen REAL,PRIMARY KEY(source_id,config_id));'''); c.commit(); c.close()
