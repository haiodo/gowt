package swt

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Needs a display (the Linux stand): the accent provider must load and survive a reset to no accent.
func TestApplyAccent(t *testing.T) {
	d := NewDisplay()
	defer d.Dispose()
	sh := NewShellDisplayStyle(d, SHELL_TRIM)
	NewLabel(&sh.Composite, NONE).SetText("accent")
	d.applyAccent([]float64{0.2, 0.4, 0.9})
	if accentProvider == 0 {
		t.Fatal("accent provider not created")
	}
	sh.Open()
	for d.ReadAndDispatch() {
	}
	d.applyAccent([]float64{-1, -1, -1})
	for d.ReadAndDispatch() {
	}
}

// Run with GOWT_GUI_TEST=1 on the Linux stand: FollowSystemTheme must not wait for a slow portal.
// A child process, since GLib keeps the first session bus it connects to.
func TestFollowSystemThemeAsync(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if os.Getenv("GOWT_FAKE_PORTAL") == "" {
		if exec.Command("python3", "-c", "import gi").Run() != nil {
			t.Skip("needs dbus-run-session and python3-gi")
		}
		script, err := filepath.Abs("testdata/fakeportal.py")
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		svc := "[D-BUS Service]\nName=org.freedesktop.portal.Desktop\nExec=/usr/bin/env python3 " + script + "\n"
		conf := `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN" "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig><type>session</type><listen>unix:tmpdir=/tmp</listen><servicedir>` + dir + `</servicedir>
<policy context="default"><allow send_destination="*" eavesdrop="true"/><allow eavesdrop="true"/><allow own="*"/></policy></busconfig>`
		if err := os.WriteFile(filepath.Join(dir, "portal.service"), []byte(svc), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bus.conf"), []byte(conf), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("dbus-run-session", "--config-file="+filepath.Join(dir, "bus.conf"), "--",
			os.Args[0], "-test.run=^TestFollowSystemThemeAsync$", "-test.v")
		cmd.Env = append(os.Environ(), "GOWT_FAKE_PORTAL=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		t.Logf("%s", out)
		return
	}
	d := NewDisplay()
	defer d.Dispose()
	start := time.Now()
	d.FollowSystemTheme()
	if el := time.Since(start); el > 300*time.Millisecond {
		t.Fatalf("FollowSystemTheme waited %v for the portal", el)
	}
	if DisplayThemeDark {
		t.Fatal("dark before the portal answered")
	}
	for deadline := time.Now().Add(10 * time.Second); !DisplayThemeDark || accentProvider == 0; {
		if time.Now().After(deadline) {
			t.Fatalf("portal values not applied: dark %v, accent %v", DisplayThemeDark, accentProvider != 0)
		}
		if !d.ReadAndDispatch() {
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Logf("applied %v after FollowSystemTheme", time.Since(start))
}
