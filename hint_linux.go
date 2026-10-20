package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// paintOver runs draw over the text: the Paint event of a GtkTextView, a container, comes
// before the view paints and is covered.
func paintOver(t *swt.Text, draw func(g *swt.GC)) {
	off := gtk.DrawOverTextView(t.Handle, func(cr int64) {
		data := swt.NewGCData()
		data.Cairo = cr
		g := swt.GCGtk_newDrawableData(t, data)
		draw(g)
		g.Dispose()
	})
	t.AddDisposeListener(disposeFunc(off))
}

func hintColor(t *swt.Text) *swt.Color {
	c := gtk.PlaceholderColor(t.Handle)
	return swt.NewColorRedGreenBlue(int32(c.Red*255+0.5), int32(c.Green*255+0.5), int32(c.Blue*255+0.5))
}

// hintOrigin is where GtkTextView puts the first character: with no text that is the caret.
func hintOrigin(t *swt.Text) (x, y int32) {
	p := t.GetCaretLocation()
	return p.X, p.Y
}
