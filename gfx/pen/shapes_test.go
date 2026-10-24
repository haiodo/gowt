package pen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

const shapesJSON = `{"version":"2.21","children":[{"type":"frame","id":"f","name":"Shapes","width":800,"height":500,"layout":"none",
 "fill":{"type":"gradient","gradientType":"linear","enabled":true,"rotation":180,"colors":[{"color":"#FFFFFF","position":0},{"color":"#CCCCCC","position":1}]},
 "children":[
  {"type":"ellipse","id":"e","x":10,"y":10,"width":40,"height":40,"fill":"#FF0000","effect":{"type":"shadow","shadowType":"outer","color":"#00000040","offset":{"x":0,"y":2},"blur":4}},
  {"type":"path","id":"p","x":60,"y":10,"width":20,"height":20,"viewBox":[0,0,10,10],"geometry":"M0 0L10 10","stroke":"#000000","strokeWidth":1},
  {"type":"frame","id":"c","x":100,"y":10,"width":50,"height":50,"cornerRadius":[8,0,0,8],"fill":"#00FF00","stroke":"#000000","strokeWidth":2,"strokeAlignment":"inner"},
  {"type":"ref","id":"r","ref":"missing","x":0,"y":0},
  {"type":"text","id":"t","x":10,"y":100,"content":"a b c d e f g h","textGrowth":"fixed-width","width":20,"fontFamily":"Other","fontSize":10,"fill":"#000000"},
  {"type":"frame","id":"off","enabled":false,"x":0,"y":0,"width":5,"height":5,"fill":"#000000"}]}]}`

func TestShapesAndUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.pen")
	if err := os.WriteFile(path, []byte(shapesJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Open(path, Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene(gfx.R(0, 0, 1000, 1000), 128)
	var sc *Screen
	s.Update(func(tx *gfx.Tx) { sc, err = d.Build(tx, nil, 0) })
	if err != nil {
		t.Fatal(err)
	}
	s.Commit()
	// root, ellipse, path, corner frame, text; the disabled frame and the missing ref are left out
	if sc.Nodes != 5 || sc.Paths != 1 || sc.Texts != 1 {
		t.Errorf("nodes %d paths %d texts %d", sc.Nodes, sc.Paths, sc.Texts)
	}
	if u := d.Unsupported(); len(u) != 1 {
		t.Errorf("unsupported %v", u)
	}
	// at 6 pt per letter two words fit in 20 pt: 4 lines of 12 pt
	var tn *gfx.Node
	for _, n := range sc.Root.Children() {
		if n.Picture() != nil && n.Picture().Bounds.H() > 40 {
			tn = n
		}
	}
	if tn == nil {
		t.Error("wrapped text not found")
	}
	if n := s.HitTest(gfx.Point{X: 30, Y: 30}, 1); n == nil || n.Picture() == nil {
		t.Error("the ellipse is not hit")
	}
}
