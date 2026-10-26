package view

import (
	"testing"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/swt"
)

func TestChildRect(t *testing.T) {
	cam := gfx.Camera{Origin: gfx.Point{X: 100, Y: 50}, Zoom: 2}
	for _, c := range []struct {
		name string
		cam  gfx.Camera
		r    gfx.Rect
		want [4]int
	}{
		{"at the origin", cam, gfx.R(100, 50, 40, 20), [4]int{0, 0, 80, 40}},
		{"left of the origin", cam, gfx.R(90, 40, 10, 10), [4]int{-20, -20, 20, 20}},
		{"fractions grow the box", gfx.Camera{Origin: gfx.Point{X: 0.25, Y: 0}, Zoom: 1.5}, gfx.R(1, 1, 3, 3), [4]int{1, 1, 5, 5}},
		{"zoom out", gfx.Camera{Zoom: 0.5}, gfx.R(10, 10, 100, 30), [4]int{5, 5, 50, 15}},
	} {
		x, y, w, h := childRect(c.cam, c.r)
		if got := [4]int{x, y, w, h}; got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestZoomedPx(t *testing.T) {
	for _, c := range []struct{ px, zoom, want float64 }{
		{12, 1, 12}, {12, 1.5, 18}, {13.5, 0.5, 6.75}, {12, 1.01, 12}, {12, 1.03, 12.25}, {10, 0.01, 1},
	} {
		if got := zoomedPx(c.px, c.zoom); got != c.want {
			t.Errorf("zoomedPx(%v, %v) = %v, want %v", c.px, c.zoom, got, c.want)
		}
	}
}

// fakeEditor records what an editor does to its control and to the app.
type fakeEditor struct {
	e        *editor
	text     string
	finished []bool // refocus of each finish
	commits  []string
	cancels  int
}

func newFakeEditor(multiline bool) *fakeEditor {
	f := &fakeEditor{text: "abc"}
	f.e = &editor{
		text:      func() string { return f.text },
		finish:    func(refocus bool) { f.finished = append(f.finished, refocus) },
		onCommit:  func(s string) { f.commits = append(f.commits, s) },
		onCancel:  func() { f.cancels++ },
		multiline: multiline,
	}
	return f
}

func TestEditorKeys(t *testing.T) {
	enter, esc := int32(swt.CR), int32(swt.ESC)
	for _, c := range []struct {
		name      string
		multiline bool
		code      int32
		mods      g.Mods
		taken     bool
		commits   int
		cancels   int
	}{
		{"Enter in one line", false, enter, g.Mods{}, true, 1, 0},
		{"keypad Enter in one line", false, swt.KEYPAD_CR, g.Mods{}, true, 1, 0},
		{"Enter in many lines types a break", true, enter, g.Mods{}, false, 0, 0},
		{"Shift+Enter in many lines types a break", true, enter, g.Mods{Shift: true}, false, 0, 0},
		{"Cmd+Enter in many lines", true, enter, g.Mods{Cmd: true}, true, 1, 0},
		{"Ctrl+Enter in many lines", true, enter, g.Mods{Ctrl: true}, true, 1, 0},
		{"Esc in one line", false, esc, g.Mods{}, true, 0, 1},
		{"Esc in many lines", true, esc, g.Mods{}, true, 0, 1},
		{"a letter", false, 'a', g.Mods{}, false, 0, 0},
	} {
		f := newFakeEditor(c.multiline)
		if got := f.e.key(c.code, c.mods); got != c.taken {
			t.Errorf("%s: taken %v, want %v", c.name, got, c.taken)
		}
		if len(f.commits) != c.commits || f.cancels != c.cancels {
			t.Errorf("%s: commits %v cancels %d, want %d and %d", c.name, f.commits, f.cancels, c.commits, c.cancels)
		}
		if c.taken && (len(f.finished) != 1 || !f.finished[0]) {
			t.Errorf("%s: finish %v, want one call that refocuses", c.name, f.finished)
		}
	}
}

func TestEditorEndsOnce(t *testing.T) {
	f := newFakeEditor(false)
	f.e.end(true, false) // the focus left
	f.e.end(false, true) // the Esc that follows, or the focus loss of disposing: ignored
	f.e.end(true, false)
	if len(f.commits) != 1 || f.commits[0] != "abc" || f.cancels != 0 || len(f.finished) != 1 || f.finished[0] {
		t.Errorf("commits %v cancels %d finished %v", f.commits, f.cancels, f.finished)
	}
}

func TestEditorReadsTextBeforeFinish(t *testing.T) {
	f := newFakeEditor(false)
	f.e.finish = func(bool) { f.text = "" } // the control is emptied or gone
	f.e.end(true, false)
	if len(f.commits) != 1 || f.commits[0] != "abc" {
		t.Errorf("commits %v, want the text typed", f.commits)
	}
}

func TestEditorNilCallbacks(t *testing.T) {
	for _, commit := range []bool{true, false} {
		f := newFakeEditor(false)
		f.e.onCommit, f.e.onCancel = nil, nil
		f.e.end(commit, false)
		if len(f.finished) != 1 {
			t.Errorf("commit=%v: finished %v", commit, f.finished)
		}
	}
}
