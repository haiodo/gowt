package gowt

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/swt"
)

// affix is what Prefix, PrefixImage, Suffix and Small give a single-line field.
type affix struct {
	prefix, suffix string
	hint           string
	nativeHint     bool
	image          *Image
	small          bool
	drag           func(dx float64, shift bool) // a Number's scrub; nil leaves the prefix inert
}

func (a *affix) shown() bool { return a.prefix != "" || a.image != nil || a.suffix != "" }

// Prefix puts a label such as "X" or "Gap" inside the frame of a single-line Text or a Number, before
// the value, in the secondary text colour; on a Number dragging it changes the value (scrub).
func Prefix(s string) Option {
	return affixOpt("Prefix", func(a *affix) { a.prefix, a.image = s, nil })
}

// PrefixImage puts an icon at its own size there instead of a label. On macOS an icons.Get image
// without a Color takes the secondary text colour; elsewhere an image keeps its own.
func PrefixImage(img *Image) Option {
	return affixOpt("PrefixImage", func(a *affix) { a.image, a.prefix = img, "" })
}

// Suffix puts a unit, such as "%" or "px", inside the frame of a single-line Text or a Number at its
// trailing end, in the secondary text colour.
func Suffix(s string) Option { return affixOpt("Suffix", func(a *affix) { a.suffix = s }) }

func affixOpt(name string, set func(*affix)) Option {
	return hook(func(w Widget) error { return withAffix(w, name, set) })
}

// withAffix runs set on the affixes of the field of w, a single-line Text or a Number.
func withAffix(w Widget, name string, set func(*affix)) error {
	var t *TextView
	switch v := w.(type) {
	case *TextView:
		if v.t.GetStyle()&(swt.MULTI|swt.SEARCH|swt.PASSWORD) == 0 {
			t = v
		}
	case *NumberView:
		t = v.f
	}
	if t == nil {
		return fmt.Errorf("gowt: %s applies to a single-line Text without Search or Password and to Number, not to %s", name, nameOf(w.control()))
	}
	if t.aff == nil {
		t.aff = &affix{}
	}
	set(t.aff)
	return nil
}

// fieldLook runs after the hooks, so the options act in any order: Small sets the font, which the
// Windows frame measures, and the affixes go into that frame.
func fieldLook(t *TextView, noBorder bool) {
	if t.aff != nil && t.aff.small {
		setFont(&t.t.Control, smallFont, func(_ *swt.Control, set func()) { set() })
		setFieldSmall(t.t)
	}
	fluentText(t, noBorder)
	if t.aff != nil && t.aff.prefix == "" && t.aff.image == nil {
		t.aff.drag = nil // nothing to scrub on
	}
	styled := t.fill != nil || t.stroke != nil || t.padX > 0
	if styled && runtime.GOOS == "darwin" {
		if t.aff == nil {
			t.aff = &affix{}
		}
		t.aff.nativeHint = true
		t.aff.hint = t.t.GetMessage()
		t.t.SetMessage("") // drawn in the cell, which puts it on the line of the value
	}
	if t.aff != nil && (t.aff.shown() || t.aff.nativeHint) {
		setFieldAffix(t.t, t.aff)
	}
	if t.fill != nil {
		setPaint(&t.t.Control, t.fill, false)
	}
	if t.fill != nil || t.stroke != nil {
		dark := systemDark()
		if a := appOf(&t.t.Control); a != nil {
			dark = a.dark
			p := a.paints[&t.t.Control]
			if p == nil {
				p = &painted{}
				a.paints[&t.t.Control] = p
				t.t.AddDisposeListener(disposeFunc(func() { delete(a.paints, &t.t.Control) }))
			}
			p.field = t
		}
		t.repaintField(dark)
	}
	if t.padX > 0 {
		setFieldInset(t.t, t.padX)
	}
	if styled {
		t.t.AddListener(swt.FocusIn, listenerFunc(func(*swt.Event) { placeFieldEditor(t.t) }))
	}
}

func (t *TextView) repaintField(dark bool) {
	if t.fill != nil {
		setFieldFill(t.t, t.radius)
	}
	if t.stroke != nil {
		c := paintRGBA(t.stroke, dark)
		setFieldStroke(t.t, c[0], c[1], c[2], c[3])
	}
}

// NumberView is a field that holds a number, see Number.
type NumberView struct {
	base
	f              *TextView // the native field; its text is the shown value
	min, max, step float64
	v              float64
	scrubbed       float64 // points of a scrub not yet a whole step
	listeners      []*func(float64)
}

// Number creates a field for a number within [min, max], at min until Value sets another. Step sets
// the increment, 1 by default; the value is rounded to it and shown with as many decimals as it has.
// A typed number takes effect on Return and when the field loses the focus (text that is no number
// is put back, Escape puts it back too); Up and Down add and subtract a step, ten with Shift; dragging
// the Prefix left or right changes the value by a step per point, ten with Shift, as in Figma. Each of
// these runs OnChange; SetValue and Value do not. The field has a frame unless NoBorder.
// Options: Prefix, PrefixImage, Suffix, Small, Step, NoBorder, Value, OnChange, Bind (float64 or int).
func Number(min, max float64, opts ...Option) Option {
	noBorder := hasStyle(opts, sNoBorder)
	opts = append(opts[:len(opts):len(opts)], lateHook(func(w Widget) error {
		fieldLook(w.(*NumberView).f, noBorder)
		return nil
	}))
	return leaf("Number", sNoBorder, opts, func(p *swt.Composite) Widget {
		t := swt.NewText(p, resolve(swt.BORDER, opts))
		textThemeBackground(t)
		n := &NumberView{base: base{&t.Control}, f: &TextView{base: base{&t.Control}, t: t}, min: min, max: max, step: 1}
		n.f.aff = &affix{drag: n.scrub}
		n.v = n.snap(min)
		t.SetText(n.format(n.v))
		t.AddListener(swt.DefaultSelection, eventFunc(func(*swt.Event) { n.commitText() }))
		t.AddListener(swt.FocusOut, eventFunc(func(*swt.Event) { n.commitText() }))
		t.AddListener(swt.KeyDown, eventFunc(n.key))
		return n
	})
}

// Step sets the increment of a Number (1 by default): the value is rounded to it, Up and Down and a
// scrub of one point add it, and the field shows as many decimals as it has, at most 6.
func Step(step float64) Option {
	return hook(func(w Widget) error {
		n, ok := w.(*NumberView)
		if !ok || step <= 0 {
			return fmt.Errorf("gowt: Step(%v) applies to Number with a step above 0, not to %s", step, nameOf(w.control()))
		}
		n.step = step
		n.SetValue(n.v)
		return nil
	})
}

type eventFunc func(*swt.Event)

func (f eventFunc) HandleEvent(e *swt.Event) { f(e) }

// Value returns the number within [min, max].
func (n *NumberView) Value() float64 { return n.v }

// SetValue sets the number, clamped to [min, max] and rounded to the step, without running OnChange.
func (n *NumberView) SetValue(v float64) {
	n.v = n.snap(v)
	n.show()
}

// OnChange runs f with the new number after the user changed it: by typing, the arrow keys or a scrub.
func (n *NumberView) OnChange(f func(value float64)) { n.listen(f) }

// onUser is the listen of Bind: f runs after each change by the user.
func (n *NumberView) onUser(f func()) (remove func()) { return n.listen(func(float64) { f() }) }

// listen adds f to the listeners and returns its removal.
func (n *NumberView) listen(f func(float64)) (remove func()) {
	p := &f
	n.listeners = append(n.listeners, p)
	return func() {
		for i, q := range n.listeners {
			if q == p {
				n.listeners = append(n.listeners[:i:i], n.listeners[i+1:]...)
				return
			}
		}
	}
}

// Unwrap returns the underlying swt.Text, whose text is the shown number.
func (n *NumberView) Unwrap() *swt.Text { return n.f.t }

// set is a change by the user: the field shows v, and the listeners run when the number changed.
func (n *NumberView) set(v float64) {
	v = n.snap(v)
	old := n.v
	n.v = v
	n.show()
	if v == old {
		return
	}
	for _, f := range n.listeners {
		(*f)(v)
	}
}

func (n *NumberView) show() {
	if s := n.format(n.v); n.f.t.GetText() != s {
		n.f.t.SetText(s)
	}
}

// commitText takes the typed number; text that is no number gets the shown value back. The suffix
// may be typed with it, and a decimal comma counts as a point.
func (n *NumberView) commitText() {
	s := strings.TrimSpace(n.f.t.GetText())
	if suf := n.f.aff.suffix; suf != "" {
		s = strings.TrimSpace(strings.TrimSuffix(s, suf))
	}
	v, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil || math.IsNaN(v) {
		n.show()
		return
	}
	n.set(v)
}

func (n *NumberView) key(e *swt.Event) {
	steps := 1.0
	if e.StateMask&swt.SHIFT != 0 {
		steps = 10
	}
	switch e.KeyCode {
	case swt.ARROW_UP:
	case swt.ARROW_DOWN:
		steps = -steps
	case int32(swt.ESC):
		n.show()
		e.Doit = false
		return
	default:
		return
	}
	e.Doit = false
	n.commitText()
	n.set(n.v + steps*n.step)
	n.f.t.SetSelection(int32(len(n.f.t.GetText())))
}

// scrub turns a drag on the prefix into steps, one per point; the platform reports dx in points.
func (n *NumberView) scrub(dx float64, shift bool) {
	n.scrubbed += dx
	steps := math.Trunc(n.scrubbed)
	if steps == 0 {
		return
	}
	n.scrubbed -= steps
	if shift {
		steps *= 10
	}
	n.set(n.v + steps*n.step)
}

// snap clamps v to [min, max] and rounds it to the step and to the decimals shown.
func (n *NumberView) snap(v float64) float64 {
	v = math.Round(v/n.step) * n.step
	p := math.Pow(10, float64(n.decimals()))
	v = math.Round(v*p) / p
	v = math.Max(n.min, math.Min(n.max, v))
	if v == 0 {
		return 0 // not -0
	}
	return v
}

func (n *NumberView) decimals() int {
	s := strconv.FormatFloat(n.step, 'f', -1, 64)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return min(len(s)-i-1, 6)
	}
	return 0
}

func (n *NumberView) format(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// paintRGBA resolves a paint, with its alpha when it has one.
func paintRGBA(p Paint, dark bool) [4]uint8 {
	if a, ok := p.(interface{ rgba(bool) [4]uint8 }); ok {
		return a.rgba(dark)
	}
	c := p.in(dark)
	return [4]uint8{c.R, c.G, c.B, 255}
}
