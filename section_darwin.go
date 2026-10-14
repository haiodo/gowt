package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// installCard puts the grouped fill behind c and keeps it on c's frame and visibility; it goes with c.
func installCard(c *swt.Composite) {
	cocoa.InstallCard(c.View, sectionRadius, true)
	sync := eventFunc(func(e *swt.Event) {
		switch e.Type {
		case swt.Show, swt.Hide:
			cocoa.SetCardHidden(c.View, e.Type == swt.Hide)
		case swt.Dispose:
			cocoa.InstallCard(c.View, 0, false)
		default:
			cocoa.SyncCard(c.View)
		}
	})
	for _, t := range []int32{swt.Move, swt.Resize, swt.Show, swt.Hide, swt.Dispose} {
		c.AddListener(t, sync)
	}
}
