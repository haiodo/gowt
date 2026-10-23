package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Failure is one failing test with the reason the suite printed.
type Failure struct {
	Test    string `json:"test"`
	Package string `json:"package,omitempty"`
	Reason  string `json:"reason"`
}

// Counts are the totals of one suite.
type Counts struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

const (
	maxFailures = 500
	maxReason   = 400
)

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

var (
	swtLine    = regexp.MustCompile(`^(PASS|FAIL|SKIP)\s+(\S+) \(\d+\.\d+s\)(?:: (.*))?$`)
	swtSummary = regexp.MustCompile(`^(\d+) passed, (\d+) failed, (\d+) skipped, (\d+) total in `)
	gateFailed = regexp.MustCompile(`^GATE FAILED: (\d+) problems`)
)

// SwtResult is what cmd/swttest printed: its summary line, failures and the verdict of the -expected gate.
type SwtResult struct {
	Counts
	Summary      string
	Failures     []Failure
	Gate         string // "ok", "failed", or "" when the log has no gate line (the run died before it)
	GateProblems int
	GateLines    []string
}

// parseSwttest reads the plain output of `swttest -expected`.
func parseSwttest(r io.Reader) SwtResult {
	var res SwtResult
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if m := swtLine.FindStringSubmatch(line); m != nil {
			if m[1] == "FAIL" && len(res.Failures) < maxFailures {
				res.Failures = append(res.Failures, Failure{Test: m[2], Reason: clip(m[3], maxReason)})
			}
			continue
		}
		if m := swtSummary.FindStringSubmatch(line); m != nil {
			res.Passed, _ = strconv.Atoi(m[1])
			res.Failed, _ = strconv.Atoi(m[2])
			res.Skipped, _ = strconv.Atoi(m[3])
			res.Total, _ = strconv.Atoi(m[4])
			res.Summary = line
			continue
		}
		switch {
		case line == "GATE ok":
			res.Gate = "ok"
		case gateFailed.MatchString(line):
			res.Gate = "failed"
			res.GateProblems, _ = strconv.Atoi(gateFailed.FindStringSubmatch(line)[1])
		case strings.HasPrefix(line, "GATE ") && len(res.GateLines) < maxFailures:
			res.GateLines = append(res.GateLines, clip(line, maxReason))
		}
	}
	return res
}

type goEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

// GoTestResult is a `go test -json` run.
type GoTestResult struct {
	Counts
	Failures []Failure
	Packages map[string]string // package -> pass, fail or skip
}

// parseGoTestJSON reads test2json events. The tail of a failing test's output is its reason; a package
// that fails without a failing test (build error, panic in TestMain, timeout) is reported as "(package)".
func parseGoTestJSON(r io.Reader) GoTestResult {
	res := GoTestResult{Packages: map[string]string{}}
	out := map[string][]string{} // package + test -> output lines
	failedPkg := map[string]bool{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var ev goEvent
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		key := ev.Package + "\x00" + ev.Test
		switch ev.Action {
		case "output":
			l := out[key]
			if len(l) >= 40 {
				l = l[1:]
			}
			out[key] = append(l, strings.TrimRight(ev.Output, "\r\n"))
		case "pass", "fail", "skip":
			if ev.Test == "" {
				res.Packages[ev.Package] = ev.Action
				if ev.Action == "fail" && !failedPkg[ev.Package] && len(res.Failures) < maxFailures {
					res.Failures = append(res.Failures, Failure{Test: "(package)", Package: ev.Package, Reason: reasonOf(out[key])})
				}
				continue
			}
			res.Total++
			switch ev.Action {
			case "pass":
				res.Passed++
			case "skip":
				res.Skipped++
			default:
				res.Failed++
				failedPkg[ev.Package] = true
				if len(res.Failures) < maxFailures {
					res.Failures = append(res.Failures, Failure{Test: ev.Test, Package: ev.Package, Reason: reasonOf(out[key])})
				}
			}
		}
	}
	return res
}

// reasonOf drops the === RUN and --- FAIL bookkeeping lines and joins the rest.
func reasonOf(lines []string) string {
	var keep []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "=== ") || strings.HasPrefix(t, "--- ") || t == "FAIL" || t == "PASS" {
			continue
		}
		keep = append(keep, t)
	}
	return clip(strings.Join(keep, " | "), maxReason)
}

// Expected counts the lines per status of a tests/expected*.txt file.
type Expected struct {
	File  string `json:"file"`
	Pass  int    `json:"pass"`
	Fail  int    `json:"fail"`
	Skip  int    `json:"skip"`
	Flaky int    `json:"flaky"`
}

func readExpectedCounts(path string) (Expected, error) {
	e := Expected{File: path}
	f, err := os.Open(path)
	if err != nil {
		return e, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		status, _, _ := strings.Cut(sc.Text(), "\t")
		switch status {
		case "pass":
			e.Pass++
		case "fail":
			e.Fail++
		case "skip":
			e.Skip++
		case "flaky":
			e.Flaky++
		}
	}
	return e, sc.Err()
}
