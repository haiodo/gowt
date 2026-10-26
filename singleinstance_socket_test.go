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
	if first, err := singleInstanceSocket(id, []string{"/x/a", "/x/b"}); first || err != nil {
		t.Fatalf("second = %v, %v; want forwarded", first, err)
	}
	for i := 0; i < 100; i++ {
		inbox.Lock()
		got := inbox.backlog
		inbox.Unlock()
		if len(got) == 2 {
			if !reflect.DeepEqual(got, []string{"/x/a", "/x/b"}) {
				t.Fatalf("backlog = %v", got)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("nothing arrived")
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
