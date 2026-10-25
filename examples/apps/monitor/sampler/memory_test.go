package sampler

import (
	"os"
	"testing"
)

func sums(t *testing.T, name string, m Memory) {
	t.Helper()
	if got := m.Used + m.Cached + m.Free; got != m.Total {
		t.Errorf("%s: used %d + cached %d + free %d = %d, want total %d", name, m.Used, m.Cached, m.Free, got, m.Total)
	}
}

// A real /proc/meminfo of the Ubuntu 26.04 VM (5.3 GB, GNOME running): the old rule counted the cache
// twice, 2.6 GB used and 2.8 GB cached of 5.3 GB left no free memory.
func TestLinuxMemoryAddsUp(t *testing.T) {
	b, err := os.ReadFile("testdata/meminfo-ubuntu-26.04")
	if err != nil {
		t.Fatal(err)
	}
	m := linuxMemory(parseMeminfo(string(b)))
	sums(t, "ubuntu", m)
	if m.Total != 5514124*1024 || m.Used != (5514124-3132580)*1024 || m.Free != 400672*1024 {
		t.Errorf("%+v", m)
	}
	if m.Cached != (3132580-400672)*1024 {
		t.Errorf("cached %d, want what is available but not free", m.Cached)
	}
	if m.SwapTotal == 0 && m.SwapUsed != 0 {
		t.Errorf("swap %+v", m)
	}
}

func TestLinuxMemoryOddInputs(t *testing.T) {
	// no MemAvailable (old kernel): derived from free, cache, buffers and reclaimable slab
	old := linuxMemory(map[string]uint64{"MemTotal": 1000, "MemFree": 100, "Cached": 300, "Buffers": 50, "SReclaimable": 50})
	sums(t, "old kernel", old)
	if old.Used != 500 {
		t.Errorf("old kernel used %d, want 500", old.Used)
	}
	// MemAvailable above MemTotal, or MemFree above MemAvailable, must not wrap around
	sums(t, "avail above total", linuxMemory(map[string]uint64{"MemTotal": 1000, "MemFree": 100, "MemAvailable": 2000}))
	sums(t, "free above avail", linuxMemory(map[string]uint64{"MemTotal": 1000, "MemFree": 900, "MemAvailable": 500}))
}

// The numbers of the Windows 11 arm64 VM: 4 GB, 2.2 GB in use, a cache as big as what is available.
func TestWindowsMemoryAddsUp(t *testing.T) {
	m := windowsMemory(4285267968, 2064568320, 1965907968)
	sums(t, "windows", m)
	if m.Used != 2220699648 || m.Cached != 1965907968 || m.Free != 98660352 {
		t.Errorf("%+v", m)
	}
	// a cache bigger than what is available (it counts working set that is in use) is capped
	m = windowsMemory(4000, 1000, 3000)
	sums(t, "big cache", m)
	if m.Cached != 1000 || m.Free != 0 {
		t.Errorf("%+v", m)
	}
}
