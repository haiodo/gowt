package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// SuiteResult is one entry of report.json.
type SuiteResult struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"` // ok, fail, timeout, error, skipped
	Note         string            `json:"note,omitempty"`
	Command      string            `json:"command,omitempty"`
	ExitCode     int               `json:"exit_code"`
	DurationS    float64           `json:"duration_s"`
	Counts       *Counts           `json:"counts,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	Gate         string            `json:"gate,omitempty"` // swt and jface: ok, failed, or none when the run died first
	GateProblems int               `json:"gate_problems,omitempty"`
	GateLines    []string          `json:"gate_lines,omitempty"`
	Expected     *Expected         `json:"expected,omitempty"`
	Failures     []Failure         `json:"failures,omitempty"`
	Packages     map[string]string `json:"packages,omitempty"`
	Log          string            `json:"log,omitempty"`
	Files        []string          `json:"files,omitempty"`
}

func (s SuiteResult) oneLine() string {
	line := s.Status
	if s.Counts != nil {
		line += fmt.Sprintf(": %d passed, %d failed, %d skipped", s.Counts.Passed, s.Counts.Failed, s.Counts.Skipped)
	}
	if s.Gate != "" {
		line += ", gate " + s.Gate
	}
	if s.Note != "" {
		line += " (" + s.Note + ")"
	}
	return line
}

type suite struct {
	name string
	run  func(*runner) SuiteResult
}

var suites = []suite{
	{"env", (*runner).envSuite},
	{"probe", (*runner).probeSuite},
	{"swt", (*runner).swtSuite},
	{"jface", (*runner).jfaceSuite},
	{"gui", (*runner).guiSuite},
	{"qa", (*runner).qaSuite},
	{"webview", (*runner).webviewSuite},
	{"snapshots", (*runner).snapshotSuite},
}

var suiteNames = func() (n []string) {
	for _, s := range suites {
		n = append(n, s.name)
	}
	return
}()

type runner struct {
	ctx   context.Context
	work  string
	quick bool
	rep   *Report
}

func (r *runner) exe(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(r.work, "bin", name)
}

// guiSkipReason is why windows cannot open here, or "".
func (r *runner) guiSkipReason() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "no DISPLAY and no WAYLAND_DISPLAY: the suites that open windows are skipped (try xvfb-run, or run in a desktop session)"
	}
	return ""
}

// cmdResult is how one child process ended.
type cmdResult struct {
	exit     int
	timedOut bool
	err      error
	dur      time.Duration
}

// exec runs a command with its output in logs/<log>, killing it after timeout (and on Ctrl+C).
// A Windows kill ends only the process it started; a child it spawned may outlive it.
func (r *runner) exec(timeout time.Duration, log string, env []string, name string, args ...string) cmdResult {
	f, err := os.Create(filepath.Join(r.work, "logs", log))
	if err != nil {
		return cmdResult{exit: -1, err: err}
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(r.ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = f, f
	cmd.Env = append(os.Environ(), env...)
	cmd.WaitDelay = 10 * time.Second
	fmt.Fprintf(f, "$ %s %v\n", name, args)
	start := time.Now()
	err = cmd.Run()
	res := cmdResult{dur: time.Since(start), timedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		res.exit = ee.ExitCode()
	default:
		res.exit, res.err = -1, err
	}
	if res.timedOut {
		fmt.Fprintf(f, "\nfieldcheck: killed after %s\n", timeout)
	}
	return res
}

func (r *runner) base(name, log string, c cmdResult, cmdline string) SuiteResult {
	s := SuiteResult{Name: name, Command: cmdline, ExitCode: c.exit, DurationS: c.dur.Seconds(), Log: "logs/" + log, Status: "ok"}
	switch {
	case c.timedOut:
		s.Status, s.Note = "timeout", "killed by the time limit"
	case r.ctx.Err() != nil:
		s.Status, s.Note = "error", "interrupted"
	case c.err != nil:
		s.Status, s.Note = "error", c.err.Error()
	case c.exit != 0:
		s.Status = "fail"
	}
	return s
}

func (r *runner) skipped(name, why string) SuiteResult {
	return SuiteResult{Name: name, Status: "skipped", Note: why}
}

func (r *runner) envSuite() SuiteResult {
	return SuiteResult{Name: "env", Status: "ok", Note: "recorded in report.json"}
}

func (r *runner) probeSuite() SuiteResult {
	if why := r.guiSkipReason(); why != "" {
		return r.skipped("probe", "no display")
	}
	log := "probe.log"
	c := r.exec(time.Minute, log, nil, os.Args[0], "-probe")
	s := r.base("probe", log, c, "fieldcheck -probe")
	b, _ := os.ReadFile(filepath.Join(r.work, "logs", log))
	// The log starts with the "$ command" line; the JSON is the line that parses.
	for _, line := range bytes.Split(b, []byte("\n")) {
		var p Probe
		if json.Unmarshal(line, &p) == nil && p.ScreenWidth > 0 {
			r.rep.Env.Display = &p
			return s
		}
	}
	if s.Status == "ok" {
		s.Status = "fail"
	}
	r.rep.Env.DisplayError = "probe failed, see " + s.Log
	return s
}

func expectedFile(jface bool) string {
	name := "tests/expected"
	if jface {
		name += "_jface"
	}
	if runtime.GOOS != "darwin" {
		name += "_" + runtime.GOOS
	}
	return name + ".txt"
}

// swttestSuite builds cmd/swttest (with -tags jface for the JFace gate) and applies the OS gate.
func (r *runner) swttestSuite(name string, jface bool, extra ...string) SuiteResult {
	if why := r.guiSkipReason(); why != "" {
		return r.skipped(name, "no display")
	}
	tool := r.exe(name)
	build := []string{"build", "-o", tool}
	if jface {
		build = []string{"build", "-tags", "jface", "-o", tool}
	}
	build = append(build, "./cmd/swttest")
	log := name + ".log"
	if c := r.exec(10*time.Minute, name+"-build.log", nil, "go", build...); c.exit != 0 || c.err != nil {
		s := r.base(name, name+"-build.log", c, "go build ./cmd/swttest")
		s.Status, s.Note = "error", "build failed"
		return s
	}
	exp := expectedFile(jface)
	args := append([]string{"-expected", exp}, extra...)
	timeout := 100 * time.Minute
	if jface {
		timeout = 20 * time.Minute
	}
	c := r.exec(timeout, log, nil, tool, args...)
	s := r.base(name, log, c, fmt.Sprintf("swttest%s -expected %s", map[bool]string{true: " (-tags jface)"}[jface], exp))
	if f, err := os.Open(filepath.Join(r.work, "logs", log)); err == nil {
		res := parseSwttest(f)
		f.Close()
		s.Counts, s.Summary, s.Failures = &res.Counts, res.Summary, res.Failures
		s.Gate, s.GateProblems, s.GateLines = res.Gate, res.GateProblems, res.GateLines
	}
	if e, err := readExpectedCounts(exp); err == nil {
		s.Expected = &e
	}
	if s.Gate == "" {
		s.Gate = "none"
	}
	if s.Gate == "failed" && s.Status == "ok" {
		s.Status = "fail"
	}
	return s
}

func (r *runner) swtSuite() SuiteResult {
	if r.quick {
		return r.swttestSuite("swt", false, "-run", `Test_org_eclipse_swt_widgets_(Button|Text|Table|Shell)\.`)
	}
	return r.swttestSuite("swt", false)
}

func (r *runner) jfaceSuite() SuiteResult { return r.swttestSuite("jface", true) }

// goTestSuite runs the GUI tests of packages with GOWT_GUI_TEST=1 and reads `go test -json`.
func (r *runner) goTestSuite(name string, timeout time.Duration, pkgs ...string) SuiteResult {
	if why := r.guiSkipReason(); why != "" {
		return r.skipped(name, "no display")
	}
	log := name + ".jsonl"
	args := append([]string{"test", "-json", "-count=1", "-timeout", (timeout - time.Minute).String()}, pkgs...)
	c := r.exec(timeout, log, []string{"GOWT_GUI_TEST=1"}, "go", args...)
	s := r.base(name, log, c, "GOWT_GUI_TEST=1 go test -json -count=1 "+fmt.Sprint(pkgs))
	// The first log line is "$ command"; the json reader skips what is not an event.
	if f, err := os.Open(filepath.Join(r.work, "logs", log)); err == nil {
		res := parseGoTestJSON(f)
		f.Close()
		s.Counts, s.Failures, s.Packages = &res.Counts, res.Failures, res.Packages
		s.Summary = fmt.Sprintf("%d passed, %d failed, %d skipped, %d total", res.Passed, res.Failed, res.Skipped, res.Total)
		if (res.Failed > 0 || len(res.Failures) > 0) && s.Status == "ok" {
			s.Status = "fail"
		}
	}
	return s
}

// guiSuite: the packages with GOWT_GUI_TEST tests (found with grep -l GOWT_GUI_TEST --include=*_test.go).
func (r *runner) guiSuite() SuiteResult {
	return r.goTestSuite("gui", 30*time.Minute, ".", "./look", "./icons", "./swt")
}

// qaSuite: the QA driver e2e tests and the example app; both build a -tags gowtqa fixture themselves.
func (r *runner) qaSuite() SuiteResult {
	return r.goTestSuite("qa", 15*time.Minute, "./qa/qatest", "./examples/qa")
}

func (r *runner) webviewSuite() SuiteResult {
	return r.goTestSuite("webview", 10*time.Minute, "./webview")
}

// demos are built and run with GOWT_SNAP, which writes the first window as a PNG and exits.
var demos = []string{"gowtdemo", "backdropdemo", "themecheck"}

func (r *runner) snapshotSuite() SuiteResult {
	if why := r.guiSkipReason(); why != "" {
		return r.skipped("snapshots", "no display")
	}
	s := SuiteResult{Name: "snapshots", Status: "ok"}
	for _, d := range demos {
		if r.ctx.Err() != nil {
			break
		}
		tool := r.exe(d)
		if c := r.exec(10*time.Minute, "snap-"+d+"-build.log", nil, "go", "build", "-o", tool, "./cmd/"+d); c.exit != 0 || c.err != nil {
			s.Status, s.Failures = "fail", append(s.Failures, Failure{Test: d, Reason: "build failed, see logs/snap-" + d + "-build.log"})
			continue
		}
		png := filepath.Join(r.work, "snapshots", d+".png")
		c := r.exec(time.Minute, "snap-"+d+".log", []string{"GOWT_SNAP=" + png, "GOWT_SNAP_DELAY=2500"}, tool)
		if fi, err := os.Stat(png); err != nil || fi.Size() == 0 {
			reason := fmt.Sprintf("no PNG, exit %d", c.exit)
			if c.timedOut {
				reason = "no PNG, killed after 1m"
			}
			s.Status, s.Failures = "fail", append(s.Failures, Failure{Test: d, Reason: reason + ", see logs/snap-" + d + ".log"})
			continue
		}
		s.Files = append(s.Files, "snapshots/"+d+".png")
	}
	s.Summary = fmt.Sprintf("%d of %d snapshots", len(s.Files), len(demos))
	return s
}
