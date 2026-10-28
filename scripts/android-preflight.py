#!/usr/bin/env python3
"""Readiness only. No device control, installation or Android pass is implied."""
import json
import pathlib
import shutil
import subprocess

adb = shutil.which("adb")
devices = []
if adb:
    p = subprocess.run([adb, "devices"], capture_output=True, text=True, timeout=15)
    devices = [line.split()[1] for line in p.stdout.splitlines()[1:] if len(line.split()) == 2]
ready = "device" in devices
report = {
    "adb_available": bool(adb),
    "emulator_available": bool(shutil.which("emulator")),
    "hardware_acceleration_available": pathlib.Path("/dev/kvm").exists(),
    "authorized_device_count": devices.count("device"),
    "android_acceptance": "not_run",
    "ready_for_device_setup": ready,
    "blocker": None if ready else "no_authorized_android_test_device",
}
print(json.dumps(report, indent=2))
raise SystemExit(0 if ready else 2)
