package shot

import (
	"time"

	"github.com/ebitengine/purego/objc"
)

// CaptureView writes the window of the popover whose content is view, as the window server has it:
// a view capture of an NSPopover window comes out black with a block of colour. The Frame is that
// window's, arrow included.
func CaptureView(view int64, path string) (Frame, error) {
	win := objc.ID(view).Send(objc.RegisterName("window"))
	win.Send(objc.RegisterName("displayIfNeeded"))
	objc.ID(objc.GetClass("CATransaction")).Send(objc.RegisterName("flush"))
	time.Sleep(300 * time.Millisecond)
	if err := screenPNG(objc.ID(view), path); err != nil {
		return Frame{}, err
	}
	return frameOf(win), nil
}
