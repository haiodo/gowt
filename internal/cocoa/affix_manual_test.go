//go:build darwin

package cocoa

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
)

func TestPlaceFieldEditorWithoutAffixes(t *testing.T) {
	if os.Getenv("GOWT_COCOA_EDITOR_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPlaceFieldEditorWithoutAffixes$")
		cmd.Env = append(os.Environ(), "GOWT_COCOA_EDITOR_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return
	}
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		f := affixField(12, "120", false)
		defer msg(f, "release")
		view := NewNSViewOverload1(int64(f))
		SetFieldFill(view, 6)
		ed := msg(msg(class("NSView"), "alloc"), "init")
		defer msg(ed, "release")
		msg(f, "addSubview:", ed)
		fieldClass := allocClass(class("NSTextField"), "GowtEditorTestField", 0)
		addMethod(fieldClass, sel("currentEditor"), purego.NewCallback(func(self, cmd uintptr) uintptr { return ed }), "@@:")
		registerClass(fieldClass)
		originalClass := OSObject_setClass(int64(f), int64(fieldClass))
		defer OSObject_setClass(int64(f), originalClass)
		if affixBounds != nil {
			t.Error("affix functions initialized before placing the editor")
			return
		}
		PlaceFieldEditor(view)
		got := affixBounds(ed, sel("frame"))
		want := affixDrawingRect(msg(f, "cell"), affixBounds(f, sel("bounds")))
		if got != want || got.Y <= 0 || got.Height >= 22 {
			t.Errorf("editor frame %+v, want centred text rect %+v", got, want)
		}
	})
}

func TestFieldEditorFocusLifecycle(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		var point func(self, sel uintptr) NSPoint
		var glyphPoint func(self, sel, index uintptr) NSPoint
		var floating func(self, sel uintptr) float64
		purego.RegisterFunc(&point, objcMsgSend)
		purego.RegisterFunc(&glyphPoint, objcMsgSend)
		purego.RegisterFunc(&floating, objcMsgSend)
		for _, prefix := range []string{"", "X"} {
			w := NewNSWindowOverload1(int64(msg(class("NSWindow"), "alloc"))).InitWithContentRect(NSRect{Width: 200, Height: 50}, 0, 2, false)
			f := affixField(11, "1", false)
			msgRectOnly(f, sel("setFrame:"), NSRect{Width: 48, Height: 22})
			view := NewNSViewOverload1(int64(f))
			SetFieldAffix(view, FieldAffix{Inset: 6, Prefix: prefix})
			msg(msg(uintptr(w.Id), "contentView"), "addSubview:", f)
			msg(f, "selectText:", 0)
			ed := msg(f, "currentEditor")
			if ed == 0 {
				t.Fatalf("prefix %q: no editor after selectText in hidden window", prefix)
			}
			clip := msg(ed, "superview")
			cell := msg(f, "cell")
			want := affixDrawingRect(cell, affixBounds(f, sel("bounds")))
			lead, trail := affixes[f].widths(cell)
			want.X += lead
			want.Width -= lead + trail
			for _, phase := range []string{"native selection", "explicit placement"} {
				if phase == "explicit placement" {
					PlaceFieldEditor(view)
				}
				got := affixBounds(clip, sel("frame"))
				if math.Abs(got.X-want.X) > 0.5 || math.Abs(got.Y-want.Y) > 0.5 || math.Abs(got.Width-want.Width) > 0.5 {
					t.Errorf("prefix %q, %s: editor clip %+v, want %+v", prefix, phase, got, want)
				}
				if font := floating(msg(ed, "font"), sel("pointSize")); font != 11 {
					t.Errorf("prefix %q, %s: editor font %.1f pt, want 11 pt", prefix, phase, font)
				}
				origin := point(ed, sel("textContainerOrigin"))
				glyph := glyphPoint(msg(ed, "layoutManager"), sel("locationForGlyphAtIndex:"), 0)
				glyph.X += origin.X
				glyph.Y += origin.Y
				glyph = affixConvertPoint(f, sel("convertPoint:fromView:"), glyph, ed)
				if math.Abs(glyph.X-(want.X+2)) > 0.5 {
					t.Errorf("prefix %q, %s: glyph x %.1f, want %.1f", prefix, phase, glyph.X, want.X+2)
				}
				baseline := want.Y + floating(msg(cell, "font"), sel("ascender"))
				if math.Abs(glyph.Y-baseline) > 1 {
					t.Errorf("prefix %q, %s: glyph baseline %.1f, want %.1f", prefix, phase, glyph.Y, baseline)
				}
			}
			msg(ed, "setString:", nsString("-141.0"))
			msg(ed, "selectAll:", 0)
			if got := affixBounds(clip, sel("frame")); math.Abs(got.X-want.X) > 0.5 {
				t.Errorf("prefix %q: editor clip after long selection %+v, want x %.1f", prefix, got, want.X)
			}
			ReleaseFieldAffix(view)
			msg(f, "release")
			w.Release()
		}
	})
}

func TestFieldEditorEmptyCaret(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		var caretRect func(self, sel uintptr, r NSRange, actual uintptr) NSRect
		var edit func(self, sel uintptr, r NSRect, view, editor, delegate, event uintptr)
		purego.RegisterFunc(&caretRect, objcMsgSend)
		purego.RegisterFunc(&edit, objcMsgSend)
		for _, prefix := range []string{"", "X"} {
			w := NewNSWindowOverload1(int64(msg(class("NSWindow"), "alloc"))).InitWithContentRect(NSRect{Width: 200, Height: 50}, 0, 2, false)
			f := affixField(11, "", false)
			msgRectOnly(f, sel("setFrame:"), NSRect{Width: 48, Height: 22})
			view := NewNSViewOverload1(int64(f))
			SetFieldAffix(view, FieldAffix{Inset: 6, Prefix: prefix})
			msg(msg(uintptr(w.Id), "contentView"), "addSubview:", f)
			msg(f, "selectText:", 0)
			ed := msg(f, "currentEditor")
			if ed == 0 {
				fatalf(t, "no editor in hidden window")
			}
			for phase, value := range []string{"", "1", ""} {
				if phase > 0 {
					msg(ed, "setString:", nsString(value))
					NewNSTextOverload1(int64(ed)).SetSelectedRange(NSRange{Location: int64(len(value))})
				}
				r := caretRect(ed, sel("firstRectForCharacterRange:actualRange:"), NSRange{Location: int64(len(value))}, 0)
				r = affixConvertRect(uintptr(w.Id), sel("convertRectFromScreen:"), r, 0)
				r = affixConvertRect(f, sel("convertRect:fromView:"), r, 0)
				want := 8.0
				if prefix != "" {
					want += math.Ceil(affixString(msg(f, "cell"), prefix).Size().Width) + affixGap
				}
				if value != "" {
					want += affixString(msg(f, "cell"), value).Size().Width
				}
				if math.Abs(r.X-want) > 0.5 {
					t.Errorf("prefix %q, phase %d: caret x %.1f, want %.1f", prefix, phase, r.X, want)
				}
			}
			edit(msg(f, "cell"), sel("editWithFrame:inView:editor:delegate:event:"), affixBounds(f, sel("bounds")), f, ed, f, 0)
			if msg(f, "currentEditor") == 0 {
				t.Error("nil-event editing lost the field editor")
			}
			ReleaseFieldAffix(view)
			msg(f, "release")
			w.Release()
		}
	})
}

func TestFieldHintMatchesValueOrigin(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		f := affixField(12, "", false)
		defer msg(f, "release")
		view := NewNSViewOverload1(int64(f))
		defer ReleaseFieldAffix(view)
		affixInit()
		originalDraw := affixDrawInRect
		defer func() { affixDrawInRect = originalDraw }()
		var hintRect NSRect
		affixDrawInRect = func(self, sel uintptr, r NSRect) { hintRect = r }
		var drawingRect func(self, sel uintptr, r NSRect) NSRect
		purego.RegisterFunc(&drawingRect, objcMsgSend)
		for _, a := range []FieldAffix{
			{Hint: "Hint"},
			{Hint: "Hint", Inset: 6},
			{Hint: "Hint", Inset: 6, Prefix: "X", Suffix: "px"},
		} {
			SetFieldAffix(view, a)
			cell := msg(f, "cell")
			bounds := affixBounds(f, sel("bounds"))
			affixes[f].draw(cell, bounds)
			want := drawingRect(cell, sel("drawingRectForBounds:"), bounds)
			if hintRect.X != want.X || hintRect.Y != want.Y {
				t.Errorf("affix %+v: hint origin (%v, %v), want (%v, %v)", a, hintRect.X, hintRect.Y, want.X, want.Y)
			}
		}
	})
}

// affixField makes a field in no window with font size pt and text s.
func affixField(pt float64, s string, bezeled bool) uintptr {
	var systemFont func(cls, sel uintptr, size float64) uintptr
	purego.RegisterFunc(&systemFont, objcMsgSend)
	f := msg(msg(class("NSTextField"), "alloc"), "init")
	msg(f, "setBezeled:", boolArg(bezeled))
	msg(f, "setDrawsBackground:", boolArg(bezeled))
	msg(f, "setFont:", systemFont(class("NSFont"), sel("systemFontOfSize:"), pt))
	msg(f, "setStringValue:", nsString(s))
	msgRectOnly(f, sel("setFrame:"), NSRect{Width: 120, Height: 22})
	return f
}

// renderField draws the field at 2x into a bitmap (no window) and returns it.
func renderField(f uintptr, appearance string) *NSBitmapImageRep {
	if appearance != "" {
		msg(f, "setAppearance:", msg(class("NSAppearance"), "appearanceNamed:", nsString(appearance)))
	}
	b := affixBounds(f, sel("bounds"))
	w, h := int64(2*b.Width), int64(2*b.Height)
	rep := NewNSBitmapImageRep()
	rep.Id = int64(msg(class("NSBitmapImageRep"), "alloc"))
	rep = rep.InitWithBitmapDataPlanes(0, w, h, 8, 4, true, false, OSNSDeviceRGBColorSpace_, 0, 4*w, 32)
	rep.SetSize(NSSize{Width: b.Width, Height: b.Height})
	var cache func(self, sel uintptr, r NSRect, rep uintptr)
	purego.RegisterFunc(&cache, objcMsgSend)
	cache(f, sel("cacheDisplayInRect:toBitmapImageRep:"), b, uintptr(rep.Id))
	return rep
}

// inkColumns returns, per pixel column, whether any pixel there has alpha over 0.25, and the lowest
// inked row of the columns in [from, to).
func inkColumns(rep *NSBitmapImageRep) (cols []bool, bottom func(from, to int) int) {
	w, h := int(rep.PixelsWide()), int(rep.PixelsHigh())
	cols = make([]bool, w)
	low := make([]int, w)
	for x := 0; x < w; x++ {
		low[x] = -1
		for y := 0; y < h; y++ {
			a := NewNSColorOverload1(int64(msg(uintptr(rep.Id), "colorAtX:y:", uintptr(x), uintptr(y)))).AlphaComponent()
			if a > 0.25 {
				cols[x] = true
				low[x] = y
			}
		}
	}
	return cols, func(from, to int) int {
		b := -1
		for x := from; x < to && x < w; x++ {
			b = max(b, low[x])
		}
		return b
	}
}

// runs splits the inked columns into runs separated by more than gap empty columns.
func runs(cols []bool, gap int) [][2]int {
	var out [][2]int
	empty := gap + 1
	for x, on := range cols {
		if !on {
			empty++
			continue
		}
		if empty > gap {
			out = append(out, [2]int{x, x + 1})
		} else {
			out[len(out)-1][1] = x + 1
		}
		empty = 0
	}
	return out
}

func savePNG(t *testing.T, rep *NSBitmapImageRep, name string) {
	dir := os.Getenv("GOWT_AFFIX_PNG")
	if dir == "" {
		return
	}
	data := msg(uintptr(rep.Id), "representationUsingType:properties:", 4, msg(class("NSDictionary"), "dictionary"))
	b := make([]byte, msg(data, "length"))
	msg(data, "getBytes:length:", uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
		t.Error(err)
	}
}

// No Display and no window: the value moves right by the prefix, the cell asks for the affixes' width,
// the prefix sits before the value on its baseline and the suffix after it.
func TestFieldAffixLayout(t *testing.T) {
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		var cellSize func(self, sel uintptr) NSSize
		purego.RegisterFunc(&cellSize, objcMsgSend)

		plain := affixField(11, "120", false)
		defer msg(plain, "release")
		f := affixField(11, "120", false)
		defer msg(f, "release")
		view := NewNSViewOverload1(int64(f))
		SetFieldAffix(view, FieldAffix{Prefix: "W", Suffix: "px"})
		defer ReleaseFieldAffix(view)

		cell, plainCell := msg(f, "cell"), msg(plain, "cell")
		b := affixBounds(f, sel("bounds"))
		r0 := superDrawingRect(plainCell, sel("drawingRectForBounds:"), b)
		var drawingRect func(self, sel uintptr, r NSRect) NSRect
		purego.RegisterFunc(&drawingRect, objcMsgSend)
		r := drawingRect(cell, sel("drawingRectForBounds:"), b)
		st := affixes[f]
		lead, trail := st.widths(cell)
		if lead < 8 || trail < 12 {
			t.Fatalf("lead %.1f, trail %.1f: want the width of W and of px plus the gap", lead, trail)
		}
		if r.X != r0.X+lead || r.Width != r0.Width-lead-trail {
			t.Errorf("drawing rect %+v, want %+v moved by %.1f and narrowed by %.1f", r, r0, lead, lead+trail)
		}
		if got, want := cellSize(cell, sel("cellSize")).Width, cellSize(plainCell, sel("cellSize")).Width+lead+trail; got != want {
			t.Errorf("cellSize width %.1f, want %.1f", got, want)
		}
		if h := st.handleWidth(f); h != r.X {
			t.Errorf("handle width %.1f, want the value start %.1f", h, r.X)
		}

		rep := renderField(f, "")
		defer rep.Release()
		savePNG(t, rep, "affix-ink.png")
		cols, bottom := inkColumns(rep)
		parts := runs(cols, 2*affixGap-3)
		if len(parts) != 3 {
			t.Fatalf("ink runs %v, want three: prefix, value, suffix", parts)
		}
		p, v, s := parts[0], parts[1], parts[2]
		if gap := float64(v[0]-p[1]) / 2; gap < affixGap-1 || gap > affixGap+3 {
			t.Errorf("prefix ends at %d px, value starts at %d px: gap %.1f pt, want about %d", p[1], v[0], gap, affixGap)
		}
		if float64(s[1])/2 > b.Width-r0.X+1 {
			t.Errorf("suffix ends at %d px, past the drawing rect", s[1])
		}
		if pb, vb := bottom(p[0], p[1]), bottom(v[0], v[1]); pb < vb-1 || pb > vb+1 {
			t.Errorf("prefix baseline row %d, value %d: not on one line", pb, vb)
		}
	})
}

// GOWT_AFFIX_PNG=dir keeps bezeled small and regular fields, light and dark, for the eye.
func TestFieldAffixShots(t *testing.T) {
	if os.Getenv("GOWT_AFFIX_PNG") == "" {
		t.Skip("set GOWT_AFFIX_PNG=dir")
	}
	onMain(func() {
		pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
		defer msg(pool, "drain")
		img := msg(class("NSImage"), "imageWithSystemSymbolName:accessibilityDescription:", nsString("arrow.left.and.right"), 0)
		for _, ap := range []string{"NSAppearanceNameAqua", "NSAppearanceNameDarkAqua"} {
			for _, small := range []bool{false, true} {
				pt := 13.0
				if small {
					pt = 11
				}
				for i, a := range []FieldAffix{{Prefix: "X", Suffix: "px"}, {Image: NewNSImageOverload1(int64(img))}, {Prefix: "Gap", Suffix: "%"}} {
					f := affixField(pt, "128", true)
					if small {
						SetControlSize(NewNSViewOverload1(int64(f)), 1)
					}
					SetFieldAffix(NewNSViewOverload1(int64(f)), a)
					rep := renderField(f, ap)
					name := ap[len("NSAppearanceName"):]
					if small {
						name += "-small"
					}
					savePNG(t, rep, name+"-"+string(rune('0'+i))+".png")
					rep.Release()
					ReleaseFieldAffix(NewNSViewOverload1(int64(f)))
					msg(f, "release")
				}
			}
		}
	})
}
