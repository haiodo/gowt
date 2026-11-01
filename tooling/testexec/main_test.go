package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func init() {
	if os.Getenv("TESTEXEC_FAKE_MONITOR") == "" || len(os.Args) < 2 || os.Args[1] != "-accepteula" {
		return
	}
	fmt.Printf("args=%q\n", os.Args[1:])
	fmt.Fprintln(os.Stderr, "monitor stderr")
	if os.Getenv("TESTEXEC_FAKE_MONITOR") == "hang" {
		time.Sleep(time.Minute)
	}
	os.Exit(9)
}

func TestProcDump(t *testing.T) {
	dir := t.TempDir()
	wrapper := filepath.Join(dir, "testexec.exe")
	if out, err := exec.Command("go", "build", "-o", wrapper, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "look.test.exe")
	if err := os.WriteFile(child, binary, 0755); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"fail", "missing", "hang"} {
		t.Run(mode, func(t *testing.T) {
			crashDir := filepath.Join(t.TempDir(), "crash")
			monitor := os.Args[0]
			if mode == "missing" {
				monitor = filepath.Join(dir, "missing.exe")
			}
			cmd := exec.Command(wrapper, child, "-test.run=^TestChild$", "--", "one", "two")
			cmd.Env = append(os.Environ(), "TESTEXEC_CHILD_EXIT=23", "TESTEXEC_FAKE_MONITOR="+mode, "GOWT_PROCDUMP="+monitor, "GOWT_CRASH_DIR="+crashDir)
			started := time.Now()
			out, err := cmd.CombinedOutput()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 23 {
				t.Fatalf("child status changed: %v\n%s", err, out)
			}
			logs, err := filepath.Glob(filepath.Join(crashDir, "procdump-*.log"))
			if err != nil || len(logs) != 1 {
				t.Fatalf("monitor logs: %v %v\n%s", logs, err, out)
			}
			pid := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(logs[0]), "procdump-"), ".log")
			copied, err := os.ReadFile(filepath.Join(crashDir, pid+"-look.test.exe"))
			if err != nil || !bytes.Equal(copied, binary) {
				t.Fatalf("failed executable copy: %v", err)
			}
			if !strings.Contains(string(out), strconv.Quote(logs[0])) {
				t.Fatalf("missing log path: %s", out)
			}
			if mode == "missing" {
				if !strings.Contains(string(out), "procdump start error=") {
					t.Fatalf("missing start error: %s", out)
				}
				return
			}
			log, err := os.ReadFile(logs[0])
			want := fmt.Sprintf("args=%q", []string{"-accepteula", "-ma", "-e", "1", "-f", "C0000005", "-n", "1", pid, crashDir})
			if err != nil || !strings.Contains(string(log), want) || !strings.Contains(string(log), "monitor stderr") {
				t.Fatalf("monitor output: %v\n%s", err, log)
			}
			if mode == "hang" && (!strings.Contains(string(out), "procdump timeout") || time.Since(started) > 25*time.Second) {
				t.Fatalf("monitor timeout failed after %s: %s", time.Since(started), out)
			}
		})
	}
	for _, executable := range []string{child, os.Args[0]} {
		t.Run("disabled-"+filepath.Base(executable), func(t *testing.T) {
			crashDir := t.TempDir()
			monitor := os.Args[0]
			if executable == child {
				monitor = ""
			}
			cmd := exec.Command(wrapper, executable, "-test.run=^TestChild$", "--", "one", "two")
			cmd.Env = append(os.Environ(), "TESTEXEC_CHILD_EXIT=23", "GOWT_PROCDUMP="+monitor, "GOWT_CRASH_DIR="+crashDir)
			out, err := cmd.CombinedOutput()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 23 || strings.Contains(string(out), "procdump") {
				t.Fatalf("disabled monitor: %v\n%s", err, out)
			}
			entries, err := os.ReadDir(crashDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("disabled monitor wrote files: %v %v", entries, err)
			}
		})
	}
}

func TestTestexec(t *testing.T) {
	wrapper := filepath.Join(t.TempDir(), "testexec.exe")
	if out, err := exec.Command("go", "build", "-o", wrapper, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	codes := []int{0, 23}
	if runtime.GOOS == "windows" {
		codes = append(codes, 0xC0000005)
	}
	for _, code := range codes {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			cmd := exec.Command(wrapper, os.Args[0], "-test.run=^TestChild$", "--", "argument with spaces", "literal\"quote")
			cmd.Env = append(os.Environ(), "TESTEXEC_CHILD_EXIT="+strconv.Itoa(code))
			cmd.Stdin = strings.NewReader("stdin payload")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != code {
				t.Fatalf("exit: %v, wanted %d; stderr: %s", err, code, &stderr)
			}
			if stdout.String() != "stdin payload [\"argument with spaces\" \"literal\\\"quote\"]" {
				t.Errorf("stdout/args: %q", stdout.String())
			}
			for _, want := range []string{"child stderr", "testexec: start pid=", "testexec: complete pid=", "duration=", fmt.Sprintf("exit=%d hex=0x%08X", code, code)} {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr missing %q: %s", want, &stderr)
				}
			}
		})
	}
	cmd := exec.Command(wrapper, filepath.Join(t.TempDir(), "missing"))
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "testexec: start executable=") {
		t.Fatalf("start failure: %v, %s", err, out)
	}
}

func TestChild(t *testing.T) {
	value := os.Getenv("TESTEXEC_CHILD_EXIT")
	if value == "" {
		return
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s %q", input, os.Args[len(os.Args)-2:])
	fmt.Fprintln(os.Stderr, "child stderr")
	code, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	os.Exit(code)
}
