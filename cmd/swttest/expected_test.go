package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateMissingFlaky(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expected.txt")
	if err := os.WriteFile(path, []byte("flaky\tClass.test\tintermittent failure\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := collected
	collected = nil
	t.Cleanup(func() { collected = old })
	if got := gate(path, true); got != 1 {
		t.Fatalf("missing flaky test: got %d problems, want 1", got)
	}
	if got := gate(path, false); got != 0 {
		t.Fatalf("filtered run: got %d problems, want 0", got)
	}
	collected = []outcome{{"Class.test", "PASS", ""}}
	if got := gate(path, true); got != 0 {
		t.Fatalf("present flaky test: got %d problems, want 0", got)
	}
}

func TestMain(m *testing.M) {
	if status := os.Getenv("GOWT_RUNNER_TEST_CHILD_STATUS"); status != "" {
		if len(os.Args) <= 1 || os.Args[1] != "-child" {
			os.Exit(supervise())
		}
		if status == "oversized" {
			fmt.Println(strings.Repeat("x", 1<<20+1))
			os.Exit(0)
		}
		fmt.Println(`{"Action":"run","Test":"Class.test"}`)
		fmt.Printf("{\"Action\":%q,\"Test\":\"Class.test\"}\n", status)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSuperviseWithoutExpected(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   int
	}{{"pass", 0}, {"fail", 1}, {"skip", 0}, {"oversized", 1}} {
		t.Run(tc.status, func(t *testing.T) {
			cmd := exec.Command(os.Args[0])
			cmd.Env = append(os.Environ(), "GOWT_RUNNER_TEST_CHILD_STATUS="+tc.status)
			output, err := cmd.CombinedOutput()
			got := 0
			if err != nil {
				got = exitCode(err)
			}
			if got != tc.want {
				t.Fatalf("supervise: got exit %d, want %d: %s", got, tc.want, output)
			}
			if tc.status == "oversized" && !strings.Contains(string(output), "reading child output: bufio.Scanner: token too long") {
				t.Fatalf("missing scanner error: %s", output)
			}
		})
	}
}
