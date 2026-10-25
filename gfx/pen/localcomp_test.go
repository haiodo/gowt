package pen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

const localCompJSON = `{"version":"2.21","children":[
 {"type":"frame","id":"comp","name":"Comp","reusable":true,"width":100,"layout":"vertical","children":[
  {"type":"text","id":"t","name":"t","content":"x","fontFamily":"IBM Plex Sans","fontSize":10,"fill":"#000000"}]},
 {"type":"frame","id":"f","name":"Screen","width":200,"height":100,"layout":"none","children":[
  {"type":"ref","id":"i","ref":"comp","x":0,"y":0,"descendants":{"t":{"content":"yyyyyyyyyy"}}}]}]}`

// pen.dev rounds the sum of ascent and descent, not each of them: 12 px Plex gives 16, not 15.
func TestLineHeightRoundsSum(t *testing.T) {
	l := &layouter{
		d:   &Doc{o: Options{Measurer: gfx.EstimateText{}, FontName: func(string, int) string { return "x" }}},
		cnt: &Counters{},
	}
	// EstimateText: ascent 0.8 em, descent 0.25 em, 6 px gives 4.8 + 1.5
	if h := l.lineH(&rn{fsize: 6}); h != 6 {
		t.Errorf("line height %v, want 6", h)
	}
}

// A component of the file itself is addressed by the instance path plus the bare id, and its
// descendants are overridden by the bare id.
func TestLocalComponentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "l.pen")
	if err := os.WriteFile(path, []byte(localCompJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Open(path, Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene()
	var sc *Screen
	s.Update(func(tx *gfx.Tx) { sc, err = d.BuildNode(tx, nil, "f") })
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range sc.Boxes() {
		if b.ID == "i/t" {
			found = true
			if b.W != 60 {
				t.Errorf("overridden text width %v, want 60", b.W)
			}
		}
	}
	if !found {
		t.Error("no node i/t")
	}
}
