// Command fieldcheck runs gowt's checks on a real machine and writes one zip file with the environment,
// the results of every suite, the full logs and a few window snapshots. See docs/field-check.md.
//
//	go run ./cmd/fieldcheck [-quick] [-only suite[,suite]] [-out dir]
//
// It opens windows (except for -only env) and never modifies a tracked file.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/haiodo/gowt"
)

// AppKit must run on the process's main thread (the -probe mode runs a Display).
func init() { runtime.LockOSThread() }

var (
	quickFlag = flag.Bool("quick", false, "skip the full swttest: run a few widget classes of it (the gate then checks only those)")
	onlyFlag  = flag.String("only", "", "comma-separated suites to run: "+strings.Join(suiteNames, ", "))
	outFlag   = flag.String("out", ".", "directory for the result zip")
	probeFlag = flag.Bool("probe", false, "internal: print the display facts as JSON and exit")
)

func main() {
	flag.Parse()
	if *probeFlag {
		runProbe()
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fieldcheck:", err)
		os.Exit(1)
	}
}

func run() error {
	if b, err := os.ReadFile("go.mod"); err != nil || !strings.HasPrefix(string(b), "module github.com/haiodo/gowt\n") {
		return fmt.Errorf("run it from the root of a gowt checkout: go run ./cmd/fieldcheck")
	}
	selected, err := selectSuites(*onlyFlag)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	work, err := os.MkdirTemp("", "gowt-fieldcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	for _, d := range []string{"logs", "snapshots", "bin"} {
		if err := os.MkdirAll(filepath.Join(work, d), 0o755); err != nil {
			return err
		}
	}

	rep := Report{Schema: 1, Created: time.Now().UTC().Format(time.RFC3339), Args: []string{fmt.Sprintf("-quick=%v", *quickFlag), "-only=" + *onlyFlag}, Env: collectEnv()}
	r := &runner{ctx: ctx, work: work, quick: *quickFlag, rep: &rep}
	if skip := r.guiSkipReason(); skip != "" {
		fmt.Println("note:", skip)
	}
	for _, s := range selected {
		if ctx.Err() != nil {
			break
		}
		fmt.Printf("== %s\n", s.name)
		res := s.run(r)
		res.Name = s.name
		fmt.Printf("   %s\n", res.oneLine())
		rep.Suites = append(rep.Suites, res)
	}
	if ctx.Err() != nil {
		rep.Interrupted = true
	}
	rep.Finish()

	date := time.Now().Format("2006-01-02")
	base := fmt.Sprintf("gowt-fieldcheck-%s-%s-%s", runtime.GOOS, runtime.GOARCH, date)
	zipPath := filepath.Join(*outFlag, base+".zip")
	for n := 2; fileExists(zipPath); n++ {
		zipPath = filepath.Join(*outFlag, fmt.Sprintf("%s-%d.zip", base, n))
	}
	js, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(work, "report.json"), js, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(work, "report.md"), []byte(rep.Markdown()), 0o644); err != nil {
		return err
	}
	if err := writeZip(zipPath, work); err != nil {
		return err
	}
	abs, _ := filepath.Abs(zipPath)
	fmt.Printf("\n%s\n%s\n", abs, rep.Summary())
	fmt.Println("Attach this one file to a new issue: choose the \"Field check\" template at https://github.com/haiodo/gowt/issues/new/choose")
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// runProbe reads the display facts from a gowt app that opens no window.
func runProbe() {
	var p Probe
	err := gowt.Run(func(app *gowt.App) {
		p.Dark = app.Dark()
		p.ScreenWidth, p.ScreenHeight = app.ScreenSize()
		p.ZoomPercent = int(app.Unwrap().GetPrimaryMonitor().GetZoom())
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe:", err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(p)
}

func selectSuites(only string) ([]suite, error) {
	if only == "" {
		return suites, nil
	}
	var out []suite
	for _, n := range strings.Split(only, ",") {
		i := slices.IndexFunc(suites, func(s suite) bool { return s.name == strings.TrimSpace(n) })
		if i < 0 {
			return nil, fmt.Errorf("unknown suite %q, have: %s", n, strings.Join(suiteNames, ", "))
		}
		out = append(out, suites[i])
	}
	return out, nil
}
