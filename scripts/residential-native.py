#!/usr/bin/env python3
"""Pinned native protocol driver. Shares isolation, never Xray wire mappings."""
import copy, ipaddress, socket

def sandbox_type(lab):
    class NativeSandbox(lab.Sandbox):
        def core_mounts(self):
            # Official linux-amd64 build uses glibc. Mount only its fixed loader
            # dependencies read-only, never a host directory or config path.
            args=['--ro-bind','/lib64/ld-linux-x86-64.so.2','/lib64/ld-linux-x86-64.so.2']
            for name in ['libdl.so.2','libpthread.so.0','libc.so.6']:
                args+=['--ro-bind','/usr/lib/x86_64-linux-gnu/'+name,'/lib/x86_64-linux-gnu/'+name]
            return args
        def core_command(self,test):
            return ['/xray','check' if test else 'run','-c','/config.json']
    return NativeSandbox

def prepare(config):
    outbounds=config.get('outbounds',[])
    if len(outbounds)!=1 or outbounds[0].get('type') not in ('tuic','hysteria','hysteria2'):
        raise ValueError('outbound_driver_pending')
    ob=outbounds[0]
    if ob.get('detour') or ob.get('server_ports') or ob.get('domain_resolver'):
        raise ValueError('outbound_driver_pending')
    host=ob['server'];port=int(ob['server_port'])
    if not 1<=port<=65535:raise ValueError('invalid_port')
    try:literal=ipaddress.ip_address(host)
    except ValueError:literal=None
    if literal is not None and literal.version==6:raise ValueError('ipv6_only_probe_pending')
    addresses=sorted({x[4][0] for x in socket.getaddrinfo(host,port,socket.AF_INET,socket.SOCK_DGRAM)})
    if not addresses or any(not ipaddress.ip_address(ip).is_global for ip in addresses):
        raise ValueError('endpoint_not_public')
    resolved={host:{'ips':addresses,'port':port,'ports':[str(port)]}}
    runtime=copy.deepcopy(config)
    original=next((i for i in runtime.get('inbounds',[]) if i.get('type')=='socks'),{})
    runtime['inbounds']=[{'type':'socks','tag':original.get('tag','socks'),'listen':'127.0.0.1','listen_port':10808}]
    runtime['log']={'level':'warn'}
    # Pin only the actual upstream endpoint; target destinations remain domains
    # carried by the original protocol. TLS authenticates the original name.
    proxy=runtime['outbounds'][0];proxy['server']=addresses[0]
    tls=proxy.setdefault('tls',{})
    if not tls.get('server_name'):tls['server_name']=host
    return resolved,runtime
