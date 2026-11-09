#!/usr/bin/env python3
"""Isolated Ads smart profile validator; never shares normal worker resources."""
import importlib.util,pathlib
path=pathlib.Path(__file__).resolve().parent/'probe-one.py'
spec=importlib.util.spec_from_file_location('ads_driver',path)
driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
driver.PREFIX='csa'
if __name__=='__main__':driver.main(core_only=True)
