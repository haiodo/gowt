// Hand-written opaque stubs for widget/graphics types not translated yet (manual.txt). Display is
// real as of Round 6; GC/Touch/Dialog remain, the rest lives in swt/widgets_stubs2_manual.go.
package swt

import (
	"fmt"

	"github.com/haiodo/gowt/internal/cocoa"
)

type GC struct {
	Handle      *cocoa.NSGraphicsContext
	isDisposed_ bool
	data        *GCData
}

func GCCocoa_new(drawable any, data *GCData) *GC {
	panic("stub until translated: GC.cocoa_new")
}

func (g *GC) IsDisposed() bool { return g.isDisposed_ }

func (g *GC) GetGCData() *GCData { return g.data }

func (g *GC) FillRectangle(x int32, y int32, width int32, height int32) {}

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
