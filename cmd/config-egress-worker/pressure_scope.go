package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Host CPU PSI includes unrelated collectors and projects. It must not make an
// idle Residential lane back off while its own tasks have CPU time available.
// Observe both trusted helpers and isolated cores; keep host memory/I/O gates
// and the sandbox's CPU quota unchanged. Missing telemetry falls back to host PSI.
func scopedCPUPressure(prefix string, read func(string) ([]byte, error)) (float64, bool) {
	if prefix != "cge" && prefix != "cgt" {
		return 0, false
	}
	data, err := read("/proc/self/cgroup")
	if err != nil {
		return 0, false
	}
	member := ""
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "0::/") {
			member = filepath.Clean(strings.TrimPrefix(line, "0::"))
			break
		}
	}
	if member == "" {
		return 0, false
	}
	data, err = read(filepath.Join("/sys/fs/cgroup", member, "cpu.pressure"))
	if err != nil {
		return 0, false
	}
	value, ok := cpuSome(data)
	if !ok {
		return 0, false
	}
	data, err = read("/sys/fs/cgroup/" + prefix + "-sandboxes/cpu.pressure")
	if os.IsNotExist(err) {
		return value, true // no sandbox exists before the first core starts
	}
	if err != nil {
		return 0, false
	}
	cores, ok := cpuSome(data)
	return max(value, cores), ok
}

func cpuSome(data []byte) (float64, bool) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "some" {
			continue
		}
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "avg10=") {
				value, err := strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
				return value, err == nil && value >= 0 && value <= 100
			}
		}
	}
	return 0, false
}
