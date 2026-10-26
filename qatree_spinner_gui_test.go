package gowt

import (
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/swt"
)

func TestQATreeSpinner(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	var sp *SpinnerView
	changes := 0
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			w := app.Window("qa spinner")
			w.Add(VBox(
				Spinner(0, 100, ID("sp"), Ref(&sp), OnChange(func(int) { changes++ })),
				Spinner(0, 100, ID("ro"), ReadOnly())))
			w.Show()
			s := qatree.Selector{ID: "sp"}
			check := func(want int) {
				t.Helper()
				if got := sp.Value(); got != want {
					t.Errorf("value %d, want %d", got, want)
				}
				if n := qaFind(t, app, s).Value; n != strconv.Itoa(want) {
					t.Errorf("dump value %q, want %d", n, want)
				}
				if got := sp.s.GetText(); got != strconv.Itoa(want) {
					t.Errorf("text %q, want %d", got, want)
				}
			}
			qaAct(t, app, s, "type", `{"text":"42"}`)
			check(42)
			if changes < 2 {
				t.Errorf("%d changes for two characters", changes)
			}
			qaAct(t, app, s, "key", `{"keys":"Up"}`)
			check(43)
			qaAct(t, app, s, "key", `{"keys":"PageDown"}`)
			check(33)
			qaAct(t, app, s, "type", `{"text":"500"}`)
			check(100)
			for _, bad := range []string{"abc", "-5", "1.5", ""} {
				if err := qatree.Act(app.Unwrap(), s, "type", []byte(`{"text":"`+bad+`"}`)); err == nil {
					t.Errorf("type %q was accepted", bad)
				}
			}
			if err := qatree.Act(app.Unwrap(), qatree.Selector{ID: "ro"}, "type", []byte(`{"text":"5"}`)); err == nil {
				t.Error("type into a read-only spinner was accepted")
			}
			qaAct(t, app, qatree.Selector{Class: "Shell"}, "close", "")
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAppMenuHooks(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1 to open windows")
	}
	about, prefs := 0, 0
	var gotAbout, gotPrefs bool
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.Window("hooks").Show()
			gotAbout = app.OnAbout(func() { about++ })
			gotPrefs = app.OnPreferences(func() { prefs++ })
			if m := app.Unwrap().GetSystemMenu(); m != nil {
				for _, it := range m.GetItems() {
					if id := it.GetID(); id == swt.ID_ABOUT || id == swt.ID_PREFERENCES {
						it.NotifyListeners(swt.Selection, swt.NewEvent())
					}
				}
			}
			app.Quit()
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	mac := runtime.GOOS == "darwin"
	if gotAbout != mac || gotPrefs != mac {
		t.Errorf("hooked About %v, Preferences %v, want both %v", gotAbout, gotPrefs, mac)
	}
	if mac && (about != 1 || prefs != 1) {
		t.Errorf("About ran %d times, Preferences %d, want 1 each", about, prefs)
	}
}
