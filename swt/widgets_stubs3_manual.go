// Round 6 manual stubs: classes Display references only off the Shell+Button path (dialogs,
// tray, dock menu, combo tracking, font metrics), the org.eclipse.swt.internal helpers whose
// static members Display/Shell/Item/SWT call, and the Callback bridge (see README "Callback design").
package swt

import (
	"fmt"
	"os"
	"reflect"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
)

// Callback: `new Callback(obj, "method", argCount)` is translated into NewCallbackFn with a Go
// closure calling the named method; the native half lives in internal/cocoa/callback_manual.go.
type Callback struct {
	address int64
}

func NewCallbackFn(fn func(args []int64) int64, argCount int32) *Callback {
	return &Callback{address: cocoa.NewCallbackN(int(argCount), fn)}
}

func (c *Callback) GetAddress() int64 { return c.address }

// purego callbacks can't be freed; the slot stays allocated (pool of 2000, SWT creates ~20).
func (c *Callback) Dispose() {}

// Callback.getEntryCount: JNI re-entry depth, only compared to 0 to decide on autorelease pools.
func CallbackGetEntryCount() int32 { return 0 }

func ThreadCurrentThread() any { return jrt.CurrentThread() }

// Display.isValidClass: SWT rejects subclasses outside its own package; the Go analogue is the
// swt package itself (getClass() on an SWT receiver is always one of its own types anyway).
func DisplayIsValidClass(clazz reflect.Type) bool {
	return clazz.Elem().PkgPath() == "github.com/haiodo/gowt/swt"
}

// org.eclipse.swt.internal.LONG: a boxed long used as a Map value.
type LONG struct {
	Value int64
}

func NewLONG(value int64) *LONG { return &LONG{Value: value} }

// Display.APPEARANCE, a nested Java enum.
type Display_APPEARANCE int32

const (
	Display_APPEARANCEDark Display_APPEARANCE = iota
	Display_APPEARANCELight
)

// DPIUtil: cocoa works in points, so the native zoom is 100 unless Display.setDeviceZoom says
// otherwise; swt.autoScale is not honoured (deviceZoom == nativeDeviceZoom).
var dpiNativeDeviceZoom int32 = 100

func DPIUtilSetDeviceZoom(nativeDeviceZoom int32) { dpiNativeDeviceZoom = nativeDeviceZoom }
func DPIUtilGetNativeDeviceZoom() int32           { return dpiNativeDeviceZoom }

// BidiUtil (emulated/bidi): no bidi support, text direction is left as is.
func BidiUtilResolveTextDirection(text string) int32 { return NONE }

func ImageUtilCreateImageRep(image *Image, size cocoa.NSSize) *cocoa.NSBitmapImageRep {
	panic("stub until translated: ImageUtil.createImageRep")
}

// Compatibility.getMessage: no SWTMessages resource bundle, the key itself is the message.
func CompatibilityGetMessage(key string, args ...[]any) string {
	if len(args) == 0 {
		return key
	}
	return fmt.Sprint(key, args[0])
}

// DefaultExceptionHandler: rethrow, as the Java lambdas do.
var DefaultExceptionHandlerRUNTIME_EXCEPTION_HANDLER = func(e error) { panic(e) }
var DefaultExceptionHandlerRUNTIME_ERROR_HANDLER = func(e error) {
	fmt.Fprintln(os.Stderr, e)
	panic(e)
}

type FontMetrics struct{}

type Combo struct{ Composite }

func (c *Combo) SendTrackingKeyEvent(nsEvent *cocoa.NSEvent, eventType int32) {}

type Tray struct {
	Widget
	itemCount int32
}

func NewTray(display *Display, style int32) *Tray { panic("stub until translated: Tray") }

type TrayItem struct{ Widget }

type TaskBar struct {
	Widget
	itemCount int32
}

func NewTaskBar(display *Display, style int32) *TaskBar { panic("stub until translated: TaskBar") }
func (t *TaskBar) GetItem(shell *Shell) *TaskItem       { return nil }

type TaskItem struct{ Widget }

func (t *TaskItem) GetMenu() *Menu { return nil }

type ColorDialog struct{}

func (d *ColorDialog) ChangeColor(id int64, sel int64, sender int64)     {}
func (d *ColorDialog) WindowWillClose(id int64, sel int64, sender int64) {}

type FontDialog struct{}

func (d *FontDialog) ChangeFont(id int64, sel int64, sender int64)      {}
func (d *FontDialog) WindowWillClose(id int64, sel int64, sender int64) {}
func (d *FontDialog) SetColor_forAttribute(id int64, sel int64, colorArg int64, attribute int64) {
}
func (d *FontDialog) ValidModesForFontPanel(id int64, sel int64, fontPanel int64) int64 { return 0 }

type FileDialog struct{}

func (d *FileDialog) SendSelection(id int64, sel int64, arg int64) {}
func (d *FileDialog) Panel_shouldEnableURL(id int64, sel int64, arg0 int64, arg1 int64) int64 {
	return 1
}
func (d *FileDialog) Panel_userEnteredFilename_confirmed(id int64, sel int64, sender int64, filename int64, okFlag int64) int64 {
	return filename
}
