#!/usr/bin/env python3
"""Capability discovery only. Termux runtime and Android APK acceptance differ."""
import json,os,pathlib,shutil,subprocess

def command(args,timeout=3):
    try:
        p=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
        return p.stdout.strip() if p.returncode==0 else None
    except (OSError,subprocess.SubprocessError):return None

def inspect():
    termux='com.termux' in os.environ.get('PREFIX','') and shutil.which('getprop') is not None
    devices=[];adb=shutil.which('adb')
    if adb:
        data=command([adb,'devices'],15) or ''
        devices=[line.split()[1] for line in data.splitlines()[1:] if len(line.split())==2]
    report={'version':'android-preflight.v2','adb_available':bool(adb),'emulator_available':bool(shutil.which('emulator')),'hardware_acceleration_available':pathlib.Path('/dev/kvm').exists(),'authorized_device_count':devices.count('device'),'android_acceptance':'not_run','apk_functional_tests':'not_run','termux_available':termux,'ready_for_device_setup':'device' in devices,'ready_for_native_core_tests':termux and bool(shutil.which('curl')),'installed_v2rayng_version':None,'blocker':None if 'device' in devices else 'no_authorized_android_apk_control'}
    if termux:
        report['device']={k:command(['getprop',v]) for k,v in {'android_release':'ro.build.version.release','sdk':'ro.build.version.sdk','model':'ro.product.model','abi':'ro.product.cpu.abi'}.items()}
        try:report['battery']=json.loads(command(['termux-battery-status']) or '{}')
        except ValueError:report['battery']={'status':'unavailable'}
        curl=command(['curl','--version']) or ''
        report['curl_http3']='HTTP3' in curl
        report['network_switch_control']='unavailable_without_authorized_android_control'
        report['network_label']='requires_current_route_measurement; registered SIM names are insufficient'
    return report
if __name__=='__main__':
    report=inspect();print(json.dumps(report,indent=2));raise SystemExit(0 if report['ready_for_device_setup'] or report['ready_for_native_core_tests'] else 2)
