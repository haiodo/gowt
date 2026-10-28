package gowt

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSocketForward(t *testing.T) {
	id := fmt.Sprintf("org.gowt.Sock%d", os.Getpid())
	t.Cleanup(func() { os.Remove(socketPath(id)) })
	if first, err := singleInstanceSocket(id, nil); !first || err != nil {
		t.Fatalf("first = %v, %v", first, err)
	}
	inbox.Lock()
	inbox.backlog = nil
	inbox.Unlock()
	if first, err := singleInstanceSocket(id, []string{"/x/a", "tenniarb://open?id=1", "/x/b"}); first || err != nil {
		t.Fatalf("second = %v, %v; want forwarded", first, err)
	}
	// The receiver reads on its own goroutine; 2 s was not enough once on a hosted Windows runner.
	var got []string
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		inbox.Lock()
		got = inbox.backlog
		inbox.Unlock()
		if len(got) >= 3 {
			break
		}
	}
	if !reflect.DeepEqual(got, []string{"/x/a", "tenniarb://open?id=1", "/x/b"}) {
		t.Fatalf("backlog = %q after 10 s, want the forwarded paths and URL", got)
	}
}

func TestSocketStaleFile(t *testing.T) {
	id := fmt.Sprintf("org.gowt.Stale%d", os.Getpid())
	if err := os.WriteFile(socketPath(id), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(socketPath(id)) })
	if first, err := singleInstanceSocket(id, nil); !first || err != nil {
		t.Errorf("with a leftover file: first = %v, %v", first, err)
	}
}
