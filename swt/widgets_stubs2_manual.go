// Hand-written opaque stubs for the rest of manual.txt's list: Menu/ScrollBar (each embeds its
// real manual superclass per Manual.WIDGET_SUPER, so promoted-field upcasts like &x.Control keep
// working), MenuItem/ToolBar/Caret/IME (Composite/Canvas/Decorations/Shell/Button's own field
// types, not translated yet), plus the graphics/accessibility leaf types Control.java reads
// fields off of. Composite/Canvas/Decorations/Shell are real as of Round 5 - removed from here.
package swt

import "github.com/haiodo/gowt/internal/cocoa"

// org.eclipse.swt.graphics.GCData: Control's own internal_new_GC/internal_dispose_GC bodies
// (translated for real) read/write every one of these fields directly.
type GCData struct {
	Device                 *Device
	Style, State           int32
	Foreground, Background []float64
	Font                   *Font
	Image                  *Image
	// A null NSRect is its zero value (see README "Struct (value-type) classes").
	PaintRect      cocoa.NSRect
	VisibleRgn     int64
	View           *cocoa.NSView
	Thread         any
	FlippedContext *cocoa.NSGraphicsContext
}

func NewGCData() *GCData { return &GCData{} }

type Menu struct {
	Widget
	parent *Decorations
	nsMenu *cocoa.NSMenu
}

// Real Menu has Menu(Display)/Menu(Display, NSMenu)/... constructors; only Display's app menu
// bar code (off the Shell+Button path) creates one.
func NewMenu(args ...any) *Menu { panic("stub until translated: Menu") }

func (m *Menu) _setVisible(visible bool)        {}
func (m *Menu) GetEnabled() bool                { return true }
func (m *Menu) SetLocation(x int32, y int32)    {}
func (m *Menu) SetVisible(visible bool)         {}
func (m *Menu) FixMenus(newParent *Decorations) { m.parent = newParent }
func (m *Menu) GetItems() []*MenuItem           { return nil }

// MenuItem: Shell.java reads a handful of members off its escMenuItem/menu items (ESC-key
// dismissal, submenu tracking) - none of it wired to a real Menu yet.
type MenuItem struct {
	Widget
	nsItem *cocoa.NSMenuItem
}

func NewMenuItem(args ...any) *MenuItem { panic("stub until translated: MenuItem") }
func (mi *MenuItem) GetEnabled() bool   { return true }

func (mi *MenuItem) GetAccelerator() int32 { return 0 }
func (mi *MenuItem) GetParent() *Menu      { return nil }
func (mi *MenuItem) GetMenu() *Menu        { return nil }

// ToolBar: only ever stored/returned by Shell, never otherwise touched on this round's path.
type ToolBar struct {
	Composite
	itemCount int32
}

func NewToolBar(parent *Composite, style int32, internal bool) *ToolBar { return &ToolBar{} }

type ScrollBar struct {
	Widget
	parent         *Scrollable
	view           *cocoa.NSScroller
	actionSelector int64
	target         *cocoa.Id
	enabled        bool
	selection      int32
	increment      int32
}

func NewScrollBar() *ScrollBar { return &ScrollBar{} }

func (b *ScrollBar) UpdateBar(x int32, y int32, width int32, height int32) {}

func (b *ScrollBar) SendSelection() {}

func (b *ScrollBar) SendSelectionEvent(eventType int32, event *Event, send bool) {}

func (b *ScrollBar) GetEnabled() bool     { return b.enabled }
func (b *ScrollBar) GetSelection() int32  { return b.selection }
func (b *ScrollBar) GetIncrement() int32  { return b.increment }
func (b *ScrollBar) SetSelection(v int32) { b.selection = v }

// Caret: Canvas.java's caret code all null-checks before use, so a nil *Caret already matches
// real SWT's "no caret installed" behavior. Fields are package-private in Java - lowercase.
type Caret struct {
	isShowing           bool
	blinkRate           int32
	image               *Image
	x, y, width, height int32
}

const CaretDEFAULT_WIDTH int32 = 1

func (c *Caret) IsDisposed() bool   { return c == nil }
func (c *Caret) BlinkCaret() bool   { return false }
func (c *Caret) Release(bool)       {}
func (c *Caret) Reskin(flags int32) {}
func (c *Caret) IsFocusCaret() bool { return false }
func (c *Caret) KillFocus()         {}
func (c *Caret) SetFocus()          {}
func (c *Caret) SetFont(font *Font) {}

// IME: same nil-checked-before-use shape as Caret. Every param is a raw objc-runtime int64
// handle (id/SEL/pointer), matching IME.java's own native-bridge method signatures.
type IME struct {
	startOffset int32
}

func (i *IME) IsDisposed() bool                                                       { return i == nil }
func (i *IME) AttributedSubstringFromRange(id int64, sel int64, rangePtr int64) int64 { return 0 }
func (i *IME) CharacterIndexForPoint(id int64, sel int64, point int64) int64          { return 0 }
func (i *IME) FirstRectForCharacterRange(id int64, sel int64, r int64) cocoa.NSRect {
	return cocoa.NSRect{}
}
func (i *IME) HasMarkedText(id int64, sel int64) bool          { return false }
func (i *IME) IsInlineEnabled() bool                           { return false }
func (i *IME) InsertText(id int64, sel int64, str int64) bool  { return true }
func (i *IME) MarkedRange(id int64, sel int64) cocoa.NSRange   { return cocoa.NSRange{} }
func (i *IME) Release(bool)                                    {}
func (i *IME) Reskin(flags int32)                              {}
func (i *IME) SelectedRange(id int64, sel int64) cocoa.NSRange { return cocoa.NSRange{} }
func (i *IME) SetMarkedText_selectedRange(id int64, sel int64, str int64, selRange int64) bool {
	return true
}
func (i *IME) ValidAttributesForMarkedText(id int64, sel int64) int64 { return 0 }

type Image struct {
	Handle *cocoa.NSImage
}

func ImageCocoa_new(device *Device, typ int32, nsImage *cocoa.NSImage) *Image {
	return &Image{Handle: nsImage}
}

func (i *Image) Dispose()                 {}
func (i *Image) IsDisposed() bool         { return i == nil }
func (i *Image) GetBounds() *Rectangle    { panic("stub until translated: Image.getBounds") }
func (i *Image) GetImageData() *ImageData { panic("stub until translated: Image.getImageData") }

// ImageData: Decorations.java's multi-resolution icon selection (setImages/compare) only ever
// runs when more than one image is provided - not exercised by a minimal Shell+Button window.
type ImageData struct {
	Width, Height int32
}

func (d *ImageData) GetTransparencyType() int32 { return 0 }

type Cursor struct {
	Handle *cocoa.NSCursor
}

func NewCursor(device *Device, style int32) *Cursor { panic("stub until translated: Cursor") }

func (c *Cursor) Dispose()         {}
func (c *Cursor) IsDisposed() bool { return c == nil }

type Region struct {
	Handle int64
}

func RegionCocoa_new(device *Display, handle int64) *Region { return &Region{Handle: handle} }

func (r *Region) IsDisposed() bool      { return r == nil }
func (r *Region) GetBounds() *Rectangle { return &Rectangle{} }

// Accessible bridge methods Control.java calls - no-op/false/nil, matching real SWT's own
// behavior for a Control with no screen-reader Accessible attached (the common case).
type Accessible struct{}

func AccessibleInternal_new_Accessible(control *Control) *Accessible { return &Accessible{} }

func (a *Accessible) Internal_accessibilityActionDescription(name *cocoa.NSString, childID int32) *cocoa.Id {
	return nil
}
func (a *Accessible) Internal_accessibilityActionNames(childID int32) *cocoa.NSArray    { return nil }
func (a *Accessible) Internal_accessibilityAttributeNames(childID int32) *cocoa.NSArray { return nil }
func (a *Accessible) Internal_accessibilityParameterizedAttributeNames(childID int32) *cocoa.NSArray {
	return nil
}
func (a *Accessible) Internal_accessibilityIsIgnored(childID int32) bool { return false }
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
