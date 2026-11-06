package look_test

import (
	"fmt"
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"math"
	"os"
	"path/filepath"
	"structs"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// Small Number fields on the inspector glass, light and dark: the field editor of the focused one
// starts after its prefix, and each scrubbed field has its handle. GOWT_PIXELS_DIR keeps the PNGs.
func TestInspectorFields(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, mode := range []gowt.Appearance{gowt.AppearanceLight, gowt.AppearanceDark} {
		name := map[gowt.Appearance]string{gowt.AppearanceLight: "light", gowt.AppearanceDark: "dark"}[mode]
		t.Run(name, func(t *testing.T) {
			r := captureInspectorFields(t, mode, filepath.Join(dir, "inspector-fields-"+name+".png"))
			if !r.editing {
				t.Fatal("the Focused field is not being edited")
			}
			if r.valueStart < 10 || math.Abs(r.editorX-r.valueStart) > 1 {
				t.Errorf("field editor at x %.1f, value drawn from %.1f: the editor must start after the prefix", r.editorX, r.valueStart)
			}
			if r.handles != 4 {
				t.Errorf("%d of 4 scrubbed fields have a handle", r.handles)
			}
		})
	}
}

type fieldReport struct {
	editorX, valueStart float64
	editing             bool
	handles             int
}

// captureInspectorFields shows small fields in an inspector, X focused, and saves the window to path.
func captureInspectorFields(t *testing.T, mode gowt.Appearance, path string) (r fieldReport) {
	t.Helper()
	var errs []error
	var err, shotErr error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("inspector fields")
			var content, insp *gowt.Panel
			n := make([]*gowt.NumberView, 4)
			field := func(i int, opts ...gowt.Option) gowt.Option {
				return gowt.Number(-1000, 1000, append([]gowt.Option{gowt.Small(), gowt.GrowX(), gowt.Ref(&n[i])}, opts...)...)
			}
			w.Add(gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(16), gowt.Gap(10),
					gowt.Label("Layout", gowt.Typeface(gowt.Type.BodyStrong)),
					gowt.Grid(2, gowt.GrowX(),
						field(0, gowt.Prefix("X"), gowt.FieldFill(gowt.RGB{R: 240, G: 240, B: 240}), gowt.Value(260), gowt.Focused()),
						field(1, gowt.Prefix("Y"), gowt.Value(-40)),
						field(2, gowt.Prefix("W"), gowt.Suffix("px"), gowt.Value(120)),
						gowt.Number(0, 100, gowt.Prefix("Gap"), gowt.Small(), gowt.GrowX(), gowt.Value(8))),
					field(3, gowt.PrefixImage(icons.Get(app, "star", icons.Size(12))), gowt.Suffix("%"), gowt.Step(0.5), gowt.Value(87.5)),
					gowt.Text(gowt.Prefix("Name"), gowt.Small(), gowt.Border(), gowt.Value("Server"), gowt.GrowX()),
					gowt.Number(0, 100, gowt.Prefix("R"), gowt.Value(12), gowt.GrowX(), gowt.Disabled())))
			look.SetInspector(w, content, insp, 260)
			w.SetSize(700, 420)
			w.Show()
			app.After(800*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				v := n[0].Unwrap().View
				r.editorX, r.editing = cocoa.FieldEditorX(v)
				r.valueStart = cocoa.FieldValueStart(v)
				for _, f := range n {
					if cocoa.FieldAffixHandle(f.Unwrap().View) != 0 {
						r.handles++
					}
				}
				shotErr = shot.WindowPNG(shotHandle(w), path)
			})
		})
	})
	for _, e := range append(errs, err, shotErr) {
		if e != nil {
			t.Fatal(e)
		}
	}
	return r
}

func TestFieldEditorMouseTracking(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, tc := range []struct{ prefix, value string }{{"", ""}, {"", "Root"}, {"X", ""}, {"X", "Root"}} {
		t.Run(tc.prefix+"_"+tc.value, func(t *testing.T) {
			onMain(func() {
				if err := gowt.Run(func(app *gowt.App) {
					app.SetAppearance(gowt.AppearanceLight)
					w := app.Window("field mouse tracking")
					var field *gowt.TextView
					w.Add(gowt.Text(gowt.Ref(&field), gowt.Value(tc.value), gowt.Prefix(tc.prefix), gowt.Small(), gowt.Size(48, 22), gowt.Inset(6, 0), gowt.Radius(4), gowt.FieldFill(gowt.RGB{R: 240, G: 240, B: 240}), gowt.FieldStroke(gowt.RGB{R: 100, G: 100, B: 100})))
					w.Show()
					w.Unwrap().ForceActive()
					app.After(100*time.Millisecond, func() {
						defer w.Close()
						defer app.SetAppearance(gowt.AppearanceSystem)
						view := field.Unwrap().View
						win := view.Window()
						addr, _ := purego.Dlsym(purego.RTLD_DEFAULT, "objc_msgSend")
						var mouseEvent func(int64, int64, uintptr, cocoa.NSPoint, uintptr, float64, int64, uintptr, int64, int64, float64) int64
						var caret func(int64, int64, cocoa.NSRange, uintptr) cocoa.NSRect
						var convert func(int64, int64, cocoa.NSRect) cocoa.NSRect
						purego.RegisterFunc(&mouseEvent, addr)
						purego.RegisterFunc(&caret, addr)
						purego.RegisterFunc(&convert, addr)
						point := view.ConvertPoint_toView_(cocoa.NSPoint{X: 24, Y: 11}, nil)
						event := func(kind uintptr) *cocoa.NSEvent {
							return cocoa.NewNSEventOverload1(mouseEvent(cocoa.OSObjc_getClass("NSEvent"), cocoa.OSSel_registerName("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"), kind, point, 0, 0, win.WindowNumber(), 0, 1, 1, 1))
						}
						readCaret := func() cocoa.NSRect {
							ed := cocoa.NewNSControlOverload1(view.Id).CurrentEditor()
							if ed == nil {
								t.Error("no field editor during mouse tracking")
								return cocoa.NSRect{}
							}
							// Mouse-up can change selection; compare the same insertion index in both phases.
							// Finish native text layout so document resizing cannot hide a tracking-time shift.
							textView := cocoa.NewNSTextViewOverload1(ed.Id)
							textView.LayoutManager().UsedRectForTextContainer(textView.TextContainer())
							r := caret(ed.Id, cocoa.OSSel_registerName("firstRectForCharacterRange:actualRange:"), cocoa.NSRange{}, 0)
							r = convert(win.Id, cocoa.OSSel_registerName("convertRectFromScreen:"), r)
							return view.ConvertRect_fromView_(r, nil)
						}
						inkRows := func() [4]int {
							rep := view.BitmapImageRepForCachingDisplayInRect(view.Bounds())
							view.CacheDisplayInRect(view.Bounds(), rep)
							rows := [4]int{int(rep.PixelsHigh()), -1, int(rep.PixelsWide()), -1}
							for y := int64(0); y < rep.PixelsHigh(); y++ {
								for x := int64(0); x < rep.PixelsWide(); x++ {
									c := rep.ColorAtX(x, y)
									if c.AlphaComponent() > 0.5 && c.RedComponent() < 0.3 && c.GreenComponent() < 0.3 && c.BlueComponent() < 0.3 {
										rows[0] = min(rows[0], int(y))
										rows[1] = max(rows[1], int(y))
										rows[2] = min(rows[2], int(x))
										rows[3] = max(rows[3], int(x))
									}
								}
							}
							return rows
						}
						value := tc.value
						win.MakeFirstResponder(nil)
						idleRows := inkRows()
						var heldRows [4]int
						var held cocoa.NSRect
						called := false
						timerClass, err := objc.RegisterClass(fmt.Sprintf("GowtTrackingTimer%d", time.Now().UnixNano()), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{Cmd: objc.RegisterName("fire:"), Fn: func(self objc.ID, cmd objc.SEL, timer objc.ID) {
							defer cocoa.NSApplicationSharedApplication().PostEvent(event(2), false)
							if called {
								return
							}
							called = true
							if err := shot.WindowPNG(shotHandle(w), filepath.Join(dir, "held-"+tc.prefix+"-"+value+".png")); err != nil {
								t.Error(err)
							}
							held = readCaret()
							heldRows = inkRows()
						}}})
						if err != nil {
							t.Error(err)
							return
						}
						target := objc.ID(timerClass).Send(objc.RegisterName("new"))
						timer := cocoa.NSTimerScheduledTimerWithTimeInterval(0.02, cocoa.NewidOverload1(int64(target)), cocoa.OSSel_registerName("fire:"), nil, true)
						cocoa.NSRunLoopCurrentRunLoop().AddTimer(timer, cocoa.NSStringStringWith("NSEventTrackingRunLoopMode"))
						win.SendEvent(event(1))
						timer.Invalidate()
						target.Send(objc.RegisterName("release"))
						released := readCaret()
						if value != "" && (idleRows[1] < idleRows[0] || heldRows[1] < heldRows[0]) {
							t.Errorf("missing glyph pixels: idle %v held %v", idleRows, heldRows)
						}
						if value == "" && (held.Height <= 0 || math.Abs(held.Y+held.Height/2-view.Bounds().Height/2) > 0.5) {
							t.Errorf("empty caret is not centered: %+v", held)
						}
						if value != "" && heldRows[2] != idleRows[2] {
							t.Errorf("text moved horizontally: idle %v, editing %v", idleRows, heldRows)
						}
						if value != "" && (math.Abs(float64(heldRows[0]-idleRows[0])) > 1 || math.Abs(float64(heldRows[1]-idleRows[1])) > 1) {
							t.Errorf("prefix %q: idle ink rows %v, editing %v", tc.prefix, idleRows, heldRows)
						}
						t.Logf("prefix %q value %q: idle rows %v held rows %v, held %+v released %+v", tc.prefix, value, idleRows, heldRows, held, released)
						if err := shot.WindowPNG(shotHandle(w), filepath.Join(dir, "released-"+tc.prefix+"-"+value+".png")); err != nil {
							t.Error(err)
						}
						if !called || held.X < 6 || math.Abs(held.X-released.X) > 0.5 || math.Abs(held.Y-released.Y) > 0.5 || math.Abs(held.Height-released.Height) > 0.5 {
							t.Errorf("prefix %q value %q: tracking=%v held %+v, released %+v", tc.prefix, value, called, held, released)
						}
					})
				}); err != nil {
					t.Error(err)
				}
			})
		})
	}
}

// A click on a field and a drag out of it: while the mouse is down AppKit autoscrolls the editor's
// clip to the pointer, and a one-line clip that scrolled draws the value off the prefix's line, raised
// and clipped at the top. The value must sit where it did before the click once the mouse is released.
func TestPrefixFieldDragOutOfField(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	onMain(func() {
		if err := gowt.Run(func(app *gowt.App) {
			app.SetAppearance(gowt.AppearanceLight)
			w := app.Window("field drag")
			var field, other *gowt.TextView
			row := func(ref **gowt.TextView, prefix, value string) gowt.Option {
				return gowt.HBox(gowt.Text(gowt.Ref(ref), gowt.Value(value), gowt.Prefix(prefix), gowt.Small(), gowt.Size(48, 22), gowt.Inset(6, 0), gowt.Radius(4), gowt.FieldFill(gowt.RGB{R: 240, G: 240, B: 240}), gowt.FieldStroke(gowt.RGB{R: 100, G: 100, B: 100})))
			}
			w.Add(gowt.VBox(row(&other, "Y", "-140"), row(&field, "X", "24")))
			w.SetSize(300, 140)
			w.Show()
			w.Unwrap().ForceActive()
			app.After(150*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem)
				view := field.Unwrap().View
				win := view.Window()
				addr, _ := purego.Dlsym(purego.RTLD_DEFAULT, "objc_msgSend")
				var mouseEvent func(int64, int64, uintptr, cocoa.NSPoint, uintptr, float64, int64, uintptr, int64, int64, float64) int64
				var autoscroll func(self, sel int64, ev int64) bool
				purego.RegisterFunc(&mouseEvent, addr)
				purego.RegisterFunc(&autoscroll, addr)
				event := func(kind uintptr, x, y float64) *cocoa.NSEvent {
					p := view.ConvertPoint_toView_(cocoa.NSPoint{X: x, Y: y}, nil)
					return cocoa.NewNSEventOverload1(mouseEvent(cocoa.OSObjc_getClass("NSEvent"), cocoa.OSSel_registerName("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"), kind, p, 0, 0, win.WindowNumber(), 0, 1, 1, 1))
				}
				// The dark pixels of the value: left, right, top, bottom in pixels of the 2x shot.
				ink := func(name string) (box [4]int) {
					rep := view.BitmapImageRepForCachingDisplayInRect(view.Bounds())
					view.CacheDisplayInRect(view.Bounds(), rep)
					box = [4]int{int(rep.PixelsWide()), -1, int(rep.PixelsHigh()), -1}
					for y := int64(0); y < rep.PixelsHigh(); y++ {
						for x := int64(0); x < rep.PixelsWide(); x++ {
							c := rep.ColorAtX(x, y)
							if c.AlphaComponent() > 0.5 && c.RedComponent() < 0.3 && c.GreenComponent() < 0.3 && c.BlueComponent() < 0.3 {
								box[0], box[1] = min(box[0], int(x)), max(box[1], int(x))
								box[2], box[3] = min(box[2], int(y)), max(box[3], int(y))
							}
						}
					}
					if err := shot.WindowPNG(shotHandle(w), filepath.Join(dir, "drag-"+name+".png")); err != nil {
						t.Error(err)
					}
					return box
				}
				other.Focus()
				app.Flush()
				win.MakeFirstResponder(nil)
				idle := ink("idle")
				editing := false
				ticks := 0
				timerClass, err := objc.RegisterClass(fmt.Sprintf("GowtDragTimer%d", time.Now().UnixNano()), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{Cmd: objc.RegisterName("fire:"), Fn: func(self objc.ID, cmd objc.SEL, timer objc.ID) {
					ticks++
					ed := cocoa.NewNSControlOverload1(view.Id).CurrentEditor()
					if ticks == 1 && ed != nil {
						editing = true
						// The pointer left the field up and to the left; this is the scroll AppKit runs.
						autoscroll(ed.Superview().Id, cocoa.OSSel_registerName("autoscroll:"), event(6, -9, 30).Id)
					}
					if ticks == 2 {
						cocoa.NSApplicationSharedApplication().PostEvent(event(2, -9, 30), false)
					}
				}}})
				if err != nil {
					t.Error(err)
					return
				}
				target := objc.ID(timerClass).Send(objc.RegisterName("new"))
				timer := cocoa.NSTimerScheduledTimerWithTimeInterval(0.02, cocoa.NewidOverload1(int64(target)), cocoa.OSSel_registerName("fire:"), nil, true)
				cocoa.NSRunLoopCurrentRunLoop().AddTimer(timer, cocoa.NSStringStringWith("NSEventTrackingRunLoopMode"))
				win.SendEvent(event(1, 28, 11))
				timer.Invalidate()
				target.Send(objc.RegisterName("release"))
				released := ink("released")
				if !editing {
					t.Fatal("the click did not start editing the field")
				}
				for i := range idle {
					if d := released[i] - idle[i]; d < -1 || d > 1 {
						t.Errorf("value %v after the drag, %v before it: the text left its line", released, idle)
						break
					}
				}
				// The drag must leave the field usable: typing, Return, leaving the field and clicking
				// it again keep the value on the prefix's line. Only the vertical ink is compared - a
				// selection or a wider value moves the horizontal edges by design.
				sameLine := func(name string, box [4]int) {
					if d := box[2] - idle[2]; d < -1 || d > 1 || box[3]-idle[3] < -1 || box[3]-idle[3] > 1 {
						t.Errorf("%s: value ink %v, idle %v: the text left its line", name, box, idle)
					}
				}
				key := func(character string, keyCode uint16) {
					s := cocoa.NSStringStringWith(character)
					for _, typ := range []uint64{10, 11} {
						e := objc.ID(objc.GetClass("NSEvent")).Send(objc.RegisterName("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"),
							typ, struct {
								_    structs.HostLayout
								X, Y float64
							}{}, uint64(0), 0.0, win.WindowNumber(), objc.ID(0), objc.ID(s.Id), objc.ID(s.Id), false, keyCode)
						cocoa.NSApplicationSharedApplication().PostEvent(cocoa.NewNSEventOverload1(int64(e)), false)
					}
					for field.Unwrap().GetDisplay().ReadAndDispatch() {
					}
				}
				clickAt := func(x, y float64) {
					timerClass, err := objc.RegisterClass(fmt.Sprintf("GowtClickTimer%d", time.Now().UnixNano()), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{Cmd: objc.RegisterName("fire:"), Fn: func(self objc.ID, cmd objc.SEL, timer objc.ID) {
						cocoa.NSApplicationSharedApplication().PostEvent(event(2, x, y), false)
					}}})
					if err != nil {
						t.Error(err)
						return
					}
					target := objc.ID(timerClass).Send(objc.RegisterName("new"))
					timer := cocoa.NSTimerScheduledTimerWithTimeInterval(0.02, cocoa.NewidOverload1(int64(target)), cocoa.OSSel_registerName("fire:"), nil, false)
					cocoa.NSRunLoopCurrentRunLoop().AddTimer(timer, cocoa.NSStringStringWith("NSEventTrackingRunLoopMode"))
					win.SendEvent(event(1, x, y))
					timer.Invalidate()
					target.Send(objc.RegisterName("release"))
				}
				was := field.Unwrap().GetText()
				key("x", 7)
				if got := field.Unwrap().GetText(); got == was {
					t.Errorf("typing did not reach the field: text %q", got)
				}
				sameLine("typed", ink("typed"))
				key("\r", 36)
				sameLine("returned", ink("returned"))
				other.Focus()
				app.Flush()
				sameLine("blurred", ink("blurred"))
				clickAt(28, 11)
				if ed := cocoa.NewNSControlOverload1(view.Id).CurrentEditor(); ed == nil {
					t.Error("the click after the drag did not start editing the field")
				}
				sameLine("reclicked", ink("reclicked"))
			})
		}); err != nil {
			t.Error(err)
		}
	})
}
