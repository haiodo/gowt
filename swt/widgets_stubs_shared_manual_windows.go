// Hand-written stand-ins shared by the platforms: classes Display references only off the Shell+Button
// path, and org.eclipse.swt.internal helpers whose static members Display/Shell/Item/SWT call.
package swt

import (
	"fmt"
	"os"
	"reflect"
)

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

// org.eclipse.swt.internal.WidgetSpy: creation/disposal tracking, off by default (matches the
// real class's own isEnabled starting false).
var WidgetSpyIsEnabled bool

type widgetSpy struct{}

func (widgetSpy) WidgetCreated(w *Widget)  {}
func (widgetSpy) WidgetDisposed(w *Widget) {}

func WidgetSpyGetInstance() widgetSpy { return widgetSpy{} }
