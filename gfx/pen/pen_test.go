package pen

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

const libJSON = `{"version":"2.21","themes":{"mode":["light","dark"]},
"variables":{"ink":{"type":"color","value":[{"value":"#112233","theme":{"mode":"light"}},{"value":"#FFFFFF","theme":{"mode":"dark"}}]}},
"children":[{"type":"frame","id":"lib","name":"Library","children":[
 {"type":"frame","id":"btn","name":"Button","reusable":true,"layout":"horizontal","gap":4,"padding":[2,10],"fill":"#FF0000","cornerRadius":6,"alignItems":"center",
  "children":[{"type":"text","id":"lbl","content":"Save","fontFamily":"IBM Plex Sans","fontSize":10,"fill":"$ink"}]},
 {"type":"frame","id":"card","name":"Card","reusable":true,"layout":"vertical","width":200,"padding":8,"gap":6,"fill":"#FFFFFF",
  "children":[
   {"type":"text","id":"t1","content":"Title","fontFamily":"IBM Plex Sans","fontSize":10,"fill":"$ink"},
   {"type":"frame","id":"row","width":"fill_container","height":20,"fill":"#00FF00"},
   {"type":"frame","id":"slot","slot":[],"layout":"vertical","width":"fill_container"}]}
]}]}`

const mainJSON = `{"version":"2.21","imports":{"F":"lib.pen"},"children":[
 {"type":"ref","id":"s1","ref":"F:card","name":"Screen","x":100,"y":50,"width":600,"height":400,"descendants":{
   "F:t1":{"content":"Hello"},
   "F:slot":{"children":[{"type":"ref","id":"b1","ref":"F:btn","descendants":{"F:lbl":{"content":"Go"}}}]}}},
 {"type":"frame","id":"small","name":"Page","width":100,"height":40}]}`

func openTest(t *testing.T) *Doc {
	dir := t.TempDir()
	for name, body := range map[string]string{"lib.pen": libJSON, "main.pen": mainJSON} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := Open(filepath.Join(dir, "main.pen"), Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestBuildRefsAndLayout(t *testing.T) {
	d := openTest(t)
	if sc := d.Screens(); len(sc) != 1 || sc[0].W != 600 || sc[0].H != 400 {
		t.Fatalf("screens %+v", sc)
	}
	s := gfx.NewScene(gfx.R(0, 0, 1000, 1000), 128)
	var scr *Screen
	var err error
	s.Update(func(tx *gfx.Tx) { scr, err = d.Build(tx, nil, 0) })
	if err != nil {
		t.Fatal(err)
	}
	s.Commit()
	if scr.W != 600 || scr.H != 400 {
		t.Fatalf("size %v x %v", scr.W, scr.H)
	}
	if scr.Texts != 2 {
		t.Errorf("texts %d, want 2 (title and button label, both overridden)", scr.Texts)
	}
	// vertical card: padding 8, title 12 high (no line height: 1.2 em), gap 6, row 20 high at y 26
	var row *gfx.Node
	for _, n := range scr.Root.Children() {
		if n.Picture() != nil && n.Picture().Bounds.H() == 20 {
			row = n
		}
	}
	if row == nil {
		t.Fatal("row not found")
	}
	if w := row.World(); w[4] != 8 || w[5] != 26 {
		t.Errorf("row at %v, want (8, 26)", w[4:])
	}
	if b := row.Picture().Bounds; b.W() != 584 {
		t.Errorf("fill_container row is %v wide, want 584", b.W())
	}
}

func TestDarkMode(t *testing.T) {
	d := openTest(t)
	d.o.Mode = "dark"
	e := &expander{d: d}
	c, ok := e.color("$F:ink", "dark")
	if !ok || c.R != 1 {
		t.Errorf("dark ink %v", c)
	}
	c, _ = e.color("$ink", "light")
	if math.Abs(c.R-0x11/255.0) > 1e-9 {
		t.Errorf("light ink %v", c)
	}
}

func TestSVGPath(t *testing.T) {
	p, err := ParseSVGPath("M7 1a6 6 0 0 1 0.55 0.03m2.98 1.12l-1-1h2v3z")
	if err != nil {
		t.Fatal(err)
	}
	b := p.Bounds()
	if b.Min.X < 7 || b.Max.X > 12 || b.Min.Y < 0.9 {
		t.Errorf("bounds %v", b)
	}
	// a half circle of radius 5 from (0,0) to (10,0), sweeping through the lower side
	p, err = ParseSVGPath("M0 0A5 5 0 0 0 10 0")
	if err != nil {
		t.Fatal(err)
	}
	if b := p.Bounds(); math.Abs(b.Max.Y-5) > 0.05 || math.Abs(b.Min.Y) > 0.05 {
		t.Errorf("half circle bounds %v", b)
	}
	if _, err := ParseSVGPath("M1 x"); err == nil {
		t.Error("garbage accepted")
	}
}

// TestPrivateFile builds every screen of the file PEN_FILE names and logs the counts. It is skipped
// when the variable is not set: the design files are private and never part of the repository.
func TestPrivateFile(t *testing.T) {
	path := os.Getenv("PEN_FILE")
	if path == "" {
		t.Skip("PEN_FILE not set")
	}
	d, err := Open(path, Options{Measurer: gfx.EstimateText{}})
	if err != nil {
		t.Fatal(err)
	}
	s := gfx.NewScene(gfx.R(0, 0, 1e5, 1e5), 256)
	for _, in := range d.Screens() {
		var sc *Screen
		s.Update(func(tx *gfx.Tx) { sc, err = d.Build(tx, nil, in.Index) })
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%-40q %5.0f x %5.0f nodes %4d texts %4d paths %4d frames %4d source %4d", in.Name, sc.W, sc.H, sc.Nodes, sc.Texts, sc.Paths, sc.Frames, sc.Source)
	}
	s.Commit()
	t.Logf("skipped: %v", d.Skipped)
}
