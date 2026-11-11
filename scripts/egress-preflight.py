"""Finite HTTPS probe scope; never relax the shared proxy-health firewall.

Resolver-only bypass rules cannot match our named HTTPS probe destinations.
Inspect a copy without those rules, but execute the original bytes unchanged.
Direct DNS remains blocked by the endpoint-only network namespace. This is not
a certificate of the source configuration's DNS leak behavior.
"""
import copy

RESOLVERS = {'1.1.1.1', '1.0.0.1', '8.8.8.8', '8.8.4.4', '9.9.9.9',
             '149.112.112.112', '104.16.248.249', '104.16.249.249'}

def dns_bypass(rule):
    if not isinstance(rule, dict) or set(rule) - {'type', 'port', 'network', 'inboundTag', 'ip', 'outboundTag'}:
        return False
    if rule.get('type', 'field') != 'field':
        return False
    if rule.get('port') == '53':
        return True
    ips = rule.get('ip')
    return isinstance(ips, list) and bool(ips) and all(isinstance(ip, str) and ip in RESOLVERS for ip in ips)

def inspect(config, preflight):
    obj = preflight.logical_for_inspection(config)
    direct = {x.get('tag') for x in obj.get('outbounds', []) if x.get('protocol') == 'freedom'}
    routing = obj.setdefault('routing', {})
    routing['rules'] = [r for r in routing.get('rules', [])
                        if not (r.get('outboundTag') in direct and dns_bypass(r))]
    return obj

def runtime(config, lab):
    obj = inspect(config, lab.preflight)
    resolved = lab.endpoints(obj)
    lab.runtime_config(obj, resolved)
    return resolved, copy.deepcopy(config)
