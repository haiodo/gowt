// Round 6 manual stubs: classes Display references only off the Shell+Button path (dialogs,
// tray, dock menu, combo tracking, font metrics), the org.eclipse.swt.internal helpers whose
// static members Display/Shell/Item/SWT call, and the Callback bridge (see README "Callback design").
package swt

import (
	"fmt"
	"os"
	"reflect"

	"github.com/haiodo/gowt/internal/cocoa"
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
