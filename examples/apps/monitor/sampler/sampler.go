// Package sampler reads the load of the host it runs on: CPU per core, memory, network rates and
// the process list. It is pure Go without cgo: libSystem through purego on macOS, /proc on Linux,
// kernel32, ntdll and iphlpapi on Windows. It knows nothing about windows or charts.
package sampler

import (
	"errors"
	"runtime"
	"sort"
	"time"
)

// Host is what does not change while the program runs.
type Host struct {
	Name   string
	OS     string // "macOS 26.0", "Windows 11 (build 26100)", the distribution on Linux
	CPU    string // brand string, empty where the OS has none
	Cores  int    // logical cores
	Memory uint64 // bytes of RAM
}

// Memory is in bytes. Used is Total minus what the system can hand out at once. App, Wired and
// Compressed split Used where the OS tells (macOS); Cached is file cache that can be dropped.
type Memory struct {
	Total, Used, App, Wired, Compressed, Cached, Free uint64
	SwapTotal, SwapUsed                               uint64
}

// Process is one row of the process list. CPU is percent of one core, so it passes 100 for a
// process that runs on several.
type Process struct {
	PID  int
	Name string
	CPU  float64
	RSS  uint64
}

// Snapshot is one reading. Rates (CPU, RxRate, TxRate) are over the time since the previous Sample
// and are zero in the first one.
type Snapshot struct {
	Time    time.Time
	CPU     float64   // 0..1, all cores
	User    float64   // share of CPU in user code, 0..1 of all cores
	System  float64   // share in the kernel
	Cores   []float64 // 0..1 each
	Load    [3]float64
	HasLoad bool
	Uptime  time.Duration
	Mem     Memory
	// RxRate and TxRate are bytes per second over all non-loopback interfaces that are up.
	RxRate, TxRate float64
	Procs          []Process // nil when this sample did not read them
}

type cpuTimes struct{ User, System, Idle uint64 }

type rawProc struct {
	Name string
	CPU  uint64 // cumulative CPU time, nanoseconds
	RSS  uint64
}

// raw is one reading of the cumulative counters, as the OS gives them.
type raw struct {
	cores   []cpuTimes
	rx, tx  uint64
	procs   map[int]rawProc
	mem     Memory
	load    [3]float64
	hasLoad bool
	uptime  time.Duration
}

type source interface {
	host() Host
	read(procs bool) (raw, error)
	close()
}

// ErrUnsupported is returned by New on an OS without a source.
var ErrUnsupported = errors.New("sampler: this OS is not supported")

// Sampler turns the counters of a source into rates.
type Sampler struct {
	src   source
	prev  raw
	prevT time.Time
	pprev map[int]rawProc // the last reading that had processes
	pT    time.Time
	have  bool
}

// New opens the source of the running OS.
func New() (*Sampler, error) {
	src, err := newSource()
	if err != nil {
		return nil, err
	}
	return &Sampler{src: src}, nil
}

// Host describes the machine.
func (s *Sampler) Host() Host {
	h := s.src.host()
	h.Cores = runtime.NumCPU()
	return h
}

// Close releases the source.
func (s *Sampler) Close() { s.src.close() }

// Sample reads the host; with procs it reads the process list too. Call it from one goroutine.
func (s *Sampler) Sample(procs bool) (Snapshot, error) {
	r, err := s.src.read(procs)
	if err != nil {
		return Snapshot{}, err
	}
	return s.rates(r, time.Now(), procs), nil
}

func (s *Sampler) rates(r raw, now time.Time, procs bool) Snapshot {
	snap := Snapshot{Time: now, Mem: r.mem, Load: r.load, HasLoad: r.hasLoad, Uptime: r.uptime}
	snap.Cores = make([]float64, len(r.cores))
	if s.have && len(s.prev.cores) == len(r.cores) {
		var user, sys, total float64
		for i, c := range r.cores {
			p := s.prev.cores[i]
			du, ds, di := delta(c.User, p.User), delta(c.System, p.System), delta(c.Idle, p.Idle)
			if t := du + ds + di; t > 0 {
				snap.Cores[i] = float64(du+ds) / float64(t)
			}
			user, sys, total = user+float64(du), sys+float64(ds), total+float64(du+ds+di)
		}
		if total > 0 {
			snap.User, snap.System = user/total, sys/total
			snap.CPU = snap.User + snap.System
		}
		if dt := now.Sub(s.prevT).Seconds(); dt > 0 {
			snap.RxRate = float64(delta(r.rx, s.prev.rx)) / dt
			snap.TxRate = float64(delta(r.tx, s.prev.tx)) / dt
		}
	}
	if procs {
		snap.Procs = s.procRates(r.procs, now)
	}
	s.prev, s.prevT, s.have = r, now, true
	return snap
}

// delta is the growth of a counter; a counter that went back (an interface reset) counts as no growth.
func delta(now, before uint64) uint64 {
	if now < before {
		return 0
	}
	return now - before
}

func (s *Sampler) procRates(cur map[int]rawProc, now time.Time) []Process {
	dt := now.Sub(s.pT).Seconds()
	out := make([]Process, 0, len(cur))
	for pid, p := range cur {
		pr := Process{PID: pid, Name: p.Name, RSS: p.RSS}
		// a reused PID shows as another name: its first rate is 0, not a jump
		if old, ok := s.pprev[pid]; ok && dt > 0 && old.Name == p.Name {
			pr.CPU = float64(delta(p.CPU, old.CPU)) / 1e9 / dt * 100
		}
		out = append(out, pr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	s.pprev, s.pT = cur, now
	return out
}
