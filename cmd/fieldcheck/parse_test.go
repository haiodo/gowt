package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseSwttest(t *testing.T) {
	f, err := os.Open("testdata/swttest.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := parseSwttest(f)
	if r.Passed != 3 || r.Failed != 2 || r.Skipped != 1 || r.Total != 6 {
		t.Errorf("counts %+v", r.Counts)
	}
	if len(r.Failures) != 2 || r.Failures[0].Test != "Test_org_eclipse_swt_widgets_Tree.test_Virtual" ||
		!strings.HasPrefix(r.Failures[0].Reason, "SetData count") {
		t.Errorf("failures %+v", r.Failures)
	}
	if r.Gate != "failed" || r.GateProblems != 1 || len(r.GateLines) != 2 {
		t.Errorf("gate %q %d %v", r.Gate, r.GateProblems, r.GateLines)
	}
	if got := parseSwttest(strings.NewReader("1 passed, 0 failed, 0 skipped, 1 total in 0.1s\nGATE ok\n")); got.Gate != "ok" || got.Total != 1 {
		t.Errorf("ok run: %+v", got)
	}
}

func TestParseGoTestJSON(t *testing.T) {
	f, err := os.Open("testdata/gotest.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := parseGoTestJSON(f)
	if r.Passed != 1 || r.Failed != 1 || r.Skipped != 1 || r.Total != 3 {
		t.Errorf("counts %+v", r.Counts)
	}
	if len(r.Failures) != 2 {
		t.Fatalf("failures %+v", r.Failures)
	}
	if r.Failures[0].Test != "TestB" || r.Failures[0].Reason != "b_test.go:12: got 1, want 2" {
		t.Errorf("test failure %+v", r.Failures[0])
	}
	if r.Failures[1].Test != "(package)" || r.Failures[1].Package != "example.com/broken" || !strings.Contains(r.Failures[1].Reason, "undefined: x") {
		t.Errorf("package failure %+v", r.Failures[1])
	}
}

func TestParseKV(t *testing.T) {
	m := parseKV("NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 24.04.1 LTS\"\nID=ubuntu\n# c\n")
	if m["PRETTY_NAME"] != "Ubuntu 24.04.1 LTS" || m["ID"] != "ubuntu" {
		t.Errorf("%v", m)
	}
}

func TestParseReg(t *testing.T) {
	out := "\r\nHKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\r\n    ProductName    REG_SZ    Windows 10 Pro\r\n    CurrentBuild    REG_SZ    26100\r\n"
	for name, want := range map[string]string{"ProductName": "Windows 10 Pro", "CurrentBuild": "26100", "Missing": ""} {
		if v := parseReg(out, name); v != want {
			t.Errorf("%s: %q, want %q", name, v, want)
		}
	}
}

func TestReadExpectedCounts(t *testing.T) {
	e, err := readExpectedCounts("../../tests/expected_jface.txt")
	if err != nil || e.Pass == 0 {
		t.Errorf("%+v %v", e, err)
	}
}
