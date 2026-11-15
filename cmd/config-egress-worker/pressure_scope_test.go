package main

import (
	"os"
	"testing"
)

func TestScopedCPUIncludesHelpersAndCores(t *testing.T) {
	files := map[string]string{
		"/proc/self/cgroup": "0::/system.slice/config-repair-egress.service\n",
		"/sys/fs/cgroup/system.slice/config-repair-egress.service/cpu.pressure": "some avg10=12.5\nfull avg10=2\n",
		"/sys/fs/cgroup/cge-sandboxes/cpu.pressure":                             "some avg10=85\n",
	}
	read := func(path string) ([]byte, error) {
		if text, ok := files[path]; ok {
			return []byte(text), nil
		}
		return nil, os.ErrNotExist
	}
	if value, ok := scopedCPUPressure("cge", read); !ok || value != 85 {
		t.Fatal("core pressure ignored", value, ok)
	}
	delete(files, "/sys/fs/cgroup/cge-sandboxes/cpu.pressure")
	if value, ok := scopedCPUPressure("cge", read); !ok || value != 12.5 {
		t.Fatal("startup helper pressure lost", value, ok)
	}
	files["/sys/fs/cgroup/cge-sandboxes/cpu.pressure"] = "some avg10=NaN\n"
	if _, ok := scopedCPUPressure("cge", read); ok {
		t.Fatal("invalid telemetry must fall back")
	}
	delete(files, "/sys/fs/cgroup/system.slice/config-repair-egress.service/cpu.pressure")
	if _, ok := scopedCPUPressure("cge", read); ok {
		t.Fatal("missing helper telemetry must fall back")
	}
	files["/proc/self/cgroup"] = "2:cpu:/legacy\n"
	if _, ok := scopedCPUPressure("cge", read); ok {
		t.Fatal("legacy hierarchy must fall back")
	}
}
