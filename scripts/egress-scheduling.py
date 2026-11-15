"""Transport scheduling hints only; Go remains the publication authority.

Two paired waves may overlap under the session's shared 12-connection cap.
Every measurement retains its original timeout, nonce, TLS and wave binding.
Recovery still measures a complete paired third set when required. Serialized
transports and discovery retain their conservative ordering.
"""
SERIAL_TRANSPORTS = frozenset(('xhttp', 'splithttp'))

def session_deadline(targets, waves, timeout, sequential):
 # The semaphore also queues observations. Two overlapping waves can occupy
 # two full timeout batches; an adaptive third wave needs its own budget.
 # The per-observation timeout is unchanged and the supervisor remains bounded.
 batches = ((targets + 11) // 12) * waves if sequential else (targets * min(waves, 2) + 11) // 12 + ((targets + 11) // 12 if waves == 3 else 0)
 return max(25, batches * timeout + 10)

def sequential(request, config):
 if request.get('discovery'):
  return True
 return any(out.get('streamSettings', {}).get('network') in SERIAL_TRANSPORTS
            for out in config.get('outbounds', []))
