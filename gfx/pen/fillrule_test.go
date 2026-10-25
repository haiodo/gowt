package pen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

const fillRuleJSON = `{"version":"2.21","children":[{"type":"frame","id":"f","name":"Rule","width":100,"height":50,"layout":"none","children":[
 {"type":"path","id":"eo","x":0,"y":0,"width":20,"height":20,"viewBox":[0,0,20,20],"fillRule":"evenodd","geometry":"M0 0H20V20H0ZM5 5H15V15H5Z","fill":"#000000"},
 {"type":"path","id":"nz","x":30,"y":0,"width":20,"height":20,"viewBox":[0,0,20,20],"geometry":"M0 0H20V20H0ZM5 5H15V15H5Z","fill":"#000000"}]}]}`

func TestPathFillRule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.pen")
	if err := os.WriteFile(path, []byte(fillRuleJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Open(path, Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene()
	var sc *Screen
	s.Update(func(tx *gfx.Tx) { sc, err = d.Build(tx, nil, 0) })
	if err != nil {
		t.Fatal(err)
	}
	s.Commit()
	var got []bool
	for _, n := range sc.Root.Children() {
		var it gfx.Iter
		it.Reset(n.Picture())
		for it.Next() {
			if it.Op().Kind == gfx.OpFill {
				got = append(got, it.Op().Path.EvenOdd)
			}
		}
	}
	if !reflect.DeepEqual(got, []bool{true, false}) {
		t.Errorf("even-odd flags %v, want [true false]", got)
	}
}

func TestWrapBreaksAfterHyphen(t *testing.T) {
	l := &layouter{
		d:   &Doc{o: Options{Measurer: gfx.EstimateText{}, FontName: func(string, int) string { return "x" }}},
		cnt: &Counters{},
	}
	n := &rn{content: "go push-notif x", fsize: 10}
	var got []string
	for _, ln := range l.wrap(n, 60) { // 6 per letter
		got = append(got, ln.s)
	}
	if want := []string{"go push-", "notif x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines %q, want %q", got, want)
	}
	if got := hyphenPieces("a-b--c-d-"); !reflect.DeepEqual(got, []string{"a-", "b--c-", "d-"}) {
		t.Errorf("pieces %q", got)
	}
}
