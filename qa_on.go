//go:build gowtqa

package gowt

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

// qaStart serves the QA driver (package qa) when GOWT_QA_SOCKET is set.
func qaStart(a *App) {
	err := qa.Start(qa.Host{
		Display: a.display,
		Sync:    a.Sync,
		Flush:   a.Flush,
		Pending: func() string {
			if a.Pending() == 0 {
				return ""
			}
			return fmt.Sprintf("%d rx effects and posts, %d layouts", rx.Pending(), len(a.relayouts))
		},
		Quit:       a.Quit,
		ShotHandle: shotHandle,
	})
	if err != nil {
		log.Print(err)
	}
}

// qaQuiet is true when windows must open without taking the focus. Linux keeps Open: the stands
// there have no user focus to lose.
func qaQuiet() bool { return os.Getenv(qa.EnvSocket) != "" && runtime.GOOS != "linux" }
