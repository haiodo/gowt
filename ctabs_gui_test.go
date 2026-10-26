package gowt

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/swt"
)

// probeRenderer is a Go subclass of the SWT renderer that counts the selected tabs it is asked to draw.
type probeRenderer struct {
	*swt.CTabFolderRenderer
	selected int
}

func (p *probeRenderer) Draw_(part, state int32, b *swt.Rectangle, gc *swt.GC) {
	if part >= 0 && state&swt.SELECTED != 0 {
		p.selected++
	}
	p.CTabFolderRenderer.Draw(part, state, b, gc)
}

// A renderer subclass installed with SetRenderer is called by the folder when it paints.
func TestCTabsRendererOverride(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var tabs *CTabsView
	var probe *probeRenderer
	var errs []error
	var height int32
	var ring [3]bool
	onMain(func() {
		err := Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("ctabs")
			w.SetSize(400, 200)
			w.Add(CTabs(Ref(&tabs), Grow(), Tab("one"), Tab("two", Closable())))
			w.Show()
			steps(app,
				func() {
					f := tabs.Unwrap()
					height = f.GetTabHeight()
					r := newTabRenderer(f)
					ring[0] = r.keys
					(&tabKeys{r}).HandleEvent(&swt.Event{Type: swt.KeyDown, Character: '\t'})
					ring[1] = r.keys
					(&tabKeys{r}).HandleEvent(&swt.Event{Type: swt.MouseDown})
					ring[2] = r.keys
					probe = &probeRenderer{}
					probe.CTabFolderRenderer = swt.NewCTabFolderRenderer(f)
					probe.SetImpl_(probe)
					f.SetRenderer(probe)
					f.Redraw()
				},
				func() { w.Close() })
		})
		if err != nil {
			errs = append(errs, err)
		}
	})
	for _, e := range errs {
		t.Errorf("%v", e)
	}
	if ring != [3]bool{false, true, false} {
		t.Errorf("focus ring at start, after Tab, after a click: %v, want [false true false]", ring)
	}
	if probe == nil || probe.selected == 0 {
		t.Errorf("the subclass drew no selected tab")
	}
	if want := int32(Spacing.ControlHeight); height != want {
		t.Errorf("tab height %d, want the ControlHeight token %d", height, want)
	}
}
