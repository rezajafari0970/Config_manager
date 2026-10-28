#!/usr/bin/env python3
"""Controlled synthetic core-schema tests; never a connectivity or Android test."""
import argparse
import copy
import hashlib
import json
import pathlib
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument("--core", type=pathlib.Path, required=True)
args = parser.parse_args()
core = args.core.resolve()
lock = json.loads((ROOT / "compatibility/target.lock.json").read_text())["core"]
if hashlib.sha256(core.read_bytes()).hexdigest() != lock["linux_amd64_binary_sha256"]:
    raise SystemExit("unexpected core binary; refusing unpinned oracle")
version = subprocess.check_output([str(core), "version"], text=True, timeout=10).splitlines()[0]
uid = "00000000-0000-4000-8000-000000000001"
endpoint = "192.0.2.1"

def config(protocol, settings):
    return {"log": {"loglevel": "warning"}, "stats": {}, "inbounds": [],
            "outbounds": [{"tag": "proxy", "protocol": protocol, "settings": settings}],
            "routing": {"domainStrategy": "AsIs", "rules": []}}

cases = []
for protocol in ["vless", "vmess"]:
    user = {"id": uid, **({"encryption": "none"} if protocol == "vless" else {"security": "auto"})}
    flat = config(protocol, {"address": endpoint, "port": 443, **user})
    legacy = config(protocol, {"vnext": [{"address": endpoint, "port": 443, "users": [user]}]})
    cases += [(protocol + "_flat", flat, True), (protocol + "_legacy_vnext", legacy, True)]
cases += [("trojan_flat", config("trojan", {"address": endpoint, "port": 443, "password": "synthetic-test-password"}), True)]
cases += [("trojan_legacy_servers", config("trojan", {"servers": [{"address": endpoint, "port": 443, "password": "synthetic-test-password"}]}), True)]
cases += [("ss_flat", config("shadowsocks", {"address": endpoint, "port": 8388, "method": "aes-128-gcm", "password": "synthetic-test-password"}), True)]
hy = config("hysteria", {"version": 2, "address": endpoint, "port": 443})
hy["outbounds"][0]["streamSettings"] = {"network": "hysteria", "security": "tls", "tlsSettings": {"serverName": "example.com", "alpn": ["h3"]}, "hysteriaSettings": {"version": 2, "auth": "synthetic-test-password"}}
cases += [("hysteria2", hy, True)]
hy1 = copy.deepcopy(hy)
hy1["outbounds"][0]["settings"]["version"] = 1
cases += [("hysteria1_rejected", hy1, False)]
key = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
wg = config("wireguard", {"secretKey": key, "address": ["10.0.0.2/32"], "peers": [{"publicKey": key, "endpoint": "192.0.2.1:51820"}], "reserved": [0, 0, 0], "mtu": 1280})
cases += [("wireguard", wg, True)]
valid = config("vless", {"address": endpoint, "port": 443, "id": uid, "encryption": "none"})
bad = copy.deepcopy(valid)
bad["stats"] = []
cases += [("stats_array_rejected", bad, False)]
missing = copy.deepcopy(valid)
del missing["routing"]
cases += [("core_accepts_missing_routing_android_envelope_differs", missing, True)]

results = []
with tempfile.TemporaryDirectory(prefix="config-repair-core-") as temp:
    temp = pathlib.Path(temp)
    for name, document, expected in cases:
        path = temp / (name + ".json")
        path.write_text(json.dumps(document))
        p = subprocess.run([str(core), "run", "-test", "-c", str(path)], capture_output=True, text=True, timeout=15)
        results.append({"name": name, "expected_core_acceptance": expected, "exit_code": p.returncode, "passed": (p.returncode == 0) == expected})
    original = temp / "stats_array.json"
    repaired = temp / "stats_object.json"
    original.write_text(json.dumps(bad))
    cli = subprocess.run([str(ROOT / "bin/config-repair"), "repair-stats", str(original), str(repaired)], capture_output=True, text=True, timeout=10)
    p = subprocess.run([str(core), "run", "-test", "-c", str(repaired)], capture_output=True, text=True, timeout=15) if cli.returncode == 0 else None
    results.append({"name": "go_stats_repair_accepted_by_pinned_core", "passed": bool(p and p.returncode == 0), "cli_exit_code": cli.returncode})

report = {"core": version, "binary_sha256": lock["linux_amd64_binary_sha256"], "tests": results,
          "passed": all(x["passed"] for x in results), "android_acceptance": "not_run", "network_connection": "not_run"}
print(json.dumps(report, indent=2))
raise SystemExit(0 if report["passed"] else 1)
