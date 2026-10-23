package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

// TestMain builds the app with the QA server (-tags gowtqa) and hands it to qatest. Opening windows
// needs GOWT_GUI_TEST=1.
func TestMain(m *testing.M) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "qaexample")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "app")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-tags", "gowtqa", "-o", bin, ".").CombinedOutput(); err != nil {
		println(string(out))
		os.Exit(1)
	}
	qatest.Main(m, bin)
}

func TestGreet(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	app := qatest.App(t) // runs the "reset" handler first

	app.Find(qa.ID("name")).Type("Ada")
	app.Find(qa.ID("greet")).Click() // actions settle before they return
	app.Settle()                     // explicit settle: for work the test did not start itself

	if got := app.Find(qa.ID("greeting")).Node().Text; got != "Hello, Ada" {
		app.Shot("greeting") // a failed test also saves a shot and the dump; Shot is for a picture mid-test
		t.Fatalf("greeting is %q", got)
	}
	if got := string(app.State("greeting")); got != `"Hello, Ada"` {
		t.Errorf("model says %s", got)
	}
}

func TestLoad(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	app := qatest.App(t)

	// The click settles, and settle waits for the Job and its Then: no sleep, no polling.
	app.Find(qa.ID("load")).Click()

	if got := app.Find(qa.ID("greeting")).Node().Text; got != "Loaded" {
		t.Fatalf("greeting is %q", got)
	}
}
