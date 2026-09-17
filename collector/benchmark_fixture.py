import sqlite3,pathlib,time,json,os
ROOT=pathlib.Path('/root/Config_manager/data');SRC=ROOT/'collector.db';DST=ROOT/'benchmark-fixture.db';META=ROOT/'benchmark-fixture.json'
def create():
 tmp=ROOT/'benchmark-fixture.tmp.db';tmp.unlink(missing_ok=True);src=sqlite3.connect(SRC);dst=sqlite3.connect(tmp);src.backup(dst,pages=256);dst.close();src.close();c=sqlite3.connect(tmp);check=c.execute('pragma integrity_check').fetchone()[0];counts={'candidates':c.execute('select count(*) from test_candidates').fetchone()[0],'configs':c.execute('select count(*) from configs').fetchone()[0]};c.close()
 if check!='ok':tmp.unlink(missing_ok=True);raise RuntimeError('fixture-integrity:'+check)
 os.replace(tmp,DST);out={'created':time.time(),'integrity':check,**counts};META.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(create()))