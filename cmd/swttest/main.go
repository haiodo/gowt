// Command swttest runs the translated SWT JUnit tests (tests/swttests) on the main thread with one
// Display, the way SWT's tests expect: a panic fails its test, not the run; a hung test ends the
// run with its name. `make test-swt` runs it.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/haiodo/gowt/internal/junit"
	_ "github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
	_ "github.com/haiodo/gowt/swt/swtreflect"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

var (
	runFlag      = flag.String("run", "", "only tests whose Class.method matches this regexp")
	tagFlag      = flag.String("tag", "", "comma-separated tags: a test must have one of them; !tag excludes")
	timeoutFlag  = flag.Duration("timeout", 30*time.Second, "per-test timeout; a hung test aborts the run")
	jsonFlag     = flag.Bool("json", false, "emit go test -json compatible events")
	listFlag     = flag.Bool("list", false, "list the matching tests and exit")
	childFlag    = flag.Bool("child", false, "run in this process (a crash ends the run); default supervises a child and resumes after a crash")
	expectedFlag = flag.String("expected", "", "compare the run with this expected-results file; exit 1 on a regression or an undescribed failure")
	updateFlag   = flag.String("update", "", "rewrite this expected-results file from a full run")
	skipFlag     = flag.Int("skip", 0, "with -child: skip the first N matching tests")
)

// exitTimeout is distinct from 2, the exit code of an uncaught Go panic.
const exitTimeout = 3

type result struct {
	status  string // PASS, FAIL, SKIP
	message string
	elapsed time.Duration
}

func main() {
	flag.Parse()
	if !*childFlag && !*listFlag {
		supervise()
		return
	}
	var runRe *regexp.Regexp
	if *runFlag != "" {
		runRe = regexp.MustCompile(*runFlag)
	}
	display := newDisplay()
	var passed, failed, skipped int
	start := time.Now()
	index := 0
	for _, c := range junit.Classes {
		var tests []junit.Test
		for _, t := range c.Tests {
			if (runRe == nil || runRe.MatchString(c.Name+"."+t.Name)) && tagsMatch(t.Tags) {
				index++
				if index > *skipFlag {
					tests = append(tests, t)
				}
			}
		}
		if len(tests) == 0 {
			continue
		}
		if *listFlag {
			for _, t := range tests {
				fmt.Println(c.Name + "." + t.Name)
			}
			continue
		}
		if res := classStep(c.BeforeAll); res.status != "" {
			// JUnit: a failed @BeforeAll fails every test of the class, a failed assumption skips them.
			for _, t := range tests {
				name := c.Name + "." + t.Name
				event("run", name, 0, "")
				report(name, res)
				if res.status == "SKIP" {
					skipped++
				} else {
					failed++
				}
			}
			continue
		}
		for _, t := range tests {
			// The Display tests create their own: a second live Display is ERROR_NOT_IMPLEMENTED.
			if strings.HasSuffix(c.Name, "_widgets_Display") {
				disposeCurrent()
			} else if display.IsDisposed() {
				disposeCurrent() // a test that disposed the shared Display may have made its own
				display = newDisplay()
			}
			name := c.Name + "." + t.Name
			event("run", name, 0, "")
			r := runTest(c, t, name)
			if strings.HasSuffix(c.Name, "_widgets_Display") {
				disposeCurrent() // a failed test leaves its own Display alive
			}
			report(name, r)
			switch r.status {
			case "PASS":
				passed++
			case "FAIL":
				failed++
			default:
				skipped++
			}
		}
		if res := classStep(c.AfterAll); res.status == "FAIL" {
			name := c.Name + ".@AfterAll"
			event("run", name, 0, "")
			report(name, res)
			failed++
		}
	}
	if *listFlag {
		return
	}
	summarize(passed, failed, skipped, start)
}

func summarize(passed, failed, skipped int, start time.Time) {
	summary := fmt.Sprintf("%d passed, %d failed, %d skipped, %d total in %.1fs", passed, failed, skipped,
		passed+failed+skipped, time.Since(start).Seconds())
	if *jsonFlag {
		action := "pass"
		if failed > 0 {
			action = "fail"
		}
		event("output", "", 0, summary+"\n")
		event(action, "", time.Since(start), "")
	} else {
		fmt.Println(summary)
	}
}

type primaryFilter struct{}

// Shells land on the screen of the key window or mouse, which on a mixed-scale setup changes
// snapshots, focus and pixel bounds; a shell the test placed on the primary monitor stays put.
func (primaryFilter) HandleEvent(e *swt.Event) {
	area := e.Display.GetPrimaryMonitor().GetClientArea()
	for _, s := range e.Display.GetShells() {
		if b := s.GetBounds(); !area.Intersects(b.X, b.Y, b.Width, b.Height) {
			s.SetLocation(area.X+50, area.Y+50)
		}
	}
}

func newDisplay() *swt.Display {
	d := swt.NewDisplay()
	d.AddFilter(swt.Show, primaryFilter{})
	d.AddFilter(swt.Activate, primaryFilter{})
	return d
}

// disposeCurrent disposes the Display of this thread, if any.
func disposeCurrent() {
	defer func() { _ = recover() }()
	if d := swt.DisplayGetCurrent(); d != nil && !d.IsDisposed() {
		d.Dispose()
	}
}

// runTest runs @BeforeEach, the test and @AfterEach on a fresh instance, each step recovered.
func runTest(c *junit.Class, t junit.Test, name string) result {
	begin := time.Now()
	if t.Skip != "" {
		return result{"SKIP", t.Skip, 0}
	}
	timeout := *timeoutFlag
	if t.Timeout > 0 {
		timeout = t.Timeout
	}
	watchdog := time.AfterFunc(timeout, func() {
		report(name, result{"FAIL", fmt.Sprintf("timeout after %s, run aborted", timeout), time.Since(begin)})
		event("fail", "", 0, "")
		os.Exit(exitTimeout)
	})
	defer watchdog.Stop()

	// A parameterized test's name is "method[n]".
	junit.Current = &junit.TestInfo{Method: strings.SplitN(t.Name, "[", 2)[0]}
	var instance any
	res := step(func() { instance = c.New() })
	if res.status == "" {
		for _, fn := range c.BeforeEach {
			if res = step(func() { fn(instance) }); res.status != "" {
				break
			}
		}
		if res.status == "" {
			res = step(func() { t.Run(instance) })
		}
		for _, fn := range c.AfterEach {
			if r := step(func() { fn(instance) }); res.status == "" {
				res = r
			}
		}
	}
	if res.status == "" {
		res.status = "PASS"
	}
	res.elapsed = time.Since(begin)
	return res
}

// classStep runs the class-level fns in order and stops at the first one that panics.
func classStep(fns []func()) (res result) {
	for _, fn := range fns {
		if res = step(fn); res.status != "" {
			break
		}
	}
	return
}

// step runs fn; a panic becomes a FAIL (or SKIP for a failed assumption) with a one-line message:
// the value plus, for anything but a failed assertion, the frame that panicked.
func step(fn func()) (res result) {
	defer func() {
		r := recover()
		switch v := r.(type) {
		case nil:
		case *junit.Skipped:
			res = result{status: "SKIP", message: v.Reason}
		case *junit.AssertionFailed:
			res = result{status: "FAIL", message: v.Message + " at " + panicSite(true)}
		default:
			res = result{status: "FAIL", message: fmt.Sprintf("panic: %v at %s", r, panicSite(false))}
		}
	}()
	fn()
	return
}

// panicSite is the first stack frame outside the runtime, the junit shim and this runner.
func panicSite(assertion bool) string {
	if os.Getenv("SWTTEST_STACK") != "" {
		return string(debug.Stack())
	}
	for _, line := range strings.Split(string(debug.Stack()), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ".go:") || strings.Contains(line, "/runtime/") ||
			strings.Contains(line, "/cmd/swttest/") || assertion && strings.Contains(line, "/internal/junit/") {
			continue
		}
		if wd, err := os.Getwd(); err == nil {
			line = strings.TrimPrefix(line, wd+"/")
		}
		if i := strings.LastIndex(line, " +0x"); i >= 0 {
			line = line[:i]
		}
		return line
	}
	return "?"
}

func tagsMatch(tags []string) bool {
	if *tagFlag == "" {
		return true
	}
	wanted, matched := false, false
	for _, t := range strings.Split(*tagFlag, ",") {
		if name, ok := strings.CutPrefix(t, "!"); ok {
			if slices.Contains(tags, name) {
				return false
			}
			continue
		}
		wanted = true
		matched = matched || slices.Contains(tags, t)
	}
	return !wanted || matched
}

func report(name string, r result) {
	if *jsonFlag {
		if r.message != "" {
			event("output", name, 0, r.message+"\n")
		}
		event(strings.ToLower(r.status), name, r.elapsed, "")
		return
	}
	line := fmt.Sprintf("%-4s %s (%.3fs)", r.status, name, r.elapsed.Seconds())
	if r.message != "" {
		line += ": " + r.message
	}
	fmt.Println(line)
}

type testEvent struct {
	Time    time.Time
	Action  string
	Package string
	Test    string  `json:",omitempty"`
	Elapsed float64 `json:",omitempty"`
	Output  string  `json:",omitempty"`
}

func event(action, test string, elapsed time.Duration, output string) {
	if !*jsonFlag {
		return
	}
	b, _ := json.Marshal(testEvent{time.Now(), action, pkg, test, elapsed.Seconds(), output})
	fmt.Println(string(b))
}
