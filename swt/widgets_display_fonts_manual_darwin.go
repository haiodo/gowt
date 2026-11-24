package swt

import "github.com/haiodo/gowt/internal/cocoa"

func (this *Display) cachedWidgetFont(slot **cocoa.NSFont, class, selector int64) *cocoa.NSFont {
	if *slot == nil {
		*slot = this.GetFont(class, selector)
	}
	return *slot
}
