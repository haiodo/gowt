package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// supervise runs the tests in a child process. AppKit aborts the process on an uncaught NSException
// (purego cannot catch it), so a crash is reported as a failure of the running test and the run
// resumes with the next one in a fresh child.
func supervise() {
	start := time.Now()
	var passed, failed, skipped, seen int
	for {
		args := []string{"-child", "-json", "-skip", strconv.Itoa(seen), "-timeout", timeoutFlag.String()}
		if *runFlag != "" {
			args = append(args, "-run", *runFlag)
		}
		if *tagFlag != "" {
			args = append(args, "-tag", *tagFlag)
		}
		cmd := exec.Command(os.Args[0], args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintln(os.Stderr, "swttest:", err)
			os.Exit(1)
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "swttest:", err)
			os.Exit(1)
		}
		running, message := "", ""
		before := seen
		sc := bufio.NewScanner(out)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var ev testEvent
			if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Test == "" {
				continue
			}
			switch ev.Action {
			case "run":
				running, message = ev.Test, ""
				seen++
				event("run", ev.Test, 0, "")
			case "output":
				message = strings.TrimSuffix(ev.Output, "\n")
			case "pass", "fail", "skip":
				report(ev.Test, result{strings.ToUpper(ev.Action), message, time.Duration(ev.Elapsed * float64(time.Second))})
				collected = append(collected, outcome{ev.Test, strings.ToUpper(ev.Action), message})
				running = ""
				switch ev.Action {
				case "pass":
					passed++
				case "fail":
					failed++
				default:
					skipped++
				}
			}
		}
		werr := cmd.Wait()
		if running != "" {
			failed++
			msg := "process died: " + crashReason(stderr.String(), werr)
			report(running, result{"FAIL", msg, 0})
			collected = append(collected, outcome{running, "FAIL", msg})
			continue
		}
		if werr == nil {
			break
		}
		// The child's own timeout abort has already reported its test. Without it, a death before any
		// test started would restart into the same crash forever.
		if exitCode(werr) != exitTimeout || seen == before {
			fmt.Fprintln(os.Stderr, "swttest: child failed outside a test:", crashReason(stderr.String(), werr))
			os.Exit(1)
		}
	}
	summarize(passed, failed, skipped, start)
	full := *runFlag == "" && *tagFlag == ""
	if *updateFlag != "" {
		if !full {
			fmt.Fprintln(os.Stderr, "swttest: -update needs a full run (no -run/-tag)")
			os.Exit(1)
		}
		if err := update(*updateFlag); err != nil {
			fmt.Fprintln(os.Stderr, "swttest:", err)
			os.Exit(1)
		}
	}
	if *expectedFlag != "" && gate(*expectedFlag, full) > 0 {
		os.Exit(1)
	}
}

func exitCode(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}

// crashReason is the NSException line if AppKit printed one, else the first fatal/signal line.
func crashReason(stderr string, err error) string {
	for _, marker := range []string{"reason:", "fatal error:", "SIG", "panic:"} {
		for _, line := range strings.Split(stderr, "\n") {
			if i := strings.Index(line, marker); i >= 0 {
				return strings.TrimSpace(line[i:])
			}
		}
	}
	return fmt.Sprint(err)
}
