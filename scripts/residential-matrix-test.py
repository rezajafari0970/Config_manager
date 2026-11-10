#!/usr/bin/env python3
import copy, importlib.util, pathlib, unittest, subprocess, sys, time, threading, os, tempfile, io
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('matrix',pathlib.Path(__file__).with_name('residential-matrix.py'))
matrix=importlib.util.module_from_spec(spec);spec.loader.exec_module(matrix)

def report(index,raw='UNKNOWN',template='UNKNOWN',infra=True):
    pin=matrix.PINS[index]
    return {'core_id':pin['id'],'core_sha256':pin['sha256'],'started':100,'finished':200,
            'infrastructure_ok':infra,'variants':[{'name':'raw','core':'passed','health':raw},
            {'name':'template','core':'passed','health':template}], 'controls':[], 'observations':[]}

class Tests(unittest.TestCase):
    def tearDown(self):
        matrix.STOP.clear()
        self.assertFalse(matrix.PROCESSES)
    def test_stream_limit_terminates_nonclosing_writer(self):
        begin=time.monotonic()
        with self.assertRaises(ValueError):matrix.run_bounded([sys.executable,'-c','import os,time;os.write(1,b"x"*5000000);time.sleep(30)'],b'{}',10)
        self.assertLess(time.monotonic()-begin,4)
    def test_bounded_request_accepts_regular_redirected_stdin(self):
        with tempfile.TemporaryFile() as source:
            source.write(b'{"plan":{}}');source.seek(0)
            with io.TextIOWrapper(source) as wrapped:
                with patch.object(matrix.sys,'stdin',wrapped):self.assertEqual(matrix.read_request(),{'plan':{}})
    def test_wrong_owner_rejected_before_blocking_stdin(self):
        p=subprocess.Popen([sys.executable,str(pathlib.Path(__file__).with_name('residential-matrix.py')),'--slot','1','--core','/not-used','--run-id','abcdef'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=os.environ|{'SHEN_RESIDENTIAL_PARENT_PID':'0'})
        try:self.assertNotEqual(p.wait(timeout=3),0)
        finally:
            matrix.terminate(p)
            for pipe in [p.stdin,p.stdout,p.stderr]:pipe.close()
    def test_cancel_during_spawn_reaps_new_child(self):
        original=subprocess.Popen;spawned=[]
        def spawn(*a,**kw):
            p=original(*a,**kw);spawned.append(p);matrix.stop();return p
        with patch.object(matrix.subprocess,'Popen',spawn):
            with self.assertRaises(InterruptedError):matrix.run_bounded([sys.executable,'-c','import time;time.sleep(30)'],b'{}',10)
        self.assertIsNotNone(spawned[0].poll())
    def test_cancel_during_read_is_bounded(self):
        timer=threading.Timer(.1,matrix.stop);timer.start();begin=time.monotonic()
        try:
            with self.assertRaises(InterruptedError):matrix.run_bounded([sys.executable,'-c','import time;time.sleep(30)'],b'{}',10)
        finally:timer.join()
        self.assertLess(time.monotonic()-begin,4)
    def test_cleanup_attempts_every_slot_after_timeout_and_error(self):
        calls=[]
        def cleanup(cmd,**kw):
            calls.append(cmd)
            if len(calls)==1:raise subprocess.TimeoutExpired(cmd,5)
            return subprocess.CompletedProcess(cmd,1 if len(calls)==2 else 0)
        with patch.object(matrix.subprocess,'run',cleanup):
            with self.assertRaises(RuntimeError):matrix.cleanup_slots('helper',2,'abcdef')
        self.assertEqual([c[c.index('--cleanup-slot')+1] for c in calls],['4','5','6'])
    def test_healthy_older_core_wins_without_mixing_reports(self):
        latest=report(0);latest['variants'][0]['core']='failed'
        old=report(2,'HEALTHY','HEALTHY');old['observations']=[{'ip':'192.0.2.7'}]
        result=matrix.select([latest,old]);self.assertEqual(result['core_id'],old['core_id'])
        self.assertEqual(result['observations'],old['observations']);self.assertEqual(result['finished'],200)
        self.assertEqual(len(result['compatibility']),2)
    def test_different_cores_cannot_supply_one_variant_each(self):
        a=report(0,'HEALTHY');b=report(1,template='HEALTHY')
        result=matrix.select([a,b]);self.assertFalse(matrix.healthy(result))
        self.assertEqual(result['variants'],a['variants'])
    def test_missing_driver_is_not_confirmation_of_failure(self):
        result=matrix.select([report(0,infra=False),report(1,'UNHEALTHY','UNHEALTHY')])
        self.assertFalse(result['infrastructure_ok'])
    def test_healthy_attempt_survives_unavailable_other_core(self):
        self.assertTrue(matrix.healthy(matrix.select([report(0,infra=False),report(2,'HEALTHY','HEALTHY')])))
    def test_ties_are_stable_and_input_remains_immutable(self):
        a=report(0);before=copy.deepcopy(a);b=report(1)
        self.assertEqual(matrix.select([a,b])['core_id'],a['core_id']);self.assertEqual(a,before)

if __name__=='__main__':unittest.main()
