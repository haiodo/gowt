package gfx

import "testing"

func box(s *Scene, id int, r Rect) *Node {
	n := NewNode(id)
	rec := NewRecorder()
	rec.FillRect(r, Black)
	s.Update(func(tx *Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})
	return n
}

func TestSceneHasNoBounds(t *testing.T) {
	s := NewScene()
	huge := box(s, 2, R(-1e9, -1e9, 2e9, 2e9)) // a backdrop larger than any grid
	far := box(s, 1, R(5e6, -7e6, 10, 10))
	s.Commit()
	if got := s.HitTest(Point{X: 5e6 + 5, Y: -7e6 + 5}, 0); got != far {
		t.Errorf("hit far node: %v", got)
	}
	if got := s.HitTest(Point{X: 1, Y: 1}, 0); got != huge {
		t.Errorf("hit backdrop: %v", got)
	}
	if n := len(s.Query(R(-1e12, -1e12, 2e12, 2e12), nil)); n != 2 {
		t.Errorf("query of a huge area found %d nodes, want 2", n)
	}
}

func TestUnnamedNodesCount(t *testing.T) {
	s := NewScene()
	a, b := box(s, 0, R(0, 0, 5, 5)), box(s, 0, R(10, 0, 5, 5))
	s.Commit()
	if s.Len() != 2 || s.Node(0) != nil {
		t.Fatalf("len %d, node(0) %v", s.Len(), s.Node(0))
	}
	s.Update(func(tx *Tx) { tx.Remove(a) })
	s.Commit()
	if s.Len() != 1 || b.Parent() == nil {
		t.Errorf("len %d after removing one", s.Len())
	}
}

func TestColorHelpers(t *testing.T) {
	if c := RGB(255, 0, 51); c != (Color{1, 0, 0.2, 1}) {
		t.Errorf("RGB %v", c)
	}
	if c := White.WithAlpha(0.5); c.A != 0.5 || White.A != 1 {
		t.Errorf("WithAlpha %v, White %v", c, White)
	}
}

func TestRenderErrorsNameTheCause(t *testing.T) {
	s := NewScene()
	empty := NewNode(1) // no picture
	s.Update(func(tx *Tx) { tx.Add(nil, empty) })
	if _, err := RenderSubtree(&fakeRenderer{}, s, empty, Rect{}, 1, White); err == nil {
		t.Error("an empty subtree rendered")
	}
	full := box(s, 2, R(0, 0, 10, 10))
	if _, err := RenderSubtree(&fakeRenderer{}, s, full, Rect{}, 1e9, White); err == nil {
		t.Error("an absurd scale rendered")
	}
	if _, err := RenderSubtree(&fakeRenderer{}, s, nil, Rect{}, 1, White); err != nil {
		t.Errorf("nil root, the whole scene: %v", err)
	}
}

func TestLazyMeasurerOpensOnce(t *testing.T) {
	n := 0
	m := LazyMeasurer(func() (TextMeasurer, error) { n++; return EstimateText{}, nil })
	m.MeasureText("a", Font{Size: 10})
	m.MeasureText("b", Font{Size: 10})
	if n != 1 {
		t.Errorf("opened %d times", n)
	}
}
