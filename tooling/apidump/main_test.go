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

func TestFacadeDiff(t *testing.T) {
	removed, added := facadeDiff([]string{"gowt func A()", "gowt func B(x int)"}, []string{"gowt func A()", "gowt func B(x string)"})
	if len(removed) != 1 || removed[0] != "gowt func B(x int)" || len(added) != 1 || added[0] != "gowt func B(x string)" {
		t.Errorf("removed = %v, added = %v", removed, added)
	}
}

func TestFacadeDiffFieldAndConst(t *testing.T) {
	removed, _ := facadeDiff([]string{"gowt Grid.Margin field int"}, []string{"gowt Grid.Gap field int"})
	if len(removed) != 1 {
		t.Errorf("renamed field not reported: %v", removed)
	}
	removed, _ = facadeDiff([]string{"gowt const AlignCenter Align = 1"}, []string{"gowt const AlignCenter Align = 2"})
	if len(removed) != 1 {
		t.Errorf("changed const value not reported: %v", removed)
	}
}
