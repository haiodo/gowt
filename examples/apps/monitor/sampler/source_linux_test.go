package sampler

import "testing"

func TestParseStat(t *testing.T) {
	got := parseStat("cpu  10 1 5 80 2 0 1 0\ncpu0 6 0 3 40 1 0 1 0\ncpu1 4 1 2 40 1 0 0 0\nintr 1\n")
	if len(got) != 2 {
		t.Fatalf("cores %d, want 2 (the total line is not a core)", len(got))
	}
	if want := (cpuTimes{User: 6, System: 4, Idle: 41}); got[0] != want {
		t.Errorf("cpu0 %+v, want %+v", got[0], want)
	}
}

func TestParsePidStat(t *testing.T) {
	// the name holds a space and a parenthesis: fields are counted from the last ")"
	line := "42 (my (odd) proc) S 1 42 42 0 -1 4194560 100 0 0 0 150 50 0 0 20 0 3 0 1000 1000000 500 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0\n"
	p, ok := parsePidStat(line, 4096)
	if !ok || p.Name != "my (odd) proc" || p.CPU != 200*nsPerTick || p.RSS != 500*4096 {
		t.Errorf("%+v %v", p, ok)
	}
}
