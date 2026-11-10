#!/usr/bin/env python3
import asyncio, importlib.util, pathlib, tempfile, time, unittest
spec=importlib.util.spec_from_file_location('intel',pathlib.Path(__file__).with_name('residential-intel.py'))
intel=importlib.util.module_from_spec(spec);spec.loader.exec_module(intel)

class Driver:
    def __init__(self,fail=False):self.calls=0;self.fail=fail
    async def lookup(self,ip):
        self.calls+=1;await asyncio.sleep(.001)
        if self.fail:raise TimeoutError()
        now=int(time.time()*1000)
        return dict(ip=ip,source='fixture',at=now,expires=now+10000,**{'class':'Unknown'})

class Tests(unittest.IsolatedAsyncioTestCase):
    async def test_exact_ip_coalescing_and_ttl(self):
        registry=intel.Registry([]);driver=Driver();registry.register('fixture',driver)
        await asyncio.gather(*(registry.lookup('fixture','8.8.8.8') for _ in range(20)))
        self.assertEqual(driver.calls,1)
        await registry.lookup('fixture','1.1.1.1');self.assertEqual(driver.calls,2)
        registry.cache['fixture','8.8.8.8']['expires']=1
        await registry.lookup('fixture','8.8.8.8');self.assertEqual(driver.calls,3)
    async def test_failure_isolated_and_unknown(self):
        registry=intel.Registry([]);bad=Driver(True);good=Driver();registry.register('bad',bad);registry.register('good',good)
        for _ in range(4):self.assertIsNone(await registry.lookup('bad','8.8.8.8'))
        self.assertEqual(bad.calls,3);self.assertEqual((await registry.lookup('good','8.8.8.8'))['class'],'Unknown')
    async def test_persistent_cache_bound_to_provider_config(self):
        with tempfile.TemporaryDirectory() as directory:
            path=str(pathlib.Path(directory)/'cache.json')
            a=intel.Registry([]);a.register('fixture',Driver());a.persisted(path,'v1');await a.collect(['8.8.8.8'])
            b=intel.Registry([]);driver=Driver();b.register('fixture',driver);b.persisted(path,'v1');await b.collect(['8.8.8.8']);self.assertEqual(driver.calls,0)
            c=intel.Registry([]);c.register('fixture',driver);c.persisted(path,'v2');await c.collect(['8.8.8.8']);self.assertEqual(driver.calls,1)

if __name__=='__main__':unittest.main()
