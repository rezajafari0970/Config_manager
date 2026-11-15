#!/usr/bin/env python3
import importlib.util,pathlib,sqlite3,unittest
spec=importlib.util.spec_from_file_location('benchmark',pathlib.Path(__file__).with_name('egress-live-throughput.py'));b=importlib.util.module_from_spec(spec);spec.loader.exec_module(b)
class CohortTest(unittest.TestCase):
 def test_distinguishes_assessment_settlement_and_insufficient_cohort(self):
  db=sqlite3.connect(':memory:');db.executescript('CREATE TABLE egress_runs(id,started);INSERT INTO egress_runs VALUES(1,1000000);CREATE TABLE egress_run_members(run,hash,status,completed_at,state,settled_at,settled_state);CREATE TABLE memberships(raw_hash);CREATE TABLE egress_jobs(hash);')
  db.executemany('INSERT INTO egress_run_members VALUES(1,?,?,?,?,?,?)',[(str(i),'completed',1000000+i*500,'UNKNOWN',1000000+i*650,'NO_SPLIT') for i in range(1000)])
  db.commit();r=b.snapshot(db)
  self.assertEqual(r['completed'],1000);self.assertEqual(r['first_states'],{'UNKNOWN':1000});self.assertTrue(r['assessed_1000_target_met']);self.assertFalse(r['definitive_1000_target_met']);self.assertFalse(r['2000_cohort_available']);self.assertIsNone(r['assessed_2000_seconds']);self.assertLess(r['definitive_at_600s'],1000)
if __name__=='__main__':unittest.main()
