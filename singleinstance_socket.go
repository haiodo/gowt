package gowt

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Unix sockets work on Windows 10 1803 and later. Paths travel NUL-separated; the sender sends
// absolute paths, since its working directory is not the receiver's.
func socketPath(id string) string { return filepath.Join(os.TempDir(), "gowt-"+id+".sock") }

func singleInstanceSocket(id string, args []string) (bool, error) {
	p := socketPath(id)
	var err error
	for try := 0; try < 2; try++ {
		var l net.Listener
		if l, err = net.Listen("unix", p); err == nil {
			go serveSocket(l)
			return true, nil
		}
		c, derr := net.DialTimeout("unix", p, 2*time.Second)
		if derr == nil {
			defer c.Close()
			c.SetDeadline(time.Now().Add(2 * time.Second))
			_, werr := io.WriteString(c, strings.Join(args, "\x00"))
			return werr != nil, werr
		}
		os.Remove(p) // nobody answers: a leftover of a crashed instance
	}
	return true, err
}

func serveSocket(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			c.SetDeadline(time.Now().Add(5 * time.Second))
			b, _ := io.ReadAll(io.LimitReader(c, 1<<20))
			var paths []string
			if len(b) > 0 {
				paths = strings.Split(string(b), "\x00")
			}
			inboxPush(paths)
		}()
	}
}
