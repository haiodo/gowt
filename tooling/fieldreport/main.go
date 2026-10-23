// Command fieldreport prints one table row per field-check zip (cmd/fieldcheck):
//
//	go run ./tooling/fieldreport gowt-fieldcheck-*.zip
//
// swt and jface cells are pass/fail/skip of the run, then the gate verdict; "exp" is the same
// counts from the expected file of that OS, so a difference is visible at a glance.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// report is the part of report.json this tool reads.
type report struct {
	Verdict string `json:"verdict"`
	Env     struct {
		GOARCH string            `json:"goarch"`
		OS     map[string]string `json:"os"`
		Git    map[string]string `json:"git"`
	} `json:"env"`
	Suites []struct {
		Name     string                                 `json:"name"`
		Status   string                                 `json:"status"`
		Gate     string                                 `json:"gate"`
		Counts   *struct{ Passed, Failed, Skipped int } `json:"counts"`
		Expected *struct{ Pass, Fail, Skip int }        `json:"expected"`
		Failures []json.RawMessage                      `json:"failures"`
	} `json:"suites"`
}

var columns = []string{"swt", "jface", "gui", "qa", "webview", "snapshots"}

func read(path string) (*report, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "report.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		var r report
		return &r, json.Unmarshal(b, &r)
	}
	return nil, fmt.Errorf("no report.json")
}

func (r *report) cell(name string) string {
	for _, s := range r.Suites {
		if s.Name != name {
			continue
		}
		if s.Counts == nil {
			if name == "snapshots" {
				return fmt.Sprintf("%s, %d failed", s.Status, len(s.Failures))
			}
			return s.Status
		}
		c := fmt.Sprintf("%d/%d/%d", s.Counts.Passed, s.Counts.Failed, s.Counts.Skipped)
		if s.Gate != "" {
			c += " gate " + s.Gate
		} else if s.Status != "ok" {
			c += " " + s.Status
		}
		if s.Expected != nil {
			c += fmt.Sprintf(" (exp %d/%d/%d)", s.Expected.Pass, s.Expected.Fail, s.Expected.Skip)
		}
		return c
	}
	return "-"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tooling/fieldreport <fieldcheck.zip>...")
		os.Exit(2)
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "file\tOS\tarch\tgowt\tverdict\t"+strings.Join(columns, "\t"))
	bad := 0
	for _, p := range os.Args[1:] {
		r, err := read(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", p, err)
			bad++
			continue
		}
		commit := r.Env.Git["commit"]
		if len(commit) > 9 {
			commit = commit[:9]
		}
		if r.Env.Git["dirty"] == "true" {
			commit += "+dirty"
		}
		osName := strings.TrimSpace(r.Env.OS["name"] + " " + r.Env.OS["version"] + " " + r.Env.OS["build"])
		row := []string{filepath.Base(p), osName, r.Env.GOARCH, commit, r.Verdict}
		for _, c := range columns {
			row = append(row, r.cell(c))
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
	if bad > 0 {
		os.Exit(1)
	}
}
