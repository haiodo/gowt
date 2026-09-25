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
	"github.com/haiodo/gowt/swt"
	_ "github.com/haiodo/gowt/tests/swttests"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

var (
	runFlag     = flag.String("run", "", "only tests whose Class.method matches this regexp")
	tagFlag     = flag.String("tag", "", "comma-separated tags: a test must have one of them; !tag excludes")
	timeoutFlag = flag.Duration("timeout", 30*time.Second, "per-test timeout; a hung test aborts the run")
	jsonFlag    = flag.Bool("json", false, "emit go test -json compatible events")
	listFlag    = flag.Bool("list", false, "list the matching tests and exit")
)

const pkg = "github.com/haiodo/gowt/tests/swttests"

type result struct {
	status  string // PASS, FAIL, SKIP
	message string
	elapsed time.Duration
}

func main() {
	flag.Parse()
	var runRe *regexp.Regexp
	if *runFlag != "" {
		runRe = regexp.MustCompile(*runFlag)
	}
	display := swt.NewDisplay()
	var passed, failed, skipped int
	start := time.Now()
	for _, c := range junit.Classes {
		var tests []junit.Test
		for _, t := range c.Tests {
			if (runRe == nil || runRe.MatchString(c.Name+"."+t.Name)) && tagsMatch(t.Tags) {
				tests = append(tests, t)
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
		for _, fn := range c.BeforeAll {
			fn()
		}
		for _, t := range tests {
			if display.IsDisposed() {
				display = swt.NewDisplay()
			}
			name := c.Name + "." + t.Name
			event("run", name, 0, "")
			r := runTest(c, t, name)
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
		for _, fn := range c.AfterAll {
			fn()
		}
	}
	if *listFlag {
		return
	}
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
		os.Exit(2)
	})
	defer watchdog.Stop()

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
