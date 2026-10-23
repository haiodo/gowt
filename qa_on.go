//go:build gowtqa

package gowt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
)

func init() {
	qaDialog = func(c dialogCall) (json.RawMessage, bool, bool) {
		return qa.Ask(qa.Dialog{Kind: c.Kind, Title: c.Title, Message: c.Message, Buttons: c.Buttons, Dir: c.Dir,
			Name: c.Name, Save: c.Save, Multi: c.Multi, Filters: c.Filters})
	}
	qaDialogFailed = qa.Failed
}

// qaStart serves the QA driver (package qa) when GOWT_QA_SOCKET is set.
func qaStart(a *App) {
	err := qa.Start(qa.Host{
		Display: a.display,
		Sync:    a.Sync,
		Flush:   a.Flush,
		Pending: func() string {
			var parts []string
			if n := rx.Pending(); n+len(a.relayouts) > 0 {
				parts = append(parts, fmt.Sprintf("%d rx effects and posts, %d layouts", n, len(a.relayouts)))
			}
			if w := a.waitingOn(); w != "" {
				parts = append(parts, w)
			}
			return strings.Join(parts, ", ")
		},
		Jobs: func() []qa.Job {
			var out []qa.Job
			for _, j := range a.Jobs() {
				out = append(out, qa.Job{Name: j.Name, Background: j.Background, Track: j.Track,
					ElapsedMS: time.Since(j.Started).Milliseconds(), Done: j.Done, Total: j.Total, Text: j.Text})
			}
			return out
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
