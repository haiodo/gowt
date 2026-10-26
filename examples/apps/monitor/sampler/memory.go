package sampler

import (
	"strconv"
	"strings"
)

// Used, Cached and Free add up to Total: Used is what the system could not hand out at once, Cached
// what it holds as droppable cache, Free what nobody holds. macOS adds its own pages in source_darwin.go.

// parseMeminfo returns bytes by key.
func parseMeminfo(s string) map[string]uint64 {
	m := map[string]uint64{}
	for _, l := range strings.Split(s, "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		f := strings.Fields(v)
		if len(f) == 0 {
			continue
		}
		n, _ := strconv.ParseUint(f[0], 10, 64)
		if len(f) > 1 && f[1] == "kB" {
			n *= 1024
		}
		m[k] = n
	}
	return m
}

// linuxMemory reads /proc/meminfo values in bytes. Used is Total minus MemAvailable; of the available
// part MemFree is free and the rest is cache.
func linuxMemory(m map[string]uint64) Memory {
	var r Memory
	r.Total = m["MemTotal"]
	avail, ok := m["MemAvailable"]
	if !ok { // kernels before 3.14
		avail = m["MemFree"] + m["Cached"] + m["Buffers"] + m["SReclaimable"]
	}
	avail = min(avail, r.Total)
	r.Free = min(m["MemFree"], avail)
	r.Cached = avail - r.Free
	r.Used = r.Total - avail
	r.App = r.Used
	r.SwapTotal = m["SwapTotal"]
	r.SwapUsed = r.SwapTotal - min(m["SwapFree"], r.SwapTotal)
	return r
}

// windowsMemory takes the physical total and available bytes of GlobalMemoryStatusEx and the system
// cache of GetPerformanceInfo. Available holds the standby cache, so Cached is capped at it and Free
// is the rest: a machine that has been up a while shows little Free, as Task Manager does.
func windowsMemory(total, avail, cache uint64) Memory {
	var r Memory
	r.Total = total
	avail = min(avail, total)
	r.Cached = min(cache, avail)
	r.Free = avail - r.Cached
	r.Used = total - avail
	r.App = r.Used
	return r
}
