package dbus

import (
	"bufio"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func startBus(t *testing.T) {
	t.Helper()
	path, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is not installed")
	}
	cmd := exec.Command(path, "--session", "--nofork", "--print-address=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(addr))
}

func TestPathOf(t *testing.T) {
	if got := PathOf("org.example.My-App"); got != "/org/example/My_App" {
		t.Errorf("PathOf = %q", got)
	}
}

func TestNoBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/bus")
	if _, err := Connect(); err == nil {
		t.Error("Connect to a missing bus succeeded")
	}
}

func TestOpenReachesOwner(t *testing.T) {
	startBus(t)
	const name = "org.gowt.DbusTest"
	owner, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if ok, err := owner.RequestName(name); !ok || err != nil {
		t.Fatalf("RequestName = %v, %v", ok, err)
	}
	got := make(chan []string, 4)
	go owner.Serve(func(u []string) { got <- u })

	other, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if ok, err := other.RequestName(name); ok || err != nil {
		t.Fatalf("second RequestName = %v, %v; want false", ok, err)
	}
	uris := []string{"file:///tmp/a%20b.txt", "file:///tmp/ü.txt"}
	if err := other.Open(name, uris); err != nil {
		t.Fatal(err)
	}
	if err := other.Open(name, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]string{uris, nil} {
		select {
		case u := <-got:
			if !reflect.DeepEqual(u, want) {
				t.Errorf("owner got %q, want %q", u, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("owner got nothing")
		}
	}
	if err := other.Open("org.gowt.Nobody", uris); err == nil {
		t.Error("Open on a name nobody owns succeeded")
	}
}
