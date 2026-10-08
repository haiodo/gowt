// TaskItem.updateImage (manual.txt): draws through a bitmap-backed NSGraphicsContext instead of the
// deprecated NSImage lockFocus/unlockFocus.
package swt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
)

func (this *TaskItem) UpdateImage() {
	var drawProgress bool = this.progress != 0 && this.progressState != DEFAULT
	var drawIntermidiate bool = this.progressState == INDETERMINATE
	var app *cocoa.NSApplication = cocoa.NSApplicationSharedApplication()
	var dock *cocoa.NSDockTile = app.DockTile()
	var drawImage bool = this.overlayImage != (nil) && dock.BadgeLabel() == (nil)
	if !drawImage && !drawProgress && !drawIntermidiate {
		app.SetApplicationIconImage(this.defaultImage)
		return
	}
	var size cocoa.NSSize = this.defaultImage.Size()
	var newImage *cocoa.NSImage = castcocoaNSObjectTococoaNSImage(cocoa.NewNSImage().Alloc())
	newImage = newImage.InitWithSize(size)
	var rep *cocoa.NSBitmapImageRep = castcocoaNSObjectTococoaNSBitmapImageRep(cocoa.NewNSBitmapImageRep().Alloc())
	rep = rep.InitWithBitmapDataPlanes(int64(0), int64(int32(size.Width)), int64(int32(size.Height)), int64(8), int64(4), true, false, cocoa.OSNSDeviceRGBColorSpace_, int64(cocoa.OSNSAlphaFirstBitmapFormat), int64(int32(size.Width)*4), int64(32))
	newImage.AddRepresentation(upcastcocoaNSBitmapImageRepTococoaNSImageRep(rep))
	var rect cocoa.NSRect = cocoa.NSRect{}
	rect.Height = size.Height
	rect.Width = size.Width
	cocoa.NSGraphicsContextStatic_saveGraphicsState()
	cocoa.NSGraphicsContextSetCurrentContext(cocoa.NSGraphicsContextGraphicsContextWithBitmapImageRep(rep))
	this.defaultImage.DrawInRect(rect, rect, int64(cocoa.OSNSCompositingOperationSourceOver), float64(1))
	if drawImage {
		var badgetImage *cocoa.NSImage = this.overlayImage.Handle
		var badgeSize cocoa.NSSize = badgetImage.Size()
		var srcRect cocoa.NSRect = cocoa.NSRect{}
		srcRect.Height = badgeSize.Height
		srcRect.Width = badgeSize.Width
		var dstRect cocoa.NSRect = cocoa.NSRect{}
		dstRect.X = size.Width / 2
		dstRect.Height = size.Height / 2
		dstRect.Width = size.Width / 2
		badgetImage.DrawInRect(dstRect, srcRect, int64(cocoa.OSNSCompositingOperationSourceOver), float64(1))
	}
	if drawIntermidiate || drawProgress {
		switch this.progressState {
		case ERROR:
			cocoa.NSColorColorWithDeviceRed(float64(1), float64(0), float64(0), float64(0.6)).SetFill()
			break
		case PAUSED:
			cocoa.NSColorColorWithDeviceRed(float64(1), float64(1), float64(0), float64(0.6)).SetFill()
			break
		default:
			cocoa.NSColorColorWithDeviceRed(float64(1), float64(1), float64(1), float64(0.6)).SetFill()
		}
		rect.Width = size.Width / float64((TaskItemPROGRESS_BARS*2 - 1))
		rect.Height = size.Height / 3
		var count int32
		if drawIntermidiate {
			count = this.iProgress
			this.iProgress = (this.iProgress + 1) % (TaskItemPROGRESS_BARS + 1)
			this.GetDisplay().TimerExec(TaskItemPROGRESS_TIMER, jrt.NewRunnable(func() {
				this.UpdateImage()
			}))
		} else {
			count = this.progress * TaskItemPROGRESS_BARS / TaskItemPROGRESS_MAX
		}
		for i := int32(0); i <= count; i++ {
			rect.X = float64(i) * float64(2) * rect.Width
			cocoa.NSBezierPathFillRect(rect)
		}
	}
	cocoa.NSGraphicsContextStatic_restoreGraphicsState()
	rep.Release()
	app.SetApplicationIconImage(newImage)
	newImage.Release()
}
