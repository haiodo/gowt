package gfx

import (
	"sync"
	"testing"
)

func boxPic(w, h float64) *Picture {
	r := NewRecorder()
	var p Path
	p.Rect(0, 0, w, h)
	r.FillPath(&p, Color{A: 1})
	return r.Finish()
}

func rectShape(w, h float64) *Path {
	p := &Path{}
	p.Rect(0, 0, w, h)
	return p
}

// add puts a w x h box at (x, y) under parent and commits.
func add(s *Scene, parent *Node, id int, x, y, w, h float64) *Node {
	n := NewNode(id)
	s.Update(func(tx *Tx) {
		tx.Add(parent, n)
		tx.SetPicture(n, boxPic(w, h))
		tx.SetShape(n, rectShape(w, h), HitFill)
		tx.Move(n, x, y)
	})
	s.Commit()
	return n
}

func testScene() *Scene { return NewScene() }

func TestTransactionAppliedOnCommit(t *testing.T) {
	s := testScene()
	n := NewNode(1)
	s.Update(func(tx *Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, boxPic(10, 10))
		tx.Move(n, 100, 100)
	})
	if !s.Pending() || len(s.Query(R(0, 0, 500, 500), nil)) != 0 {
		t.Fatal("a transaction must not show before Commit")
	}
	d := s.Commit()
	if s.Pending() || len(d) != 1 || d[0] != R(100, 100, 10, 10) {
		t.Fatalf("damage %v", d)
	}
	if got := s.Query(R(0, 0, 500, 500), nil); len(got) != 1 || got[0] != n {
		t.Fatalf("query %v", got)
	}
	if s.Node(1) != n || s.Len() != 1 {
		t.Error("lookup by ID")
	}
}

func TestDamageIsOldAndNewBounds(t *testing.T) {
	s := testScene()
	a := add(s, nil, 1, 0, 0, 10, 10)
	add(s, nil, 2, 500, 500, 10, 10)
	s.Update(func(tx *Tx) { tx.Move(a, 200, 0) })
	d := s.Commit()
	if len(d) != 2 || d[0] != R(0, 0, 10, 10) || d[1] != R(200, 0, 10, 10) {
		t.Fatalf("damage %v: only the moved node's old and new bounds expected", d)
	}
	s.Update(func(tx *Tx) { tx.Move(a, 200, 0) })
	if d := s.Commit(); len(d) != 0 {
		t.Errorf("no-op move made damage %v", d)
	}
}

func TestFrameMovesSubtree(t *testing.T) {
	s := testScene()
	frame := add(s, nil, 1, 100, 100, 300, 300)
	kid := add(s, frame, 2, 10, 20, 30, 30)
	if b := kid.Bounds(); b != R(110, 120, 30, 30) {
		t.Fatalf("child bounds %v", b)
	}
	rev, _ := kid.Version()
	s.Update(func(tx *Tx) { tx.Move(frame, 150, 100) })
	d := s.Commit()
	if b := kid.Bounds(); b != R(160, 120, 30, 30) {
		t.Errorf("child bounds after the move %v", b)
	}
	if r2, _ := kid.Version(); r2 != rev {
		t.Error("a plain move must not change the child's version")
	}
	if len(d) != 4 {
		t.Errorf("damage %v: old and new bounds of frame and child", d)
	}
	// the frame's own picture was not touched
	if fr, _ := frame.Version(); fr != 0 && fr != 1 {
		t.Errorf("frame rev %d", fr)
	}
}

func TestRotatedParent(t *testing.T) {
	s := testScene()
	g := NewNode(1)
	k := NewNode(2)
	s.Update(func(tx *Tx) {
		tx.Add(nil, g)
		tx.SetTransform(g, TranslateBy(100, 0).Mul(RotateBy(3.141592653589793/2)))
		tx.Add(g, k)
		tx.SetPicture(k, boxPic(10, 20))
		tx.SetShape(k, rectShape(10, 20), HitFill)
	})
	s.Commit()
	// 90 degrees: local (x, y) -> (-y, x), so the 10x20 box covers x in [80, 100], y in [0, 10]
	b := k.Bounds()
	if !approx(b.Min.X, 80, 1e-6) || !approx(b.Max.X, 100, 1e-6) || !approx(b.Max.Y, 10, 1e-6) {
		t.Fatalf("bounds %v", b)
	}
	if s.HitTest(Point{90, 5}, 0) != k || s.HitTest(Point{110, 5}, 0) != nil {
		t.Error("hit test through a rotation")
	}
}

func TestClipHidesAndBlocksHits(t *testing.T) {
	s := testScene()
	frame := add(s, nil, 1, 0, 0, 100, 100)
	s.Update(func(tx *Tx) { tx.ClipRect(frame, R(0, 0, 100, 100)) })
	s.Commit()
	in := add(s, frame, 2, 80, 80, 60, 60) // sticks out to 140
	out := add(s, frame, 3, 200, 200, 20, 20)
	if b := in.Bounds(); b != R(80, 80, 20, 20) {
		t.Errorf("clipped bounds %v", b)
	}
	if !out.Bounds().Empty() {
		t.Errorf("a child fully outside the clip has bounds %v", out.Bounds())
	}
	if s.HitTest(Point{90, 90}, 0) != in {
		t.Error("hit inside the clip")
	}
	if h := s.HitTest(Point{120, 120}, 0); h != nil {
		t.Errorf("hit outside the clip got node %d", h.ID())
	}
	if h := s.HitTest(Point{210, 210}, 0); h != nil {
		t.Errorf("hit on a clipped away node got %d", h.ID())
	}
	var b Items
	s.Collect(R(0, 0, 400, 400), &b)
	if len(b.List) != 2 {
		t.Fatalf("%d items", len(b.List))
	}
	if c := b.List[1].Clips; len(c) != 1 || c[0].Kind != ClipRect {
		t.Errorf("item clips %+v", c)
	}
}

func TestRoundedAndPathClipHit(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 0, 0, 100, 100)
	s.Update(func(tx *Tx) { tx.ClipRRect(f, R(0, 0, 100, 100), 40) })
	s.Commit()
	k := add(s, f, 2, 0, 0, 100, 100)
	if s.HitTest(Point{2, 2}, 0) != nil {
		t.Error("the rounded corner must not hit")
	}
	if s.HitTest(Point{50, 50}, 0) != k {
		t.Error("the middle must hit")
	}
	tri := &Path{}
	tri.MoveTo(0, 0)
	tri.LineTo(100, 0)
	tri.LineTo(0, 100)
	tri.Close()
	s.Update(func(tx *Tx) { tx.ClipPath(f, tri) })
	s.Commit()
	if s.HitTest(Point{80, 80}, 0) != nil || s.HitTest(Point{20, 20}, 0) != k {
		t.Error("path clip hit")
	}
}

func TestZOrderAndHit(t *testing.T) {
	s := testScene()
	a := add(s, nil, 1, 0, 0, 100, 100)
	b := add(s, nil, 2, 50, 50, 100, 100)
	if s.HitTest(Point{75, 75}, 0) != b {
		t.Fatal("later sibling is on top")
	}
	s.Update(func(tx *Tx) { tx.SetZ(a, 5) })
	s.Commit()
	if s.HitTest(Point{75, 75}, 0) != a {
		t.Error("higher Z is on top")
	}
	q := s.Query(R(0, 0, 200, 200), nil)
	if len(q) != 2 || q[0] != b || q[1] != a {
		t.Errorf("paint order after SetZ: %v", q)
	}
	// children draw after the parent and before the next sibling
	c := add(s, b, 3, 0, 0, 10, 10)
	q = s.Query(R(0, 0, 200, 200), nil)
	if q[0] != b || q[1] != c || q[2] != a {
		t.Errorf("order with a child: %d %d %d", q[0].ID(), q[1].ID(), q[2].ID())
	}
}

func TestHitModes(t *testing.T) {
	s := testScene()
	line := NewNode(1)
	sh := &Path{}
	sh.MoveTo(0, 0)
	sh.LineTo(100, 0)
	s.Update(func(tx *Tx) {
		tx.Add(nil, line)
		tx.SetPicture(line, boxPic(100, 4))
		tx.SetShape(line, sh, HitStroke)
		tx.SetTransform(line, TranslateBy(0, 50).Mul(ScaleBy(2, 2)))
	})
	s.Commit()
	if s.HitTest(Point{100, 52}, 3) != line {
		t.Error("within the tolerance of the stroke")
	}
	if s.HitTest(Point{100, 60}, 3) != nil {
		t.Error("outside the tolerance")
	}
	s.Update(func(tx *Tx) { tx.SetShape(line, sh, HitNone) })
	s.Commit()
	if s.HitTest(Point{100, 52}, 3) != nil {
		t.Error("HitNone")
	}
}

func TestOpacityMultiplies(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 0, 0, 10, 10)
	k := add(s, f, 2, 0, 0, 10, 10)
	s.Update(func(tx *Tx) { tx.SetOpacity(f, 0.5); tx.SetOpacity(k, 0.5) })
	s.Commit()
	var b Items
	s.Collect(R(0, 0, 20, 20), &b)
	if len(b.List) != 2 || b.List[1].Alpha != 0.25 || b.List[0].Alpha != 0.5 {
		t.Errorf("alphas %+v", b.List)
	}
}

func TestLiftSkipsAndSilences(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 0, 0, 50, 50)
	k := add(s, f, 2, 10, 10, 10, 10)
	other := add(s, nil, 3, 300, 300, 10, 10)
	r := s.liftNode(f)
	if r != R(0, 0, 50, 50) {
		t.Errorf("lift area %v", r)
	}
	var b Items
	s.Collect(R(-100, -100, 1000, 1000), &b)
	if len(b.List) != 1 {
		t.Fatalf("%d items, only the node outside the lifted subtree expected", len(b.List))
	}
	s.Update(func(tx *Tx) { tx.Move(f, 100, 100); tx.Move(other, 310, 300) })
	d := s.Commit()
	if len(d) != 2 {
		t.Errorf("damage %v: moving a lifted node must not invalidate tiles", d)
	}
	if !k.Lifted() {
		t.Error("descendants are lifted")
	}
	if r := s.landNode(f); r != R(100, 100, 50, 50) {
		t.Errorf("land area %v", r)
	}
	s.CollectSubtree(f, R(-100, -100, 1000, 1000), &b)
	if len(b.List) != 2 {
		t.Errorf("subtree items %d", len(b.List))
	}
}

func TestRemoveAndReparent(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 100, 0, 50, 50)
	k := add(s, f, 2, 0, 0, 10, 10)
	g := add(s, nil, 3, 0, 100, 50, 50)
	s.Update(func(tx *Tx) { tx.Reparent(k, g) })
	d := s.Commit()
	if k.Bounds() != R(0, 100, 10, 10) || len(d) != 2 {
		t.Errorf("reparent: bounds %v damage %v", k.Bounds(), d)
	}
	s.Update(func(tx *Tx) { tx.Remove(g) })
	d = s.Commit()
	if s.Node(3) != nil || s.Node(2) != nil || len(s.Query(R(-100, -100, 1000, 1000), nil)) != 1 {
		t.Error("remove drops the subtree from the index")
	}
	if len(d) != 2 {
		t.Errorf("remove damage %v", d)
	}
	// into its own subtree is refused
	s.Update(func(tx *Tx) { tx.Reparent(f, f) })
	s.Commit()
	if f.Parent() != s.Root() {
		t.Error("reparent into itself")
	}
}

func TestUpdateFromManyGoroutines(t *testing.T) {
	s := testScene()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				n := NewNode(g*1000 + i)
				s.Update(func(tx *Tx) {
					tx.Add(nil, n)
					tx.SetPicture(n, boxPic(5, 5))
					tx.Move(n, float64(g*100), float64(i*10))
				})
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				s.HitTest(Point{50, 50}, 2)
				s.Query(R(0, 0, 300, 300), nil)
			}
		}
	}()
	for i := 0; i < 20; i++ {
		s.Commit()
	}
	wg.Wait()
	close(done)
	s.Commit()
	if s.Len() != 800 {
		t.Errorf("%d nodes, want 800", s.Len())
	}
}

func TestBuildIndexLarge(t *testing.T) {
	s := NewScene()
	s.Update(func(tx *Tx) {
		for i := 0; i < 5000; i++ {
			n := NewNode(i)
			tx.Add(nil, n)
			tx.SetPicture(n, boxPic(20, 20))
			tx.Move(n, float64(i%100)*100, float64(i/100)*100)
		}
	})
	s.Commit()
	got := s.Query(R(0, 0, 250, 250), nil)
	if len(got) != 9 {
		t.Errorf("%d nodes in the corner, want 9", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].order >= got[i].order {
			t.Fatal("query not in paint order")
		}
	}
}
