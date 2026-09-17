import threading
STARTUP=threading.BoundedSemaphore(1)
PROBE=threading.BoundedSemaphore(6)
def startup_enter():STARTUP.acquire()
def startup_leave():STARTUP.release()
def probe_enter():PROBE.acquire()
def probe_leave():PROBE.release()