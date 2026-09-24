// Hand-written opaque stubs for the rest of manual.txt's list: ToolBar/Caret/IME (Composite/
// Canvas/Decorations/Shell/Button's own field types, not translated yet), plus the graphics/
// accessibility leaf types Control.java reads fields off of. Composite/Canvas/Decorations/Shell
// are real as of Round 5, Menu/MenuItem as of Round 7, ScrollBar as of Round 8.
package swt

import "github.com/haiodo/gowt/internal/cocoa"

// ToolBar: only ever stored/returned by Shell, never otherwise touched on this round's path.
type ToolBar struct {
	Composite
	itemCount int32
}

func NewToolBar(parent *Composite, style int32, internal bool) *ToolBar { return &ToolBar{} }

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
