package gowt

import (
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// TestPopoverDrawGUI renders the shown popover content: its field and button must have drawn.
func TestPopoverDrawGUI(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			var btn *ButtonView
			w := app.Window("popover", Button("Anchor", Ref(&btn)))
			p := w.Popover(Gap(4))
			p.Add(Text(Focused(), Value("hello field")), Button("Press"))
			w.Show()
			app.After(300*time.Millisecond, func() { p.ShowAt(btn, PopoverBelow) })
			app.After(1500*time.Millisecond, func() {
				defer w.Close()
				if !p.Shown() {
					t.Error("not shown")
					return
				}
				var dump func(c *swt.Control, ind string)
				dump = func(c *swt.Control, ind string) {
					t.Logf("%s%s bounds %v visible %v viewHidden %v frame %+v", ind, c.GetName(), c.GetBounds(), c.GetVisible(), c.View.IsHidden(), c.View.Frame())
					if comp, ok := c.Impl().(interface{ GetChildren() []*swt.Control }); ok {
						for _, ch := range comp.GetChildren() {
							dump(ch, ind+"  ")
						}
					}
				}
				dump(&p.shell.Control, "")
				img := cocoa.RenderView(p.shell.View)
				if img == nil {
					t.Fatal("the content did not render")
				}
				if path := os.Getenv("GOWT_POPOVER_PNG"); path != "" {
					if f, e := os.Create(path); e == nil {
						png.Encode(f, img)
						f.Close()
					}
				}
				b := img.Bounds()
				colors := map[[4]uint8]int{}
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						c := img.NRGBAAt(x, y)
						colors[[4]uint8{c.R, c.G, c.B, c.A}]++
					}
				}
				t.Logf("rendered %dx%d, %d distinct colors", b.Dx(), b.Dy(), len(colors))
				// A field with text and a button have borders, text and fills; an empty body is one color.
				if len(colors) < 10 {
					t.Errorf("the popover content drew %d distinct colors, want its field and button", len(colors))
				}
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
