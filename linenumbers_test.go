package gowt

import (
	"strings"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestLineNumbers(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		st := put[*StyledTextView](w, StyledText(LineNumbers(), Size(300, 200)))
		st.SetText(strings.Repeat("line\n", 11) + "last")
		put[*TextView](w, Text(LineNumbers()))
		if len(*errs) != 1 {
			t.Errorf("LineNumbers on a Text reported %q", *errs)
		}
		if st.nums == nil {
			t.Fatal("LineNumbers did not install the listener")
		}
		e := &swt.LineStyleEvent{LineOffset: st.t.GetOffsetAtLine(3)}
		st.nums.LineGetStyle(e)
		if e.Bullet == nil || e.BulletIndex != 3 || e.Bullet.Type != swt.STBULLET_NUMBER {
			t.Fatalf("line 3: bullet %+v index %d", e.Bullet, e.BulletIndex)
		}
		if wd := e.Bullet.Style.Metrics.Width; wd <= 0 {
			t.Errorf("gutter width %d", wd)
		}
		st.SetLineNumbers(false)
		if st.nums != nil {
			t.Error("SetLineNumbers(false) kept the listener")
		}
	})
}
