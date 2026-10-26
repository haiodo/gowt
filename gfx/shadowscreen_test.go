package gfx

import "testing"

func TestShadowScreenFlag(t *testing.T) {
	r := NewRecorder()
	r.FillShadowPath(RectPath(R(0, 0, 5, 5)), Color{A: 1}, Shadow{DX: 1, Screen: true})
	r.FillShadowRRect(0, 0, 5, 5, 1, Color{A: 1}, Shadow{DX: 1, Screen: true})
	r.FillShadowRRect(0, 0, 5, 5, 1, Color{A: 1}, Shadow{DX: 1})
	var it Iter
	it.Reset(r.Finish())
	var got []bool
	for it.Next() {
		got = append(got, it.Op().Shadow.Screen)
	}
	if len(got) != 3 || !got[0] || !got[1] || got[2] {
		t.Errorf("screen flags %v", got)
	}
}
