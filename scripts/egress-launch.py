#!/usr/bin/env python3
"""Minimal trusted launcher; same cgroup and limits as the shared live driver."""
import os,resource,sys
with open(sys.argv[1]+'/cgroup.procs','w') as stream:stream.write(str(os.getpid()))
resource.setrlimit(resource.RLIMIT_NOFILE,(256,256))
resource.setrlimit(resource.RLIMIT_FSIZE,(1024*1024,1024*1024))
resource.setrlimit(resource.RLIMIT_CORE,(0,0))
os.execv(sys.argv[2],sys.argv[2:])
