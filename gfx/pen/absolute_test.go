package pen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func buildBoxes(t *testing.T, body string) []Box {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.pen")
	if err := os.WriteFile(p, []byte(`{"version":"2.21","children":[`+body+`]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Open(p, Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene()
	var scr *Screen
	s.Update(func(tx *gfx.Tx) { scr, err = d.Build(tx, nil, 0) })
	if err != nil {
		t.Fatal(err)
	}
	return scr.Boxes()
}

func boxesByID(bs []Box) map[string]Box {
	out := map[string]Box{}
	for _, b := range bs {
		out[b.ID] = b
	}
	return out
}

func TestLayoutPositionAbsolute(t *testing.T) {
	list := buildBoxes(t, `{"type":"frame","id":"p","layout":"vertical","gap":7,"padding":5,"children":[
 {"type":"frame","id":"a","width":40,"height":10,"fill":"#111111"},
 {"type":"frame","id":"ab","layoutPosition":"absolute","width":300,"height":300,"x":50,"y":60,"fill":"#222222"},
 {"type":"frame","id":"c","width":40,"height":10,"fill":"#333333"}]}`)
	got := boxesByID(list)
	ref := boxesByID(buildBoxes(t, `{"type":"frame","id":"p","layout":"vertical","gap":7,"padding":5,"children":[
 {"type":"frame","id":"a","width":40,"height":10,"fill":"#111111"},
 {"type":"frame","id":"c","width":40,"height":10,"fill":"#333333"}]}`))
	if b := got["ab"]; b.X != 50 || b.Y != 60 || b.W != 300 || b.H != 300 {
		t.Fatalf("absolute box %+v", b)
	}
	for _, id := range []string{"p", "a", "c"} {
		if g, r := got[id], ref[id]; g.X != r.X || g.Y != r.Y || g.W != r.W || g.H != r.H {
			t.Errorf("%s: %+v, without the absolute child %+v", id, g, r)
		}
	}
	// z-order stays the order of the file
	if len(list) != 4 || list[1].ID != "a" || list[2].ID != "ab" || list[3].ID != "c" {
		t.Fatalf("order %+v", list)
	}
}

func TestLayoutPositionAbsoluteInHugCell(t *testing.T) {
	got := boxesByID(buildBoxes(t, `{"type":"frame","id":"s","layout":"vertical","children":[
 {"type":"frame","id":"cell","layout":"vertical","padding":4,"children":[
  {"type":"frame","id":"k","width":30,"height":20,"fill":"#111111"},
  {"type":"frame","id":"ring","layoutPosition":"absolute","x":10,"y":300,"width":60,"height":60,"stroke":{"thickness":1,"fill":"#FF0000"}}]}]}`))
	if c := got["cell"]; c.W != 38 || c.H != 28 {
		t.Fatalf("cell %+v", c)
	}
	if s := got["s"]; s.H != 28 {
		t.Fatalf("screen %+v", s)
	}
	if r := got["ring"]; r.X != 10 || r.Y != 300 {
		t.Fatalf("ring %+v", r)
	}
}
