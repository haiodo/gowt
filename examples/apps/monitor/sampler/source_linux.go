package sampler

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type linuxSource struct {
	h        Host
	pageSize uint64
}

func newSource() (source, error) {
	s := &linuxSource{pageSize: uint64(os.Getpagesize())}
	s.h.Name, _ = os.Hostname()
	s.h.OS = "Linux"
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "PRETTY_NAME="); ok {
				s.h.OS = strings.Trim(v, `"`)
			}
		}
	}
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if k, v, ok := strings.Cut(l, ":"); ok && (strings.TrimSpace(k) == "model name") {
				s.h.CPU = strings.TrimSpace(v)
				break
			}
		}
	}
	m, err := meminfo()
	if err != nil {
		return nil, err
	}
	s.h.Memory = m["MemTotal"]
	return s, nil
}

func (s *linuxSource) host() Host { return s.h }
func (s *linuxSource) close()     {}

func (s *linuxSource) read(procs bool) (raw, error) {
	var r raw
	var err error
	if r.cores, err = cpuStat(); err != nil {
		return r, err
	}
	m, err := meminfo()
	if err != nil {
		return r, err
	}
	r.mem = memoryOf(m)
	r.rx, r.tx = netDev()
	if f := fields(readFile("/proc/loadavg")); len(f) >= 3 {
		for i := range r.load {
			r.load[i], _ = strconv.ParseFloat(f[i], 64)
		}
		r.hasLoad = true
	}
	if f := fields(readFile("/proc/uptime")); len(f) >= 1 {
		up, _ := strconv.ParseFloat(f[0], 64)
		r.uptime = time.Duration(up * float64(time.Second))
	}
	if procs {
		r.procs = s.processes()
	}
	return r, nil
}

func readFile(name string) string {
	b, _ := os.ReadFile(name)
	return string(b)
}

func fields(s string) []string { return strings.Fields(s) }

// cpuStat reads the per core lines of /proc/stat, in clock ticks.
func cpuStat() ([]cpuTimes, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}
	return parseStat(string(b)), nil
}

func parseStat(s string) []cpuTimes {
	var out []cpuTimes
	for _, l := range strings.Split(s, "\n") {
		f := strings.Fields(l)
		if len(f) < 8 || !strings.HasPrefix(f[0], "cpu") || f[0] == "cpu" {
			continue
		}
		n := func(i int) uint64 { v, _ := strconv.ParseUint(f[i], 10, 64); return v }
		// user nice system idle iowait irq softirq steal
		c := cpuTimes{User: n(1) + n(2), System: n(3) + n(6) + n(7), Idle: n(4) + n(5)}
		if len(f) > 8 {
			c.System += n(8)
		}
		out = append(out, c)
	}
	return out
}

func meminfo() (map[string]uint64, error) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	return parseMeminfo(string(b)), nil
}

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

func memoryOf(m map[string]uint64) Memory {
	var r Memory
	r.Total = m["MemTotal"]
	avail, ok := m["MemAvailable"]
	if !ok {
		avail = m["MemFree"] + m["Cached"] + m["Buffers"]
	}
	r.Free = m["MemFree"]
	r.Cached = m["Cached"] + m["Buffers"] + m["SReclaimable"] - min(m["Shmem"], m["Cached"])
	if r.Total >= avail {
		r.Used = r.Total - avail
	}
	r.App = r.Used
	r.SwapTotal = m["SwapTotal"]
	if r.SwapTotal >= m["SwapFree"] {
		r.SwapUsed = r.SwapTotal - m["SwapFree"]
	}
	return r
}

// netDev sums the interfaces except loopback; a down interface does not count.
func netDev() (rx, tx uint64) {
	for _, l := range strings.Split(readFile("/proc/net/dev"), "\n") {
		name, v, ok := strings.Cut(l, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "lo" {
			continue
		}
		f := strings.Fields(v)
		if len(f) < 9 {
			continue
		}
		a, _ := strconv.ParseUint(f[0], 10, 64)
		b, _ := strconv.ParseUint(f[8], 10, 64)
		rx, tx = rx+a, tx+b
	}
	return
}

const nsPerTick = 1e9 / 100 // USER_HZ is 100 on every Linux ABI

func (s *linuxSource) processes() map[int]rawProc {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	out := make(map[int]rawProc, len(ents))
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		if p, ok := parsePidStat(string(b), s.pageSize); ok {
			out[pid] = p
		}
	}
	return out
}

// parsePidStat reads one /proc/<pid>/stat. The name is in parentheses and may hold spaces and
// parentheses, so the fields are counted from the last ")".
func parsePidStat(s string, page uint64) (rawProc, bool) {
	i, j := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if i < 0 || j < i {
		return rawProc{}, false
	}
	f := strings.Fields(s[j+1:]) // f[0] is field 3 (state)
	if len(f) < 22 {
		return rawProc{}, false
	}
	n := func(i int) uint64 { v, _ := strconv.ParseUint(f[i], 10, 64); return v }
	// utime is field 14, stime 15, rss (pages) 24
	return rawProc{Name: s[i+1 : j], CPU: (n(11) + n(12)) * nsPerTick, RSS: n(21) * page}, true
}
