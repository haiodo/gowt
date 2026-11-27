package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/haiodo/gowt/qa"
	"github.com/haiodo/gowt/qa/qatest"
)

// rgb is one colour of a shot, device pixels of the screen.
type rgb struct{ r, g, b int }

func of(c color.Color) rgb {
	r, g, b, _ := c.RGBA()
	return rgb{int(r >> 8), int(g >> 8), int(b >> 8)}
}

func (c rgb) near(w rgb, tol int) bool {
	d := func(a, b int) bool { return a-b <= tol && b-a <= tol }
	return d(c.r, w.r) && d(c.g, w.g) && d(c.b, w.b)
}

var (
	card1   = rgb{40, 42, 52} // bgCard: the codex card and the settings cards
	card2   = rgb{30, 32, 41} // bgShell: the shell card
	border  = rgb{61, 63, 73} // cardBorder
	grouped = rgb{36, 36, 36} // the grouped colour Section paints without SectionFill
)

// cardRect is the box of the long runs of fill: a card row is a run hundreds of points wide, a blend
// of two colours is a few pixels of it, so the strays stay out.
func cardRect(img image.Image, fill rgb) image.Rectangle {
	b := img.Bounds()
	out := image.Rectangle{}
	add := func(x0, x1, y int) {
		if x1-x0 >= 40 {
			out = out.Union(image.Rect(x0, y, x1, y+1))
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		run := b.Min.X
		for x := b.Min.X; x < b.Max.X; x++ {
			if of(img.At(x, y)).near(fill, 1) {
				continue
			}
			add(run, x, y)
			run = x + 1
		}
		add(run, b.Max.X, y)
	}
	return out
}

// shotPNG decodes the shot qatest wrote.
func shotPNG(t *testing.T, path string) image.Image {
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
	return img
}

// windows lists the shells of the app as x, y, width, height in window points.
func windows(t *testing.T, app *qatest.Client) [][4]int {
	t.Helper()
	raw, err := app.Call("windows", nil)
	if err != nil {
		t.Fatal(err)
	}
	var ws []struct {
		Index   int
		Title   string
		Visible bool
		Bounds  [4]int
	}
	if err := json.Unmarshal(raw, &ws); err != nil {
		t.Fatal(err)
	}
	out := make([][4]int, len(ws))
	for i, w := range ws {
		out[i] = w.Bounds
	}
	return out
}

// scale is the device pixels per point of a shot of the window win.
func scale(img image.Image, win [4]int) float64 {
	return float64(img.Bounds().Dx()) / float64(win[2])
}

// TestCardBorderHugsEdge holds the card outline to the card's own edge: the one-point CardBorder
// stroke must run one point inside every side and follow the same inset through the corners.
func TestCardBorderHugsEdge(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	app := qatest.App(t)
	app.Settle()
	img := shotPNG(t, app.Shot("rex-cards"))
	for _, n := range app.Find(qa.ID("pane-card")).Nodes() {
		t.Logf("pane card %s %v", n.Class, n.Bounds)
	}
	s := scale(img, windows(t, app)[0])
	for _, c := range []struct {
		name string
		fill rgb
	}{
		{"codex", card1},
		{"shell", card2},
	} {
		card := cardRect(img, c.fill)
		if card.Empty() || card.Dx() < int(200*s) || card.Dy() < int(200*s) {
			t.Fatalf("%s: no card of %v on the shot, got %v", c.name, c.fill, card)
		}
		t.Logf("%s card %v, %.2f px/pt", c.name, card, s)
		// The stroke belongs on the card edge: from 1 point outside the fill to 2 points in.
		band := int(4.5 * s)
		out := int(2 * s)
		for _, side := range []struct {
			name   string
			x, y   int
			dx, dy int
		}{
			{"top", card.Min.X + card.Dx()/2, card.Min.Y, 0, 1},
			{"left", card.Min.X, card.Min.Y + card.Dy()/2, 1, 0},
			{"right", card.Max.X - 1, card.Min.Y + card.Dy()/2, -1, 0},
			{"bottom", card.Min.X + card.Dx()/2, card.Max.Y - 1, 0, -1},
		} {
			found := 99.0
			for i := -out; i < band; i++ {
				if of(img.At(side.x+side.dx*i, side.y+side.dy*i)).near(border, 3) {
					found = float64(i) / s
					break
				}
			}
			if found > 2 {
				t.Errorf("%s %s side: no CardBorder on the card edge, the nearest is %.1f points in", c.name, side.name, found)
				continue
			}
			t.Logf("%s %s side: border %.1f points from the fill edge", c.name, side.name, found)
		}
		for y := card.Min.Y + 1; y < card.Min.Y+int(8*s); y++ {
			fillX, borderX := -1, -1
			for x := card.Min.X; x < card.Min.X+int(20*s); x++ {
				p := of(img.At(x, y))
				if fillX < 0 && p.near(c.fill, 2) {
					fillX = x
				}
				if borderX < 0 && p.near(border, 3) {
					borderX = x
				}
			}
			if fillX < 0 || borderX < 0 {
				continue
			}
			if got := float64(borderX-fillX) / s; got > 2 {
				t.Errorf("%s corner %.1f points below the top: border %.1f points inside the card edge, want 1", c.name, float64(y-card.Min.Y)/s, got)
			}
		}
	}
}

// TestSettingsCardColour holds the settings cards to the colour the app gives them (Background of
// bgCard): the grouped grey Section paints instead is what shows behind the row labels.
func TestSettingsCardColour(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	app := qatest.App(t)
	app.Settle()
	app.Find(qa.ID("open-settings-sidebar")).Click()
	app.Settle()
	img := shotPNG(t, app.ShotOf("settings-general", qa.ID("settings-list")))
	ws := windows(t, app)
	s := scale(img, ws[len(ws)-1]) // the settings window is the shell opened last
	b := app.Find(qa.ID("client-info-card")).Node().Bounds
	t.Logf("client info card %v, %.2f px/pt", b, s)
	verified := 0
	for _, dy := range []float64{1.5, 3, 5} {
		x := int((float64(b[0]) + float64(b[2])/2) * s)
		y := int((float64(b[1]) + dy) * s)
		got := of(img.At(x, y))
		t.Logf("card top %.1f points in at %d,%d: (%d %d %d)", dy, x, y, got.r, got.g, got.b)
		if got.near(card1, 3) {
			verified++
		} else if got.near(grouped, 3) {
			t.Errorf("card background at %d,%d is the grouped grey (%d %d %d), want bgCard (%d %d %d)", x, y, got.r, got.g, got.b, card1.r, card1.g, card1.b)
		}
	}
	if verified != 3 {
		t.Errorf("%d of 3 points of the card background are bgCard", verified)
	}
}

// luminance is the perceived brightness of a shot pixel, 0 dark and 255 light.
func luminance(c rgb) int { return (299*c.r + 587*c.g + 114*c.b) / 1000 }

// strip is the box of the shot of a control of window win, its dumped bounds less the title strip the
// shot does not carry.
func strip(b [4]int, img image.Image, win [4]int) image.Rectangle {
	s := scale(img, win)
	dy := float64(win[3]) - float64(img.Bounds().Dy())/s
	return image.Rect(int(float64(b[0])*s), int((float64(b[1])-dy)*s),
		int(float64(b[0]+b[2])*s), int((float64(b[1]+b[3])-dy)*s))
}

// ink counts the pixels of r that are darker than dark, or lighter than light.
func ink(img image.Image, r image.Rectangle, darkerThan, lighterThan int) (dark, light int) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}
			switch l := luminance(of(img.At(x, y))); {
			case l < darkerThan:
				dark++
			case l > lighterThan:
				light++
			}
		}
	}
	return dark, light
}

// TestAppearanceSwitch checks Light and Dark through the app's appearance: the window
// background, the text and the icon of the session tab and a native field follow, and Light is not a
// dark window with dark ink left on it.
func TestAppearanceSwitch(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	app := qatest.App(t)
	app.Settle()
	app.Find(qa.ID("open-settings-sidebar")).Click()
	app.Settle()
	ws := windows(t, app)
	rex, settings := ws[0], ws[len(ws)-1]
	tab := app.Find(qa.ID("codex-tab")).Node().Bounds
	field := app.Find(qa.ID("settings-search")).Node().Bounds
	for _, c := range []struct {
		mode  string
		light bool
	}{
		{"light", true},
		{"dark", false},
	} {
		app.State("appearance", "mode", c.mode)
		app.Settle()
		rexImg := shotPNG(t, app.Shot("rex-"+c.mode))
		setImg := shotPNG(t, app.ShotOf("settings-"+c.mode, qa.ID("settings-list")))
		tabs := strip(tab, rexImg, rex)
		// The tab strip is filled with the card colour and holds the title and a tinted icon.
		bg := luminance(of(rexImg.At(tabs.Min.X+tabs.Dx()/2, tabs.Max.Y-3)))
		dark, light := ink(rexImg, tabs, 100, 160)
		t.Logf("%s: tab background %d, ink dark %d light %d", c.mode, bg, dark, light)
		if c.light {
			if bg < 180 {
				t.Errorf("light: tab background luminance %d, want a light card", bg)
			}
			if dark < 100 {
				t.Errorf("light: %d dark pixels of text and icon, want the dark ink", dark)
			}
		} else {
			if bg > 90 {
				t.Errorf("dark: tab background luminance %d, want a dark card", bg)
			}
			if light < 100 {
				t.Errorf("dark: %d light pixels of text and icon, want the light ink", light)
			}
		}
		// A native field: the search box of the settings sidebar follows the appearance by itself.
		f := strip(field, setImg, settings)
		fill := luminance(of(setImg.At(f.Max.X-8, f.Min.Y+f.Dy()/2)))
		t.Logf("%s: search field fill %d", c.mode, fill)
		if c.light && fill < 180 {
			t.Errorf("light: search field fill luminance %d, want a light field", fill)
		}
		if !c.light && fill > 90 {
			t.Errorf("dark: search field fill luminance %d, want a dark field", fill)
		}
	}
}
