//go:build darwin

// Hand-written: a label, an icon and a unit inside the frame of a single-line text field, and the
// scrub handle over the label of a numeric field.
package cocoa

import (
	"math"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// FieldAffix is what SetFieldAffix shows inside a single-line NSTextField around its value.
type FieldAffix struct {
	// Inset is the space in points between the field's edge and the text, left and right.
	Inset  float64
	Prefix string
	Image  *NSImage // drawn instead of Prefix; a template image takes the colour of the prefix
	Suffix string
	// Hint is the placeholder drawn where the value starts while the field is empty.
	Hint string
	// Drag makes the prefix a scrub handle: a drag reports each horizontal move in points, a click
	// without a move starts editing with the text selected.
	Drag func(dx float64, shift bool)
}

// affixGap is the space in points between an affix and the value.
const affixGap = 4

type affixState struct {
	FieldAffix
	handle uintptr
	moved  bool
}

// affixes holds the state of each field by its view; UI thread only.
var affixes = map[uintptr]*affixState{}

var (
	affixCellClass, scrubClass uintptr
	superDrawingRect           func(self, sel uintptr, r NSRect) NSRect
	superCellSize              func(self, sel uintptr, r NSRect) NSSize
	superTitleRect             func(self, sel uintptr, r NSRect) NSRect
	superDrawInterior          func(self, sel uintptr, r NSRect, view uintptr)
	affixDrawInRect            func(self, sel uintptr, r NSRect)
	affixDrawImage             func(self, sel uintptr, r, from NSRect, op uintptr, fraction float64, flipped bool, hints uintptr)
	affixBounds                func(self, sel uintptr) NSRect
	affixAddCursorRect         func(self, sel uintptr, r NSRect, cursor uintptr)
	affixConvertPoint          func(self, sel uintptr, p NSPoint, view uintptr) NSPoint
	affixConvertRect           func(self, sel uintptr, r NSRect, view uintptr) NSRect
	affixDeltaX                func(self, sel uintptr) float64
	affixWithAlpha             func(self, sel uintptr, a float64) uintptr
	affixTransparency          func(cg, aux uintptr)
	affixEndTransparency       func(cg uintptr)
	affixSetAlpha              func(cg uintptr, a float64)
)

// affixInit registers GowtAffixFieldCell and GowtScrubHandle. The cell is no subclass of SWT's: SWT
// calls super through [self superclass], which in a subclass of its class finds its own method again.
func affixInit() {
	if affixCellClass != 0 {
		return
	}
	objcInit()
	tfc := class("NSTextFieldCell")
	purego.RegisterFunc(&superDrawingRect, uintptr(OSClass_getMethodImplementation(int64(tfc), int64(sel("drawingRectForBounds:")))))
	purego.RegisterFunc(&superCellSize, uintptr(OSClass_getMethodImplementation(int64(tfc), int64(sel("cellSizeForBounds:")))))
	purego.RegisterFunc(&superTitleRect, uintptr(OSClass_getMethodImplementation(int64(tfc), int64(sel("titleRectForBounds:")))))
	purego.RegisterFunc(&superDrawInterior, uintptr(OSClass_getMethodImplementation(int64(tfc), int64(sel("drawInteriorWithFrame:inView:")))))
	for _, f := range []any{&affixDrawInRect, &affixDrawImage, &affixBounds, &affixAddCursorRect, &affixConvertPoint, &affixConvertRect, &affixDeltaX, &affixWithAlpha} {
		purego.RegisterFunc(f, objcMsgSend)
	}
	purego.RegisterLibFunc(&affixTransparency, purego.RTLD_DEFAULT, "CGContextBeginTransparencyLayer")
	purego.RegisterLibFunc(&affixEndTransparency, purego.RTLD_DEFAULT, "CGContextEndTransparencyLayer")
	purego.RegisterLibFunc(&affixSetAlpha, purego.RTLD_DEFAULT, "CGContextSetAlpha")

	drawingRect := NewCallback(func(self, cmd uintptr, r NSRect) NSRect {
		out := affixDrawingRect(self, r)
		if st := affixOfCell(self); st != nil {
			lead, trail := st.widths(self)
			out.X += lead
			out.Width = math.Max(out.Width-lead-trail, 0)
		}
		return out
	})
	cellSize := NewCallback(func(self, cmd uintptr, r NSRect) NSSize {
		out := superCellSize(self, cmd, r)
		if st := affixOfCell(self); st != nil {
			lead, trail := st.widths(self)
			out.Width += lead + trail
		}
		return out
	})
	drawInterior := NewCallback(func(self, cmd uintptr, r NSRect, view uintptr) {
		superDrawInterior(self, cmd, r, view)
		if st := affixOfCell(self); st != nil {
			st.draw(self, r)
		}
	})
	affixCellClass = allocClass(tfc, "GowtAffixFieldCell", 0)
	addMethod(affixCellClass, sel("drawingRectForBounds:"), drawingRect, "{CGRect={CGPoint=dd}{CGSize=dd}}@:{CGRect={CGPoint=dd}{CGSize=dd}}")
	addMethod(affixCellClass, sel("titleRectForBounds:"), drawingRect, "{CGRect={CGPoint=dd}{CGSize=dd}}@:{CGRect={CGPoint=dd}{CGSize=dd}}")
	addMethod(affixCellClass, sel("cellSizeForBounds:"), cellSize, "{CGSize=dd}@:{CGRect={CGPoint=dd}{CGSize=dd}}")
	addMethod(affixCellClass, sel("drawInteriorWithFrame:inView:"), drawInterior, "v@:{CGRect={CGPoint=dd}{CGSize=dd}}@")
	registerClass(affixCellClass)

	// The handle fills the field and takes only the points over the prefix (hitTest:), so a later font
	// keeps them in step with the drawing.
	hitTest := NewCallback(func(self, cmd uintptr, p NSPoint) uintptr {
		field := msg(self, "superview")
		st := affixes[field]
		if st == nil {
			return 0
		}
		local := affixConvertPoint(self, sel("convertPoint:fromView:"), p, field)
		if local.X < 0 || local.X >= st.handleWidth(field) {
			return 0
		}
		return self
	})
	yes := purego.NewCallback(func(self, cmd, arg uintptr) uintptr { return 1 })
	no := purego.NewCallback(func(self, cmd uintptr) uintptr { return 0 })
	cursorRects := purego.NewCallback(func(self, cmd uintptr) {
		field := msg(self, "superview")
		if st := affixes[field]; st != nil {
			b := affixBounds(self, sel("bounds"))
			b.Width = math.Min(b.Width, st.handleWidth(field))
			affixAddCursorRect(self, sel("addCursorRect:cursor:"), b, msg(class("NSCursor"), "resizeLeftRightCursor"))
		}
	})
	down := purego.NewCallback(func(self, cmd, event uintptr) {
		if st := affixes[msg(self, "superview")]; st != nil {
			st.moved = false
			msg(msg(class("NSCursor"), "resizeLeftRightCursor"), "push")
		}
	})
	dragged := purego.NewCallback(func(self, cmd, event uintptr) {
		st := affixes[msg(self, "superview")]
		if st == nil || st.Drag == nil {
			return
		}
		dx := affixDeltaX(event, sel("deltaX"))
		if dx == 0 {
			return
		}
		st.moved = true
		const shiftKey = 1 << 17 // NSEventModifierFlagShift
		st.Drag(dx, msg(event, "modifierFlags")&shiftKey != 0)
	})
	up := purego.NewCallback(func(self, cmd, event uintptr) {
		field := msg(self, "superview")
		st := affixes[field]
		if st == nil {
			return
		}
		msg(class("NSCursor"), "pop")
		if !st.moved && msg(field, "isEnabled")&0xff != 0 && msg(field, "isEditable")&0xff != 0 {
			// As a click on the text does, but with all of it selected, ready to type over.
			msg(field, "selectText:", 0)
		}
	})
	scrubClass = allocClass(class("NSView"), "GowtScrubHandle", 0)
	addMethod(scrubClass, sel("hitTest:"), hitTest, "@@:{CGPoint=dd}")
	addMethod(scrubClass, sel("acceptsFirstMouse:"), yes, "c@:@")
	addMethod(scrubClass, sel("mouseDownCanMoveWindow"), no, "c@:")
	addMethod(scrubClass, sel("resetCursorRects"), cursorRects, "v@:")
	addMethod(scrubClass, sel("mouseDown:"), down, "v@:@")
	addMethod(scrubClass, sel("mouseDragged:"), dragged, "v@:@")
	addMethod(scrubClass, sel("mouseUp:"), up, "v@:@")
	registerClass(scrubClass)
}

// affixStateOf is the state of the field v, with the cell class installed on first use.
func affixStateOf(v uintptr) *affixState {
	st := affixes[v]
	if st == nil {
		st = &affixState{}
		affixes[v] = st
		if cell := msg(v, "cell"); cell != 0 && uintptr(OSObject_getClass(int64(cell))) != affixCellClass {
			OSObject_setClass(int64(cell), int64(affixCellClass))
		}
	}
	return st
}

func affixOfCell(cell uintptr) *affixState {
	if v := msg(cell, "controlView"); v != 0 {
		return affixes[v]
	}
	return nil
}

// SetFieldAffix shows a in the single-line text field view; the cell moves to GowtAffixFieldCell and
// keeps its state. Call ReleaseFieldAffix when the view is disposed.
func SetFieldAffix(view *NSView, a FieldAffix) {
	affixInit()
	v := uintptr(view.Id)
	st := affixStateOf(v)
	st.FieldAffix = a
	if a.Drag != nil && st.handle == 0 {
		h := msg(msg(scrubClass, "alloc"), "init")
		msgRectOnly(h, sel("setFrame:"), affixBounds(v, sel("bounds")))
		msg(h, "setAutoresizingMask:", 2|16) // NSViewWidthSizable | NSViewHeightSizable
		msg(v, "addSubview:", h)
		msg(h, "release")
		st.handle = h
	}
	if a.Drag == nil && st.handle != 0 {
		msg(st.handle, "removeFromSuperview")
		st.handle = 0
	}
	msg(v, "invalidateIntrinsicContentSize")
	msg(v, "setNeedsDisplay:", 1)
	if w := msg(v, "window"); w != 0 {
		msg(w, "invalidateCursorRectsForView:", v)
	}
}

// ReleaseFieldAffix forgets the affixes of view; the cell keeps its class and draws as a plain field.
func ReleaseFieldAffix(view *NSView) { delete(affixes, uintptr(view.Id)) }

// FieldAffixHandle returns the scrub handle view of a field with a Drag, 0 without one.
func FieldAffixHandle(view *NSView) uintptr {
	if st := affixes[uintptr(view.Id)]; st != nil {
		return st.handle
	}
	return 0
}

// FieldValueStart is the x in field points where the value of a field with affixes starts.
func FieldValueStart(view *NSView) float64 {
	if st := affixes[uintptr(view.Id)]; st != nil {
		return st.handleWidth(uintptr(view.Id))
	}
	return 0
}

// fieldEmpty reports whether the field of the cell has no text.
func fieldEmpty(cell uintptr) bool {
	field := msg(cell, "controlView")
	return field == 0 || msg(msg(field, "stringValue"), "length") == 0
}

// PlaceFieldEditor puts the field editor of view on the line of its text: AppKit places it at the
// top of a borderless field, its title rect notwithstanding.
func PlaceFieldEditor(view *NSView) {
	v := uintptr(view.Id)
	ed := msg(v, "currentEditor")
	if ed == 0 {
		return
	}
	cell := msg(v, "cell")
	r := affixDrawingRect(cell, affixBounds(v, sel("bounds")))
	if st := affixes[v]; st != nil {
		lead, trail := st.widths(cell)
		r.X += lead
		r.Width = math.Max(r.Width-lead-trail, 0)
	}
	if super := msg(ed, "superview"); super != v {
		r = affixConvertRect(v, sel("convertRect:toView:"), r, super)
	}
	msgRectOnly(ed, sel("setFrame:"), r)
}

// FieldEditorX is the x in field points of the field editor while the field is being edited.
func FieldEditorX(view *NSView) (x float64, ok bool) {
	affixInit()
	v := uintptr(view.Id)
	ed := msg(v, "currentEditor")
	if ed == 0 {
		return 0, false
	}
	return affixConvertPoint(v, sel("convertPoint:fromView:"), NSPoint{}, ed).X, true
}

// SetControlSize sets NSControlSize: 0 regular, 1 small, 2 mini.
func SetControlSize(view *NSView, size int) { msg(uintptr(view.Id), "setControlSize:", uintptr(size)) }

// widths returns the points the value gives up at the leading and trailing end of the cell.
func (st *affixState) widths(cell uintptr) (lead, trail float64) {
	if w := st.prefixWidth(cell); w > 0 {
		lead = w + affixGap
	}
	if st.Suffix != "" {
		trail = math.Ceil(affixString(cell, st.Suffix).Size().Width) + affixGap
	}
	if st.Inset > 0 {
		lead += st.Inset
		trail += st.Inset
	}
	return lead, trail
}

func (st *affixState) prefixWidth(cell uintptr) float64 {
	if st.Image != nil {
		return math.Ceil(st.Image.Size().Width)
	}
	if st.Prefix != "" {
		return math.Ceil(affixString(cell, st.Prefix).Size().Width)
	}
	return 0
}

// handleWidth is the x in field points where the value starts: the handle takes everything before it.
func (st *affixState) handleWidth(field uintptr) float64 {
	cell := msg(field, "cell")
	b := affixBounds(field, sel("bounds"))
	r := superDrawingRect(cell, sel("drawingRectForBounds:"), b)
	lead, _ := st.widths(cell)
	return r.X + lead
}

// affixColor is the secondary label colour, tertiary while the cell is disabled.
func affixColor(cell uintptr) uintptr {
	if msg(cell, "isEnabled")&0xff == 0 {
		return msg(class("NSColor"), "tertiaryLabelColor")
	}
	return msg(class("NSColor"), "secondaryLabelColor")
}

// affixString is s in the cell's font and affixColor.
func affixString(cell uintptr, s string) *NSAttributedString {
	keys := [2]uintptr{uintptr(OSNSFontAttributeName()), uintptr(OSNSForegroundColorAttributeName())}
	values := [2]uintptr{msg(cell, "font"), affixColor(cell)}
	attrs := msg(class("NSDictionary"), "dictionaryWithObjects:forKeys:count:", uintptr(unsafe.Pointer(&values[0])), uintptr(unsafe.Pointer(&keys[0])), 2)
	runtime.KeepAlive(keys)
	runtime.KeepAlive(values)
	str := msg(msg(msg(class("NSAttributedString"), "alloc"), "initWithString:attributes:", nsString(s), attrs), "autorelease")
	return NewNSAttributedStringOverload1(int64(str))
}

// draw puts the affixes on the baseline of the value: same font, same top of the drawing rectangle.
// pad is the line fragment padding before the first glyph of the value.
// affixDrawingRect is the cell's drawing rect, centred on the line of a borderless field.
func affixDrawingRect(cell uintptr, r NSRect) NSRect {
	out := superTitleRect(cell, sel("titleRectForBounds:"), r)
	// A borderless cell's title rect is the whole bounds, which puts the text and the editor at the top.
	if h := lineHeightOf(cell); h > 0 && h < out.Height {
		out.Y += (out.Height - h) / 2
		out.Height = h
	}
	return out
}

// lineHeightOf is the font's line height of a cell, 0 without a font.
func lineHeightOf(cell uintptr) float64 {
	font := msg(cell, "font")
	if font == 0 {
		return 0
	}
	if msgToF64 == nil {
		purego.RegisterFunc(&msgToF64, objcMsgSend)
	}
	return msgToF64(font, sel("ascender")) - msgToF64(font, sel("descender")) + msgToF64(font, sel("leading"))
}

func (st *affixState) draw(cell uintptr, frame NSRect) {
	r := affixDrawingRect(cell, frame)
	r.X += st.Inset
	r.Width = math.Max(r.Width-2*st.Inset, 0)
	_, trail := st.widths(cell)
	const pad = 2
	switch {
	case st.Image != nil:
		size := st.Image.Size()
		box := NSRect{X: r.X + pad, Y: r.Y + (r.Height-size.Height)/2, Width: size.Width, Height: size.Height}
		if msg(uintptr(st.Image.Id), "isTemplate")&0xff != 0 {
			st.drawTinted(cell, box)
		} else {
			st.drawImage(box)
		}
	case st.Prefix != "":
		s := affixString(cell, st.Prefix)
		affixDrawInRect(uintptr(s.Id), sel("drawInRect:"), NSRect{X: r.X + pad, Y: r.Y, Width: s.Size().Width + 1, Height: r.Height})
	}
	if st.Suffix != "" {
		s := affixString(cell, st.Suffix)
		w := s.Size().Width
		affixDrawInRect(uintptr(s.Id), sel("drawInRect:"), NSRect{X: r.X + r.Width - trail + affixGap - pad, Y: r.Y, Width: w + 1, Height: r.Height})
	}
	if st.Hint != "" && fieldEmpty(cell) {
		lead, _ := st.widths(cell)
		s := affixString(cell, st.Hint)
		affixDrawInRect(uintptr(s.Id), sel("drawInRect:"), NSRect{X: r.X + lead, Y: r.Y, Width: s.Size().Width + 1, Height: r.Height})
	}
}

func (st *affixState) drawImage(box NSRect) {
	affixDrawImage(uintptr(st.Image.Id), sel("drawInRect:fromRect:operation:fraction:respectFlipped:hints:"), box, NSRect{}, 2, 1, true, 0)
}

// drawTinted draws a template image in affixColor: the opaque colour over the image's pixels in a
// transparency layer, the layer at the colour's alpha. A translucent fill over black stays black.
func (st *affixState) drawTinted(cell uintptr, box NSRect) {
	ctx := msg(class("NSGraphicsContext"), "currentContext")
	color := msg(affixColor(cell), "colorUsingColorSpace:", msg(class("NSColorSpace"), "sRGBColorSpace"))
	if ctx == 0 || color == 0 {
		return
	}
	cg := msg(ctx, "CGContext")
	msg(ctx, "saveGraphicsState")
	affixSetAlpha(cg, NewNSColorOverload1(int64(color)).AlphaComponent())
	affixTransparency(cg, 0)
	st.drawImage(box)
	msg(affixWithAlpha(color, sel("colorWithAlphaComponent:"), 1), "setFill")
	OSNSRectFillUsingOperation(box, int64(OSNSCompositingOperationSourceAtop))
	affixEndTransparency(cg)
	msg(ctx, "restoreGraphicsState")
}
