package swt

import "github.com/haiodo/gowt/internal/cocoa"

// caretFill: white in difference mode inverts the pixels under the caret, but with a background
// alpha below 255 they may be transparent and the caret stays white, so it takes the text colour.
func (this *Canvas) caretFill(ctx int64) []float64 {
	if this.backgroundAlpha == 255 {
		return []float64{1, 1, 1, 1}
	}
	cocoa.OSCGContextSetBlendMode(ctx, cocoa.OSKCGBlendModeNormal)
	return this.impl.getForeground_().Handle
}
