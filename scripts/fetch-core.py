#!/usr/bin/env python3
"""Fetch only the pinned development oracle; no service/system modification."""
import hashlib
import json
import os
import pathlib
import platform
import tempfile
import urllib.request
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
if platform.system() != "Linux" or platform.machine() not in ("x86_64", "amd64"):
    raise SystemExit("This development lock currently covers Linux amd64 only")
lock = json.loads((ROOT / "compatibility/target.lock.json").read_text())["core"]
dest = ROOT / ".local/xray-26.7.28"
dest.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(dir=dest) as tmp:
    archive = pathlib.Path(tmp) / "core.zip"
    with urllib.request.urlopen(lock["linux_amd64_url"], timeout=45) as response, archive.open("wb") as out:
        size = 0
        while data := response.read(1024 * 1024):
            size += len(data)
            if size > 32 * 1024 * 1024:
                raise SystemExit("download size limit exceeded")
            out.write(data)
    if hashlib.sha256(archive.read_bytes()).hexdigest() != lock["linux_amd64_zip_sha256"]:
        raise SystemExit("archive checksum mismatch")
    with zipfile.ZipFile(archive) as source:
        if source.getinfo("xray").file_size > 100 * 1024 * 1024:
            raise SystemExit("binary size limit exceeded")
        binary = source.read("xray")
    if hashlib.sha256(binary).hexdigest() != lock["linux_amd64_binary_sha256"]:
        raise SystemExit("binary checksum mismatch")
    candidate = pathlib.Path(tmp) / "xray"
    candidate.write_bytes(binary)
    candidate.chmod(0o755)
    os.replace(candidate, dest / "xray")
print(json.dumps({"status": "verified", "version": lock["tag"], "binary_sha256": lock["linux_amd64_binary_sha256"]}))
