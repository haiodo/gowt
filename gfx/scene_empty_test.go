package gfx

import (
	"math"
	"testing"
)

// A picture that draws nothing has empty bounds; mapping them through the transform gave NaN.
func TestEmptyPictureBounds(t *testing.T) {
	s := NewScene(R(0, 0, 100, 100), 32)
	n := NewNode(1)
	s.Update(func(tx *Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, NewRecorder().Finish())
	})
	s.Commit()
	if b := n.SubtreeBounds(); !b.Empty() || math.IsNaN(b.Min.X) {
		t.Errorf("bounds %v", b)
	}
}
