package sampler

import (
	"math"
	"os"
	"testing"
	"time"
)

type fakeSource struct{ seq []raw }

func (f *fakeSource) host() Host { return Host{Name: "fake"} }
func (f *fakeSource) close()     {}
func (f *fakeSource) read(bool) (raw, error) {
	r := f.seq[0]
	f.seq = f.seq[1:]
	return r, nil
}

func TestRates(t *testing.T) {
	s := &Sampler{}
	t0 := time.Unix(1000, 0)
	first := s.rates(raw{
		cores: []cpuTimes{{User: 100, System: 50, Idle: 850}, {User: 0, System: 0, Idle: 1000}},
		rx:    1000, tx: 500,
		procs: map[int]rawProc{7: {Name: "a", CPU: 1e9}},
	}, t0, true)
	if first.CPU != 0 || first.RxRate != 0 || first.Procs[0].CPU != 0 {
		t.Fatalf("first sample must have no rates: %+v", first)
	}
	snap := s.rates(raw{
		cores: []cpuTimes{{User: 160, System: 70, Idle: 870}, {User: 10, System: 0, Idle: 1090}},
		rx:    3000, tx: 500,
		procs: map[int]rawProc{7: {Name: "a", CPU: 2e9}, 8: {Name: "b", CPU: 5e9}},
	}, t0.Add(2*time.Second), true)
	near := func(name string, got, want float64) {
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	near("core0", snap.Cores[0], 80.0/100)
	near("core1", snap.Cores[1], 10.0/100)
	near("cpu", snap.CPU, 90.0/200)
	near("user", snap.User, 70.0/200)
	near("rx", snap.RxRate, 1000)
	near("tx", snap.TxRate, 0)
	near("proc 7", snap.Procs[0].CPU, 50) // 1 s of CPU in 2 s
	near("proc 8, new", snap.Procs[1].CPU, 0)

	// a counter that went back is not a huge rate
	back := s.rates(raw{cores: snap2cores(), rx: 10, procs: map[int]rawProc{}}, t0.Add(3*time.Second), false)
	near("reset", back.RxRate, 0)
}

func snap2cores() []cpuTimes {
	return []cpuTimes{{User: 160, System: 70, Idle: 870}, {User: 10, Idle: 1090}}
}

// TestHost reads the real machine: values only have to be plausible.
func TestHost(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Skip(err)
	}
	defer s.Close()
	h := s.Host()
	if h.Name == "" || h.Memory == 0 {
		t.Errorf("host %+v", h)
	}
	if _, err := s.Sample(true); err != nil {
		t.Fatal(err)
	}
	busy := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(busy) {
	}
	snap, err := s.Sample(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("host %+v", h)
	t.Logf("cpu %.2f cores %d load %v up %v mem %+v rx %.0f tx %.0f procs %d",
		snap.CPU, len(snap.Cores), snap.Load, snap.Uptime, snap.Mem, snap.RxRate, snap.TxRate, len(snap.Procs))
	if len(snap.Cores) == 0 || snap.CPU < 0 || snap.CPU > 1 {
		t.Errorf("cpu %v cores %v", snap.CPU, snap.Cores)
	}
	for _, c := range snap.Cores {
		if c < 0 || c > 1 {
			t.Errorf("core %v", c)
		}
	}
	m := snap.Mem
	if m.Total == 0 || m.Used == 0 || m.Used > m.Total {
		t.Errorf("memory %+v", m)
	}
	if snap.Uptime <= 0 {
		t.Errorf("uptime %v", snap.Uptime)
	}
	var self *Process
	for i := range snap.Procs {
		if snap.Procs[i].PID == os.Getpid() {
			self = &snap.Procs[i]
		}
	}
	if self == nil || self.Name == "" || self.RSS == 0 {
		t.Fatalf("own process missing or empty: %+v of %d", self, len(snap.Procs))
	}
	if self.CPU < 30 { // the busy loop ran on one core for 300 ms of the 300 ms
		t.Errorf("own CPU %.1f%%, want about 100", self.CPU)
	}
}
