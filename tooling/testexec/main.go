// testexec records test process termination for go test -exec.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	var monitor *exec.Cmd
	var monitorLog *os.File
	crashDir := os.Getenv("GOWT_CRASH_DIR")
	procdump := os.Getenv("GOWT_PROCDUMP")
	capture := procdump != "" && crashDir != "" && filepath.Base(os.Args[1]) == "look.test.exe"
	if capture {
		logPath := filepath.Join(crashDir, fmt.Sprintf("procdump-%d.log", cmd.Process.Pid))
		logErr := os.MkdirAll(crashDir, 0755)
		if logErr == nil {
			monitorLog, logErr = os.Create(logPath)
		}
		if logErr != nil {
			fmt.Fprintf(os.Stderr, "testexec: procdump log=%q error=%v\n", logPath, logErr)
		} else {
			fmt.Fprintf(os.Stderr, "testexec: procdump log=%q\n", logPath)
			monitor = exec.Command(procdump, "-accepteula", "-ma", "-e", "1", "-f", "C0000005", "-n", "1", strconv.Itoa(cmd.Process.Pid), crashDir)
			monitor.Stdout, monitor.Stderr = monitorLog, monitorLog
			if err := monitor.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "testexec: procdump start error=%v\n", err)
				monitor = nil
			}
		}
	}
	err := cmd.Wait()
	code := cmd.ProcessState.ExitCode()
	fmt.Fprintf(os.Stderr, "testexec: complete pid=%d duration=%s exit=%d hex=0x%08X error=%v\n", cmd.Process.Pid, time.Since(started), code, uint32(code), err)
	if monitor != nil {
		done := make(chan error, 1)
		go func() { done <- monitor.Wait() }()
		select {
		case err := <-done:
			fmt.Fprintf(os.Stderr, "testexec: procdump complete error=%v\n", err)
		case <-time.After(15 * time.Second):
			fmt.Fprintf(os.Stderr, "testexec: procdump timeout error=%v\n", monitor.Process.Kill())
			<-done
		}
	}
	if monitorLog != nil {
		monitorLog.Close()
	}
	if capture && code != 0 {
		binary, copyErr := os.ReadFile(os.Args[1])
		if copyErr == nil {
			copyErr = os.WriteFile(filepath.Join(crashDir, fmt.Sprintf("%d-%s", cmd.Process.Pid, filepath.Base(os.Args[1]))), binary, 0755)
		}
		if copyErr != nil {
			fmt.Fprintf(os.Stderr, "testexec: copy executable error=%v\n", copyErr)
		}
	}
	if code < 0 {
		code = 1
	}
	os.Exit(code)
}
