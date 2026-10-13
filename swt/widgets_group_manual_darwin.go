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
