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

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

func init() {
	qaDialog = func(c dialogCall) (json.RawMessage, bool, bool) {
		return qa.Ask(qa.Dialog{Kind: c.Kind, Title: c.Title, Message: c.Message, Buttons: c.Buttons, Dir: c.Dir,
			Name: c.Name, Save: c.Save, Multi: c.Multi, Filters: c.Filters})
	}
	qaDialogFailed = qa.Failed
	qaPopover = func(p *Popover) {
		p.shell.SetDataKeyValue(qaview.PopoverKey, popoverQA{p})
		remove := qaview.AddPopover(p.shell)
		p.shell.AddDisposeListener(disposeFunc(remove))
	}
}

// popoverQA is what the QA driver sees of a Popover.
type popoverQA struct{ p *Popover }

func (q popoverQA) OnScreen() bool    { return q.p.shown && q.p.native.ready(q.p) }
func (q popoverQA) Owner() *swt.Shell { return q.p.owner }
func (q popoverQA) Close()            { q.p.Close() }
func (q popoverQA) Opening() bool     { return q.p.shown && !q.p.native.ready(q.p) }

func (q popoverQA) Escape() {
	if q.p.onEscape == nil || !q.p.onEscape() {
		q.p.Close()
	}
}

func (q popoverQA) Bounds() (x, y, w, h int) {
	b := q.p.native.bounds(q.p)
	return int(b.X), int(b.Y), int(b.Width), int(b.Height)
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
			if qaview.PopoverOpening() {
				parts = append(parts, "a popover opening")
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
