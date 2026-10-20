package gowt

import "testing"

func TestLauncherCount(t *testing.T) {
	for _, c := range []struct {
		text string
		n    int64
		ok   bool
	}{
		{"7", 7, true},
		{" 72 ", 72, true},
		{"0", 0, true},
		{"", 0, false},
		{"99+", 0, false},
		{"new", 0, false},
	} {
		n, ok := launcherCount(c.text)
		if n != c.n || ok != c.ok {
			t.Errorf("launcherCount(%q) = %d, %v; want %d, %v", c.text, n, ok, c.n, c.ok)
		}
	}
}
