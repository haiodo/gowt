package pen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// fracText measures 0.55 em per rune, ascent 0.8 em, descent 0.25 em, and makes the IBMPlexSans
// family 10% wider, so that a test can tell it from a font that is missing.
type fracText struct{}

func (fracText) MeasureText(s string, f gfx.Font) gfx.TextMetrics {
	w := 0.55 * f.Size * float64(len([]rune(s)))
	if f.Family == "IBM Plex Sans" {
		w *= 1.1
	}
	return gfx.TextMetrics{Width: w, Ascent: 0.8 * f.Size, Descent: 0.25 * f.Size}
}

const fidLib = `{"version":"2.21","children":[{"type":"frame","id":"lib","name":"Library","children":[
 {"type":"frame","id":"dot","name":"Dot","reusable":true,"width":10,"height":10},
 {"type":"frame","id":"dot2","name":"Dot2","reusable":true,"width":30,"height":10,"children":[
   {"type":"frame","id":"inner","width":5,"height":5}]},
 {"type":"frame","id":"card","name":"Card","reusable":true,"layout":"vertical","width":200,"padding":8,"gap":4,"children":[
   {"type":"text","id":"t1","content":"Title","fontFamily":"Other","fontSize":14,"lineHeight":1.25},
   {"type":"text","id":"t2","content":"abc","fontFamily":"Other","fontSize":10},
   {"type":"frame","id":"off","enabled":false,"width":50,"height":50}]},
 {"type":"frame","id":"outer","name":"Outer","reusable":true,"layout":"vertical","children":[
   {"type":"ref","id":"r1","ref":"dot2","descendants":{"inner":{"id":"swap","type":"ref","ref":"dot","width":7,"height":7}}}]}
]}]}`

const fidMain = `{"version":"2.21","imports":{"F":"lib.pen"},"children":[
 {"type":"ref","id":"s1","ref":"F:card","x":0,"y":0,"descendants":{
   "F:t2":{"id":"nt2","type":"text","content":"abcd","fontFamily":"Other","fontSize":10}}},
 {"type":"ref","id":"s2","ref":"F:outer","x":0,"y":500}]}`

func fidBoxes(t *testing.T, id string) map[string]Box {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"lib.pen": fidLib, "main.pen": fidMain} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := Open(filepath.Join(dir, "main.pen"), Options{Measurer: fracText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene()
	var sc *Screen
	s.Update(func(tx *gfx.Tx) { sc, err = d.BuildNode(tx, nil, id) })
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Box{}
	for _, b := range sc.Boxes() {
		out[b.ID] = b
	}
	return out
}

func TestTextMetricsLikePenDev(t *testing.T) {
	b := fidBoxes(t, "s1")
	// 14 * 1.25 = 17.5 rounds to 18 per line box
	if h := b["s1/F:t1"].H; h != 18 {
		t.Errorf("title is %v high, want 18", h)
	}
	// "abcd" at 10 pt: 4 * 5.5 = 22, fits in whole units; no line height: round(8) + round(2.5)
	nt := b["nt2"]
	if nt.W != 22 || nt.H != 11 {
		t.Errorf("replaced text is %v x %v, want 22 x 11", nt.W, nt.H)
	}
}

func TestHugTextWidthRoundsUp(t *testing.T) {
	// 3 runes * 5.5 = 16.5 hug to 17; the replaced text above has the whole width 22
	if w := fidBoxes(t, "F:card")["F:t2"].W; w != 17 {
		t.Errorf("hug text is %v wide, want 17", w)
	}
}

func TestDisabledNodeIsLeftOut(t *testing.T) {
	b := fidBoxes(t, "s1")
	if _, ok := b["s1/F:off"]; ok {
		t.Error("disabled frame was built")
	}
	// the card is padding 8 + 18 + gap 4 + 11 + padding 8
	if h := b["s1"].H; h != 49 {
		t.Errorf("card is %v high, want 49", h)
	}
}

func TestReplacedNodeIds(t *testing.T) {
	b := fidBoxes(t, "s1")
	// a node replaced by an override on an instance at the top is addressed by its own id
	if _, ok := b["nt2"]; !ok {
		t.Errorf("ids %v", keys(b))
	}
	// inside a component the instance's path leads, and the ref without alias resolves in the library
	b = fidBoxes(t, "s2")
	sw, ok := b["s2/F:r1/F:swap"]
	if !ok || sw.W != 7 {
		t.Errorf("swapped node %+v in %v", sw, keys(b))
	}
	if _, ok := b["s2/F:r1/F:inner"]; ok {
		t.Error("the replaced node is still there")
	}
}

func TestFontFromNode(t *testing.T) {
	l := &layouter{}
	n := &rn{family: "IBM Plex Sans", weight: 600, italic: true, fsize: 12, ls: 0.5}
	want := gfx.Font{Family: "IBM Plex Sans", Weight: 600, Italic: true, Size: 12, Spacing: 0.5}
	if got := l.font(n); got != want {
		t.Errorf("font %+v, want %+v", got, want)
	}
}

func TestFontFallbackProbe(t *testing.T) {
	d := &Doc{o: Options{Measurer: fracText{}}}
	if d.FontFallback() {
		t.Error("a measurer that tells Plex apart reports a fallback")
	}
	d.o.Measurer = gfx.EstimateText{}
	if !d.FontFallback() {
		t.Error("a measurer that cannot tell Plex apart reports the font")
	}
}

func keys(m map[string]Box) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
