package qatest_test

import (
	"encoding/json"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

type camera struct{ X, Y, Zoom float64 }

func cameraOf(t *testing.T, app *qatest.Client) camera {
	t.Helper()
	var m map[string]float64
	if err := json.Unmarshal(app.State("camera"), &m); err != nil {
		t.Fatal(err)
	}
	return camera{m["x"], m["y"], m["zoom"]}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// The gestures go through the view's router: the camera has moved when the call returns.
func TestGfxScrollAndPinch(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	v := app.Find(qa.ID("gfx"))

	v.ScrollEvent(qa.ScrollEvent{DX: -30, DY: 20, Precise: true, Phase: "changed"})
	if c := cameraOf(t, app); !near(c.X, 30) || !near(c.Y, -20) || c.Zoom != 1 {
		t.Fatalf("after scroll: %+v", c)
	}

	// The scene point under the pinch stays under it: (130, 80) at view point (100, 100).
	v.Pinch(qa.PinchEvent{X: 100, Y: 100, Magnification: 1, Phase: "changed"})
	if c := cameraOf(t, app); !near(c.Zoom, 2) || !near(c.X, 80) || !near(c.Y, 30) {
		t.Fatalf("after pinch: %+v", c)
	}

	v.ScrollEvent(qa.ScrollEvent{DY: 40, Precise: true, Mods: "Mod"})
	if c := cameraOf(t, app); !near(c.Zoom, 2*math.Exp(40*0.005)) {
		t.Fatalf("after Mod+scroll: %+v", c)
	}

	// A wheel without Precise moves by lines.
	app.State("camera-set", "x", 0, "y", 0)
	v.ScrollEvent(qa.ScrollEvent{DY: 1})
	if c := cameraOf(t, app); !near(c.Y, -16/c.Zoom) {
		t.Fatalf("after one wheel line: %+v", c)
	}

	wantErr(t, actErr(app, qa.ID("click"), "scrollevent", map[string]float64{"dy": 1}), "takes no scroll or pinch")
	wantErr(t, actErr(app, qa.ID("gfx"), "pinch", map[string]string{"phase": "sideways"}), "sideways")
	wantErr(t, actErr(app, qa.ID("gfx"), "scrollevent", map[string]string{"mods": "Hyper"}), "modifier")
}

// OnResize of the view's canvas reported the first size, once.
func TestGfxResize(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	var s struct {
		W, H    int
		Resizes [][2]int
	}
	app.Eventually(3*time.Second, func() (any, bool) {
		if err := json.Unmarshal(app.State("gfx-size"), &s); err != nil {
			t.Fatal(err)
		}
		return s, len(s.Resizes) > 0
	})
	if len(s.Resizes) != 1 || s.Resizes[0] != [2]int{s.W, s.H} || s.W < 300 || s.H < 200 {
		t.Errorf("view %dx%d, OnResize calls %v", s.W, s.H, s.Resizes)
	}
}

func redPixels(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, _ := img.At(x, y).RGBA(); r>>8 > 240 && g>>8 < 20 && bl>>8 < 20 {
				n++
			}
		}
	}
	return n
}

// A shot taken right after a camera change shows the new camera, and the first shot shows the tiles
// the view had no time to draw yet.
func TestGfxShotWaitsForFrame(t *testing.T) {
	guiOnly(t)
	app := qatest.App(t)
	if n := redPixels(t, app.Shot("gfx-before")); n < 5000 {
		t.Fatalf("the scene is not in the first shot: %d red pixels", n)
	}
	app.State("camera-set", "x", 5000, "y", 5000)
	if n := redPixels(t, app.Shot("gfx-after")); n > 0 {
		t.Errorf("the shot after the camera change still shows the old frame: %d red pixels", n)
	}
	app.State("camera-set", "x", 0, "y", 0)
	if n := redPixels(t, app.Shot("gfx-back")); n < 5000 {
		t.Errorf("the shot after the camera came back: %d red pixels", n)
	}
}

// A GOWT_SNAP shot of a gfx view has the scene in it, with the shortest delay the variable takes.
func TestGfxSnapHasScene(t *testing.T) {
	guiOnly(t)
	dir := t.TempDir()
	bin, out := os.Getenv("GOWT_QA_APP"), filepath.Join(dir, "snap.png")
	if bin == "" {
		bin = filepath.Join(dir, "app")
		if b, err := exec.Command("go", "build", "-o", bin, "./testdata/app").CombinedOutput(); err != nil {
			t.Fatalf("build: %v\n%s", err, b)
		}
	}
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "GOWT_SNAP="+out, "GOWT_SNAP_DELAY=1")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("snap run: %v\n%s", err, b)
	}
	if n := redPixels(t, out); n < 5000 {
		t.Errorf("the snapshot has %d red pixels, the scene is missing", n)
	}
}
