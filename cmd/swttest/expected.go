package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

// tests/expected.txt is the regression gate: one line per test, tab separated,
//
//	pass   Class.method
//	fail   Class.method   reason
//	skip   Class.method   reason
//	flaky  Class.method   reason      (either outcome is accepted)
//
// A run fails if an expected pass no longer passes, a test fails without a described reason
// (not listed, or a reason starting with "UNDESCRIBED"), or - for a full run - a listed test is gone.

type outcome struct {
	name, status, message string
}

type expectation struct{ status, reason string }

// collected is what supervise saw, in run order.
var collected []outcome

func readExpected(path string) (map[string]expectation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]expectation{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.SplitN(line, "\t", 3)
		if len(f) < 2 {
			return nil, fmt.Errorf("%s: bad line %q", path, line)
		}
		e := expectation{status: f[0]}
		if len(f) == 3 {
			e.reason = f[2]
		}
		out[f[1]] = e
	}
	return out, sc.Err()
}

// gate compares the run with the expectations and returns the number of problems (0: green).
func gate(path string, full bool) int {
	want, err := readExpected(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "swttest:", err)
		return 1
	}
	problems, better, ran := 0, 0, map[string]bool{}
	for _, r := range collected {
		ran[r.name] = true
		e, listed := want[r.name]
		switch {
		case e.status == "flaky":
		case !listed && r.status == "FAIL":
			fmt.Printf("GATE undescribed failure: %s: %s\n", r.name, r.message)
			problems++
		case !listed:
			better++
		case r.status == "FAIL" && (e.status == "fail" && strings.HasPrefix(e.reason, "UNDESCRIBED") || e.status != "fail"):
			fmt.Printf("GATE %s: %s: %s\n", verdict(e), r.name, r.message)
			problems++
		case e.status == "pass" && r.status != "PASS":
			fmt.Printf("GATE regression: %s expected pass, got %s: %s\n", r.name, r.status, r.message)
			problems++
		case e.status != "pass" && r.status == "PASS":
			better++
		}
	}
	if full {
		for name, e := range want {
			if !ran[name] && e.status != "flaky" {
				fmt.Printf("GATE missing: %s is in %s but was not run\n", name, path)
				problems++
			}
		}
	}
	if better > 0 {
		fmt.Printf("GATE %d tests differ upward (pass, or not listed): make test-swt-update to record them\n", better)
	}
	if problems > 0 {
		fmt.Printf("GATE FAILED: %d problems\n", problems)
	} else {
		fmt.Println("GATE ok")
	}
	return problems
}

func verdict(e expectation) string {
	if e.status == "fail" {
		return "undescribed failure (" + e.reason + ")"
	}
	return "regression"
}

// update rewrites the expectations from this run: passes as pass, failures keep a listed reason and
// otherwise get an UNDESCRIBED one that the gate rejects until someone writes the cause.
func update(path string) error {
	old, _ := readExpected(path)
	slices.SortFunc(collected, func(a, b outcome) int { return strings.Compare(a.name, b.name) })
	var b strings.Builder
	b.WriteString("# Regression gate for make test-swt (cmd/swttest -expected). Format and refresh: tooling/j2go/README.md \"Round 14 widget tests\".\n")
	for _, r := range collected {
		e, listed := old[r.name]
		one := strings.ReplaceAll(strings.ReplaceAll(r.message, "\t", " "), "\n", " ")
		switch {
		case listed && e.status == "flaky":
			fmt.Fprintf(&b, "flaky\t%s\t%s\n", r.name, e.reason)
		case r.status == "PASS":
			fmt.Fprintf(&b, "pass\t%s\n", r.name)
		case r.status == "SKIP":
			fmt.Fprintf(&b, "skip\t%s\t%s\n", r.name, one)
		case listed && e.status == "fail" && !strings.HasPrefix(e.reason, "UNDESCRIBED"):
			fmt.Fprintf(&b, "fail\t%s\t%s\n", r.name, e.reason)
		default:
			fmt.Fprintf(&b, "fail\t%s\tUNDESCRIBED: %s\n", r.name, one)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
