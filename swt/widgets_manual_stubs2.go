// Hand-written opaque stubs for the rest of Round 4's manual.txt list: the widget-hierarchy
// chain above Control/Scrollable (Composite/Canvas/Decorations/Shell/Menu/ScrollBar - each embeds
// its real manual superclass per Manual.WIDGET_SUPER, so promoted-field upcasts like &x.Control
// keep working) plus the graphics/accessibility leaf types Control.java reads fields off of.
package swt

import "github.com/haiodo/gowt/internal/cocoa"

type Composite struct {
	Scrollable
	backgroundMode int32
}

func (c *Composite) RemoveControl(control *Control) {}

func (c *Composite) _getChildren() []*Control { return nil }

func (c *Composite) _getTabList() []*Control { return nil }

// org.eclipse.swt.graphics.GCData: Control's own internal_new_GC/internal_dispose_GC bodies
// (translated for real) read/write every one of these fields directly.
type GCData struct {
	Device                 *Display
	Style, State           int32
	Foreground, Background []float64
	Font                   *Font
	// any, not cocoa.NSRect: both assigned a plain value and null-checked - a bare struct
	// can't be compared to nil, boxing in an interface can (nil until first assigned).
	PaintRect      any
	VisibleRgn     int64
	View           *cocoa.NSView
	Thread         any
	FlippedContext *cocoa.NSGraphicsContext
}

func NewGCData() *GCData { return &GCData{} }

type Canvas struct {
	Composite
}

type Decorations struct {
	Canvas
}

func (d *Decorations) SetSavedFocus(control *Control) {}

func (d *Decorations) FixDecorations(newDecorations *Decorations, control *Control, menus []*Menu) {}

func (d *Decorations) BringToTop(force bool) {}

type Shell struct {
	Decorations
	deferFlushing    bool
	scrolling        bool
	keyInputHappened bool
}

func (s *Shell) BringToTop(bool)                            {}
func (s *Shell) SendToolTipEvent(bool)                      {}
func (s *Shell) GetModalShell() *Shell                      { return nil }
func (s *Shell) Layout(controls []*Control, flags int32)    {}
func (s *Shell) FixShell(newShell *Shell, control *Control) {}

// Real Shell.getShell() overrides Control's climb-to-parent version to stop at itself; without
// it, the promoted Control.GetShell() (this.parent.GetShell()) recurses until parent is nil.
func (s *Shell) GetShell() *Shell { return s }

// setActiveControl has 2 real overloads (Control) and (Control, int detail); Manual.instanceMember
// has no overload awareness, so a variadic sink covers both call shapes with the one Go name.
func (s *Shell) SetActiveControl(control *Control, detail ...int32) {}

type Menu struct {
	Widget
	parent *Decorations
}

func (m *Menu) SetLocation(x int32, y int32) {}
func (m *Menu) SetVisible(visible bool)      {}

type ScrollBar struct {
	Widget
	parent         *Scrollable
	view           *cocoa.NSScroller
	actionSelector int64
	target         *cocoa.Id
}

func NewScrollBar() *ScrollBar { return &ScrollBar{} }

func (b *ScrollBar) UpdateBar(x int32, y int32, width int32, height int32) {}

func (b *ScrollBar) SendSelection() {}

type Font struct {
	Handle      *cocoa.NSFont
	ExtraTraits int32
}

func FontCocoa_new(device *Display, handle *cocoa.NSFont) *Font { return &Font{Handle: handle} }

func (f *Font) IsDisposed() bool { return f == nil }

type Color struct {
	Handle []float64
}

// Real Color.cocoa_new has a 2-arg and a 3-arg (explicit alpha) overload; Manual.instanceMember
// has no overload awareness, so a variadic trailing param covers both call shapes.
func ColorCocoa_new(device *Display, background []float64, alpha ...int32) *Color {
	return &Color{Handle: background}
}

func (c *Color) IsDisposed() bool { return c == nil }
func (c *Color) GetAlpha() int32  { return 255 }

type Image struct {
	Handle *cocoa.NSImage
}

func (i *Image) IsDisposed() bool      { return i == nil }
func (i *Image) GetBounds() *Rectangle { panic("stub until translated: Image.getBounds") }

type Cursor struct {
	Handle *cocoa.NSCursor
}

func (c *Cursor) IsDisposed() bool { return c == nil }

type Region struct {
	Handle int64
}

func RegionCocoa_new(device *Display, handle int64) *Region { return &Region{Handle: handle} }

func (r *Region) IsDisposed() bool { return r == nil }

// Accessible bridge methods Control.java calls - no-op/false/nil, matching real SWT's own
// behavior for a Control with no screen-reader Accessible attached (the common case).
type Accessible struct{}

func AccessibleInternal_new_Accessible(control *Control) *Accessible { return &Accessible{} }

func (a *Accessible) Internal_accessibilityActionDescription(name *cocoa.NSString, childID int32) *cocoa.Id {
	return nil
}
func (a *Accessible) Internal_accessibilityActionNames(childID int32) *cocoa.NSArray { return nil }
func (a *Accessible) Internal_accessibilityAttributeNames(childID int32) *cocoa.Id   { return nil }
func (a *Accessible) Internal_accessibilityParameterizedAttributeNames(childID int32) *cocoa.NSArray {
	return nil
}
func (a *Accessible) Internal_accessibilityPerformAction(action *cocoa.NSString, childID int32) bool {
	return false
}
func (a *Accessible) Internal_accessibilityFocusedUIElement(childID int32) *cocoa.Id { return nil }
func (a *Accessible) Internal_accessibilityHitTest(point cocoa.NSPoint, childID int32) *cocoa.Id {
	return nil
}
func (a *Accessible) Internal_accessibilityAttributeValue(attribute *cocoa.NSString, childID int32) *cocoa.Id {
	return nil
}
func (a *Accessible) Internal_accessibilityAttributeValue_forParameter(attribute *cocoa.NSString, parameter *cocoa.Id, childID int32) *cocoa.Id {
	return nil
}
func (a *Accessible) Internal_accessibilityIsAttributeSettable(attribute *cocoa.NSString, childID int32) bool {
	return false
}
func (a *Accessible) Internal_accessibilitySetValue_forAttribute(value *cocoa.Id, attribute *cocoa.NSString, childID int32) {
}
func (a *Accessible) Internal_addRelationAttributes(attributes int64) int64 { return attributes }
func (a *Accessible) Internal_dispose_Accessible()                          {}

// org.eclipse.swt.accessibility.ACC.CHILDID_SELF - the only ACC member the translated set reads.
const ACCCHILDID_SELF int32 = -1

// org.eclipse.swt.internal.WidgetSpy: creation/disposal tracking, off by default (matches the
// real class's own isEnabled starting false).
var WidgetSpyIsEnabled bool

type widgetSpy struct{}

func (widgetSpy) WidgetCreated(w *Widget)  {}
func (widgetSpy) WidgetDisposed(w *Widget) {}

func WidgetSpyGetInstance() widgetSpy { return widgetSpy{} }

// Reflective Callback(Object, methodName, argCount) dispatch - distinct from the real
// cocoa.NewCallback closure mechanism (README "Callback design"), unsupported-marker only.
type Callback struct{}

func NewCallback(receiver any, method string, argCount int32) *Callback {
	panic("stub until translated: internal.Callback reflective dispatch")
}

func (c *Callback) GetAddress() int64 { panic("stub until translated: Callback.getAddress") }
func (c *Callback) Dispose()          {}
