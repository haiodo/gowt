package gowt

import (
	"strings"
	"testing"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

func TestDisclosure(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		open := rx.New(true)
		var changed []bool
		var add *ButtonView
		d := put[*DisclosureView](w, Disclosure("Layout", GrowX(), Bind(open), OnChange(func(on bool) { changed = append(changed, on) }),
			Trailing(Button("+", Borderless(), Ref(&add))),
			Label("body")))
		c := put[*DisclosureView](w, Disclosure("Fill", Collapsed(), Label("hidden")))
		if len(*errs) != 0 {
			t.Fatalf("valid use reported %q", *errs)
		}
		if add == nil || add.Unwrap().GetParent() != d.trailing {
			t.Error("Trailing did not put the button in the header")
		}
		if !d.Expanded() || c.Expanded() || c.body.GetVisible() {
			t.Errorf("start: expanded %v, collapsed one %v (body visible %v)", d.Expanded(), c.Expanded(), c.body.GetVisible())
		}
		tall := preferred(d.control()).Y
		notify(d.chevron, swt.Selection) // a click on the chevron
		if d.Expanded() || open.Get() || len(changed) != 1 || changed[0] {
			t.Errorf("click: expanded %v, state %v, OnChange %v", d.Expanded(), open.Get(), changed)
		}
		if short := preferred(d.control()).Y; short >= tall {
			t.Errorf("folded height %d, expanded %d", short, tall)
		}
		open.Set(true)
		rx.Flush()
		if !d.Expanded() || !d.body.GetVisible() || len(changed) != 1 {
			t.Errorf("state true: expanded %v, body visible %v, OnChange %v", d.Expanded(), d.body.GetVisible(), changed)
		}
		notify(d.title, swt.MouseUp) // a click on the title
		if d.Expanded() {
			t.Error("a click on the title did not fold the body")
		}

		put[*LabelView](w, Label("l", Collapsed()))
		put[*LabelView](w, Label("l", Trailing(Label("x"))))
		want := []string{
			"gowt: Collapsed applies to Disclosure, not to Label",
			"gowt: Trailing applies to Disclosure, not to Label",
		}
		if got := strings.Join(*errs, "\n"); got != strings.Join(want, "\n") {
			t.Errorf("errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
		}
	})
}
