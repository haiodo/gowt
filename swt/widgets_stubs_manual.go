// Hand-written opaque stubs for widget/graphics types not translated yet (manual.txt): Widget/
// Control/Scrollable are real as of Round 4; Display/GC carry the fields/methods Round 4 needs,
// the rest of Round 4's manual.txt list lives in swt/widgets_manual_stubs2.go.
package swt

import (
	"fmt"
	"reflect"

	"github.com/haiodo/gowt/internal/cocoa"
)

// Display's event/focus/touch/gesture bookkeeping: field/method shapes only, no real event loop.
type Display struct {
	thread                    any // no thread-affinity checking yet - see README known gaps
	tooltipTarget             *Widget
	tooltipControl            *Control
	isPainting                *cocoa.NSMutableArray
	needsDisplay              *cocoa.NSMutableArray
	needsDisplayInRect        *cocoa.NSMutableArray
	focusEvent                int32
	focusControl              *Control
	ignoreFocusControl        *Control
	currentControl            *Control
	trackingControl           *Control
	clickCount                int32
	smallFonts                bool
	rotation                  float64
	magnification             float64
	gestureActive             bool
	touchCounter              int32
	primaryIdentifier         int64
	lastHandledMenuForEventId int64
	contexts                  []*GCData
	hoverTimer                any
	sendEvent                 bool
}

func (d *Display) Filters(eventType int32) bool { return false }

func (d *Display) IsValidThread() bool { return true }

func (d *Display) AddSkinnableWidget(w *Widget) {}

func (d *Display) AddWidget(view *cocoa.NSObject, w *Widget) {}

func (d *Display) RemoveWidget(view *cocoa.NSObject) *Widget { return nil }

func (d *Display) AddContext(data *GCData) {}

func (d *Display) RemoveContext(data *GCData) {}

func (d *Display) SendEvent(table *EventTable, event *Event) {}

func (d *Display) PostEvent(event *Event) {}

func (d *Display) GetLastEventTime() int32 { return 0 }

func (d *Display) GetActiveShell() *Shell { return nil }

func (d *Display) GetFocusControl() *Control { return nil }

func (d *Display) GetModalDialog() *Dialog { return nil }

func (d *Display) GetModalPanel() *cocoa.NSPanel { return nil }

func (d *Display) GetPrimaryFrame() cocoa.NSRect { return cocoa.NSRect{} }

func (d *Display) GetMonitors() []*Monitor { return nil }

func (d *Display) GetWidgetColor(id int32) *Color { return nil }

func (d *Display) GetSystemFont() *Font { return FontCocoa_new(d, nil) }

func (d *Display) GetTouchEnabled() bool { return false }

func (d *Display) SetCursor(c *Control) {}

func (d *Display) TimerExec(milliseconds int32, runnable any) {}

func (d *Display) CurrentTouches() *cocoa.NSMutableArray { return nil }

func (d *Display) FindTouchSource(touch *cocoa.NSTouch) any { return nil }

func (d *Display) CheckEnterExit(control *Control, nsEvent *cocoa.NSEvent, send bool) {}

func (d *Display) FindControl(checkTrim bool) *Control { return nil }

func (d *Display) ClearPool() {}

// Real Display.map has 4 overloads (Point/Rectangle x x,y/x,y,w,h) - split by return shape here.
func (d *Display) Map(from *Control, to *Control, x int32, y int32) *Point {
	panic("stub until translated: Display.map")
}

func (d *Display) MapRect(from *Control, to *Control, rect *Rectangle) *Rectangle {
	panic("stub until translated: Display.map(Rectangle)")
}

func DisplayIsValidClass(t reflect.Type) bool { return true }

// No real Thread/thread-affinity tracking (see Display.thread) - always the same value, so
// checkWidget()'s "!= currentThread()" check always passes instead of always panicking.
func ThreadCurrentThread() any { return nil }

// KCHR keyboard layouts are no longer resolvable without a real key-layout API; 0 means "use the
// unmodified-characters fallback", exactly the path real SWT also takes when this lookup fails.
func DisplayGetCurrentKeyLayout() int64 { return 0 }

// 0 is SWT's own "no key" value - safe default, same reasoning as DisplayGetCurrentKeyLayout.
func DisplayTranslateKey(keyCode int32) int32 { return 0 }

func DisplayIsActivateShellOnForceFocus() bool { return true }

type GC struct {
	Handle      *cocoa.NSGraphicsContext
	isDisposed_ bool
}

func GCCocoa_new(drawable any, data *GCData) *GC {
	panic("stub until translated: GC.cocoa_new")
}

func (g *GC) IsDisposed() bool { return g.isDisposed_ }

func (g *GC) Dispose() { g.isDisposed_ = true }

type Touch struct {
	Identity int64
	Source   any
	State    int32
	Primary  bool
	X, Y     int32
}

func NewTouch(identity int64, source any, state int32, primary bool, x int32, y int32) *Touch {
	return &Touch{Identity: identity, Source: source, State: state, Primary: primary, X: x, Y: y}
}

// TouchEvent.toString() calls touch.toString() on each element.
func (t Touch) ToString() string {
	return fmt.Sprintf("%+v", t)
}

// org.eclipse.swt.widgets.Dialog: only ever compared to nil in this file set (Control.isActive).
type Dialog struct{}

// org.eclipse.swt.graphics.AutoscalingMode: a Java enum (no translator rule for those yet, see
// README); Control.setAutoscalingMode's own body ignores its argument, so any type will do.
type AutoscalingMode int32
