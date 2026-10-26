package main

// repaintCards: on GTK a Canvas redraw or a label text change repaints the card background over the
// labels beside it (they show blank until the next full repaint). Toolkit bug, worked around here only.
func (m *monitor) repaintCards() {
	for _, c := range m.cards {
		c.Unwrap().RedrawChildren(true)
	}
}
