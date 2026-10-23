package main

import (
	"archive/zip"
	"os"
	"strings"
	"testing"
)

func TestReportZip(t *testing.T) {
	f, err := os.Open("testdata/swttest.txt")
	if err != nil {
		t.Fatal(err)
	}
	res := parseSwttest(f)
	f.Close()
	rep := Report{Schema: 1, Env: Env{GOARCH: "arm64", OS: map[string]string{"name": "TestOS"}}, Suites: []SuiteResult{
		{Name: "swt", Status: "fail", Counts: &res.Counts, Gate: res.Gate, GateLines: res.GateLines, Failures: res.Failures},
	}}
	rep.Finish()
	if rep.Verdict != "fail" || !strings.Contains(rep.Markdown(), "Test_org_eclipse_swt_widgets_Tree.test_Virtual") {
		t.Errorf("verdict %q\n%s", rep.Verdict, rep.Markdown())
	}
	dir := t.TempDir()
	for _, d := range []string{"logs", "snapshots"} {
		if err := os.MkdirAll(dir+"/"+d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{"report.md": "m", "report.json": "{}", "logs/swt.log": "l"} {
		if err := os.WriteFile(dir+"/"+name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeZip(dir+"/out.zip", dir); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(dir + "/out.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	if strings.Join(names, ",") != "report.md,report.json,logs/swt.log" {
		t.Errorf("zip entries %v", names)
	}
}
