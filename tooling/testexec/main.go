// testexec records test process termination for go test -exec.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "testexec: missing executable")
		os.Exit(1)
	}
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	started := time.Now()
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "testexec: start executable=%q args=%q error=%v\n", os.Args[1], os.Args[2:], err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "testexec: start pid=%d executable=%q args=%q\n", cmd.Process.Pid, os.Args[1], os.Args[2:])
	err := cmd.Wait()
	code := cmd.ProcessState.ExitCode()
	fmt.Fprintf(os.Stderr, "testexec: complete pid=%d duration=%s exit=%d hex=0x%08X error=%v\n", cmd.Process.Pid, time.Since(started), code, uint32(code), err)
	if code < 0 {
		code = 1
	}
	os.Exit(code)
}
