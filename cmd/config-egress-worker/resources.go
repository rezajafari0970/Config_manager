package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type resourceSample struct {
	availableKB int64
	cpu, io     float64
}

func pressure(data []byte, kind string) float64 {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != kind {
			continue
		}
		for _, f := range fields[1:] {
			if strings.HasPrefix(f, "avg10=") {
				v, _ := strconv.ParseFloat(strings.TrimPrefix(f, "avg10="), 64)
				return v
			}
		}
	}
	return 0
}
func sampleResources() resourceSample {
	s := resourceSample{availableKB: -1}
	b, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) > 1 && f[0] == "MemAvailable:" {
				s.availableKB, _ = strconv.ParseInt(f[1], 10, 64)
			}
		}
	}
	b, _ = os.ReadFile("/proc/pressure/cpu")
	s.cpu = pressure(b, "some")
	b, _ = os.ReadFile("/proc/pressure/io")
	s.io = pressure(b, "full")
	return s
}
func resourceCapacity(maximum int, s resourceSample) int {
	if s.availableKB < 0 {
		return min(maximum, 2)
	}
	if s.availableKB < 512*1024 || s.io >= 25 {
		return 0
	}
	limit := maximum
	if s.availableKB < 1024*1024 {
		limit = min(limit, 2)
	} else if s.availableKB < 1536*1024 {
		limit = min(limit, 4)
	}
	if s.cpu >= 90 || s.io >= 15 {
		limit = min(limit, 2)
	} else if s.cpu >= 80 || s.io >= 8 {
		limit = min(limit, max(2, maximum/2))
	}
	return limit
}

type resourceGovernor struct {
	current int
	changed time.Time
}

func (g *resourceGovernor) update(target int, now time.Time) int {
	if target < g.current {
		g.current = target
		g.changed = now
	} else if target > g.current && now.Sub(g.changed) >= 10*time.Second {
		g.current++
		g.changed = now
	}
	return g.current
}
