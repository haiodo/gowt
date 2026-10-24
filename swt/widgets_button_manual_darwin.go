// Button.setBounds and setFont set the push bezel on every layout (port.sh limits them to
// ownBezel), so a glass or circular bezel from look.GlassButton or gowt.Circle survives.

package swt

import "github.com/haiodo/gowt/internal/cocoa"

func (this *Button) ownBezel() bool {
	b := castcocoaNSViewTococoaNSButton(this.View).BezelStyle()
	return b == int64(cocoa.OSNSBezelStylePush) || b == int64(cocoa.OSNSBezelStyleFlexiblePush)
}

// registerGlassButtonClass registers SWTGlassButton, SWTButton without drawRect. On macOS 26 AppKit
// draws the glass bezel only for a class that does not override drawRect:; port.sh calls this from
// the class setup, and cocoa.SetGlassButton moves a button to it.
func (this *Display) registerGlassButtonClass(proc2, proc3, hitTestProc, needsDisplayInRectProc, setFrameOriginProc, setFrameSizeProc int64) {
	cls := cocoa.OSObjc_allocateClassPair(cocoa.OSClass_NSButton, "SWTGlassButton", int64(0))
	cocoa.OSClass_addIvar(cls, DisplaySWT_OBJECT, int64(DisplaySize), int8(DisplayAlign), DisplayTypes)
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_sendSelection, proc2, "@:")
	cocoa.OSClass_addMethod(cls, cocoa.OSSel_validateMenuItem_, proc3, "@:@")
	this.AddEventMethods(cls, proc2, proc3, 0, hitTestProc, needsDisplayInRectProc)
	this.AddFrameMethods(cls, setFrameOriginProc, setFrameSizeProc)
	cocoa.OSObjc_registerClassPair(cls)
}
