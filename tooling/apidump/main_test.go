package main

import (
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	platforms := []string{"darwin", "linux"}
	report, bad, err := diff("testdata/pkg", platforms, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(report, "\n")
	for _, want := range []string{"linux lacks func OnlyMac()", "linux lacks func Sig(a int)", "darwin lacks func Sig(a string)"} {
		if !strings.Contains(got, want) {
			t.Errorf("report misses %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Both") || bad != 3 {
		t.Errorf("bad = %d, report:\n%s", bad, got)
	}

	allowed := map[string]bool{"darwin func OnlyMac()": true, "darwin func Gone()": true}
	report, _, _ = diff("testdata/pkg", platforms, allowed)
	got = strings.Join(report, "\n")
	if strings.Contains(got, "lacks func OnlyMac") || !strings.Contains(got, "stale platform-only.txt entry: darwin func Gone()") {
		t.Errorf("allowed entry not honoured or stale one not reported:\n%s", got)
	}
}
