"""Versioned device payload proof, shared by measurement and admission.
This finite HTTP proof is independent of UDP/HTTP3 and application functionality.
"""
import math

BENCHMARK_VERSION = 'android-benchmark.v2'
PAYLOAD_VERSION = 'device-payload.v2'
DOWNLOAD_BYTES = 2097152
UPLOAD_BYTES = 1048576

def timestamp(value):
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        return None
    try:
        value = float(value)
    except (OverflowError, ValueError):
        return None
    return value if math.isfinite(value) else None

def verified(check, status):
    return isinstance(check, dict) and check.get('status') == 'passed' and check.get('transport_tls') == 'verified' and check.get('curl_exit') == 0 and check.get('http_status') == status

def complete(payload):
    if not isinstance(payload, dict) or payload.get('version') != PAYLOAD_VERSION:
        return False
    down, up, after = (payload.get(k, {}) for k in ('download', 'upload', 'post_load'))
    return (verified(down, 200) and down.get('download_bytes') == DOWNLOAD_BYTES
            and verified(up, 200) and up.get('upload_bytes') == UPLOAD_BYTES
            and verified(after, 204))
