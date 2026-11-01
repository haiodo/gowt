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
)

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
