// Group.computeTrim measures the NSBox natively on every call (port.sh rewrites the call site to
// frameFromContentFrame); the frame-minus-content offset only depends on the title text and font.

package swt

import "github.com/haiodo/gowt/internal/cocoa"

type groupTrimKey struct {
	text string
	font string
	size float64
}

// Main thread only (cocoa Display), so no lock. Bounded: distinct (text, font) pairs; cleared wholesale past the limit.
var groupTrims = map[groupTrimKey]cocoa.NSRect{}

const groupTrimsMax = 256

func cachedFrameFromContent(key groupTrimKey, content cocoa.NSRect, measure func(cocoa.NSRect) cocoa.NSRect) cocoa.NSRect {
	d, ok := groupTrims[key]
	if !ok {
		frame := measure(content)
		d = cocoa.NSRect{X: frame.X - content.X, Y: frame.Y - content.Y, Width: frame.Width - content.Width, Height: frame.Height - content.Height}
		if len(groupTrims) >= groupTrimsMax {
			clear(groupTrims)
		}
		groupTrims[key] = d
		return frame
	}
	return cocoa.NSRect{X: content.X + d.X, Y: content.Y + d.Y, Width: content.Width + d.Width, Height: content.Height + d.Height}
}

func (this *Group) measureFrame(widget *cocoa.NSBox, c cocoa.NSRect) cocoa.NSRect {
	oldRect := widget.Frame()
	this.ignoreResize = true
	widget.SetFrameFromContentFrame(c)
	frame := widget.Frame()
	widget.SetFrame(oldRect)
	this.ignoreResize = false
	return frame
}

// groupResizing is the Group whose NSBox is inside setFrameSize. Main thread only.
var groupResizing *Group

// The NSBox tiles its content view at SetText while still 0x0, so it stays 2 title heights too tall
// until AppKit re-tiles the box before display; lay out again then, or children overflow the box.
func (this *Group) setFrameSize_(id int64, sel int64, size cocoa.NSSize) {
	if this.contentView == nil || id != this.contentView.Id {
		outer := groupResizing
		groupResizing = this
		defer func() { groupResizing = outer }()
		this.Composite.setFrameSize_(id, sel, size)
		return
	}
	old := this.contentView.Frame()
	this.Composite.setFrameSize_(id, sel, size)
	// A resize of the box itself relays the Group once, in resized.
	if groupResizing == this || this.ignoreResize || this.layout == nil || (old.Width == size.Width && old.Height == size.Height) {
		return
	}
	this.impl.markLayout_(false, false)
	this.impl.updateLayout_(false)
}

func (this *Group) frameFromContentFrame(widget *cocoa.NSBox, content cocoa.NSRect) cocoa.NSRect {
	key := groupTrimKey{text: this.text}
	// A native id can be reused after the font is released; the name cannot.
	if f := widget.TitleFont(); f != nil {
		if n := f.FontName(); n != nil {
			key.font = n.GetString()
		}
		key.size = f.PointSize()
	}
	return cachedFrameFromContent(key, content, func(c cocoa.NSRect) cocoa.NSRect { return this.measureFrame(widget, c) })
}
