package gfx

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func wellFormed(t *testing.T, s string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		if _, err := d.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Fatalf("not well-formed XML: %v\n%s", err, s)
		}
	}
}

func TestPictureSVG(t *testing.T) {
	r := NewRecorder()
	var p Path
	p.RoundRect(0, 0, 40, 20, 4)
	r.Save()
	r.Translate(5, 5)
	r.ClipRRect(R(0, 0, 40, 20), 4)
	r.FillGradient(&p, NewLinearGradient(Point{}, Point{40, 0}, Stop{0, Color{1, 0, 0, 1}}, Stop{1, Color{0, 0, 1, 1}}))
	r.Restore()
	r.BeginLayer(0.5)
	r.StrokePath(&p, StrokeStyle{Width: 2, Dash: []float64{3, 2}, Cap: CapRound}, Color{0, 0, 0, 0.5})
	r.FillShadowRRect(0, 0, 40, 20, 4, Color{1, 1, 1, 1}, Shadow{DY: 2, Blur: 6, Color: Color{A: 0.4}})
	r.EndLayer()
	r.Text(`a<b & "c"`, 2, 12, Font{Name: "Helvetica Neue", Size: 10}, Color{A: 1})
	img := NewRaster(2, 2)
	img.Pix[3] = 255
	r.DrawImage(img, 0, 0, 8, 8)
	pic := r.Finish()
	var b bytes.Buffer
	if err := pic.WriteSVG(&b, 2); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	wellFormed(t, s)
	for _, want := range []string{`<svg `, `width="`, `<linearGradient`, `clip-path="url(#c`, `stroke-dasharray="3 2"`, `stroke-linecap="round"`,
		`opacity="0.5"`, `<filter`, `a&lt;b &amp; &#34;c&#34;`, `data:image/png;base64,`, `translate(5 5)`} {
		if !strings.Contains(s, want) {
			t.Errorf("SVG lacks %q:\n%s", want, s)
		}
	}
	// every group opened is closed
	if o, c := strings.Count(s, "<g"), strings.Count(s, "</g>"); o != c {
		t.Errorf("%d <g> and %d </g>", o, c)
	}
}

func TestSVGFont(t *testing.T) {
	r := NewRecorder()
	r.Text("a", 0, 10, Font{Family: "IBM Plex Sans", Weight: 600, Italic: true, Size: 12, Spacing: 0.5}, Color{A: 1})
	r.Text("b", 0, 20, Font{Name: "IBMPlexSans-Bold", Family: "X", Weight: 300, Size: 12}, Color{A: 1})
	var b bytes.Buffer
	if err := r.Finish().WriteSVG(&b, 1); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{
		`font-family="IBM Plex Sans" font-weight="600" font-style="italic" font-size="12" letter-spacing="0.5"`,
		`font-family="IBMPlexSans-Bold" font-size="12"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("SVG lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, `font-family="X"`) || strings.Contains(s, `font-weight="300"`) {
		t.Errorf("a Name must win over Family and Weight:\n%s", s)
	}
}

func TestClipRRect4(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 0, 0, 50, 50)
	s.Update(func(tx *Tx) { tx.ClipRRect4(f, R(0, 0, 30, 30), [4]float64{0, 5, 10, 15}) })
	s.Commit()
	add(s, f, 2, 10, 10, 40, 40)
	var b bytes.Buffer
	if err := WriteSubtreeSVG(&b, s, f, Rect{}, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "<clipPath") {
		t.Errorf("no clip in:\n%s", b.String())
	}
}

func TestSubtreeSVG(t *testing.T) {
	s := testScene()
	f := add(s, nil, 1, 100, 100, 50, 50)
	s.Update(func(tx *Tx) { tx.ClipRect(f, R(0, 0, 30, 30)); tx.SetOpacity(f, 0.5) })
	s.Commit()
	add(s, f, 2, 10, 10, 40, 40)
	add(s, nil, 3, 500, 500, 5, 5)
	var b bytes.Buffer
	if err := WriteSubtreeSVG(&b, s, f, Rect{}, 1); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	wellFormed(t, out)
	if strings.Count(out, "<path") != 2 || !strings.Contains(out, "<clipPath") || !strings.Contains(out, `opacity="0.5"`) {
		t.Errorf("subtree SVG:\n%s", out)
	}
	if !strings.Contains(out, `viewBox="100 100 `) {
		t.Errorf("viewBox: %s", out)
	}
}

func TestSubtreeSVGCSS(t *testing.T) {
	s := testScene()
	add(s, nil, 1, 0, 0, 50, 50)
	css := `@keyframes spin{to{transform:rotate(360deg)}}#spin{animation:spin 1s}`
	var b bytes.Buffer
	if err := WriteSubtreeSVGWith(&b, s, nil, Rect{}, 1, SVGOptions{CSS: css}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	wellFormed(t, out)
	if !strings.Contains(out, "<defs><style>"+css+"</style>") {
		t.Errorf("CSS not in <defs>:\n%s", out)
	}
}

func TestSubtreeSVGNodeID(t *testing.T) {
	s := testScene()
	add(s, nil, 1, 0, 0, 50, 50)
	n := add(s, nil, 2, 100, 0, 10, 10)
	s.Update(func(tx *Tx) { tx.SetName(n, `a<"&`) })
	s.Commit()
	var b bytes.Buffer
	if err := WriteSubtreeSVG(&b, s, nil, Rect{}, 1); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	wellFormed(t, out)
	if strings.Contains(out, `id="n`) {
		t.Errorf("an unnamed group must carry no id:\n%s", out)
	}
	if !strings.Contains(out, `id="a&lt;&#34;&amp;"`) {
		t.Errorf("name not escaped:\n%s", out)
	}
}

// The group id of an item follows its node, not its position: a CSS rule that addresses the id
// applies to the same picture whatever order the groups come out in.
func TestSubtreeSVGIDOrderIndependent(t *testing.T) {
	s := testScene()
	a := add(s, nil, 1, 0, 0, 10, 10)
	b := add(s, nil, 2, 50, 0, 20, 20)
	s.Update(func(tx *Tx) { tx.SetName(a, "aa"); tx.SetName(b, "bb") })
	s.Commit()
	write := func() string {
		var buf bytes.Buffer
		if err := WriteSubtreeSVGWith(&buf, s, nil, Rect{}, 1, SVGOptions{CSS: "#aa{animation:x}"}); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	before := write()
	s.Update(func(tx *Tx) { tx.SetZ(a, 1) }) // bb first now
	s.Commit()
	swapped := write()
	if !(strings.Index(swapped, `id="bb"`) < strings.Index(swapped, `id="aa"`)) {
		t.Errorf("z did not reorder groups:\n%s", swapped)
	}
	groupOf := func(doc, id string) string {
		i := strings.Index(doc, `id="`+id+`"`)
		if i < 0 {
			t.Fatalf("no group %s in:\n%s", id, doc)
		}
		j := strings.Index(doc[i:], "</g>")
		return doc[i : i+j]
	}
	if groupOf(before, "aa") != groupOf(swapped, "aa") || groupOf(before, "bb") != groupOf(swapped, "bb") {
		t.Errorf("an id moved to another node's group:\n%s", swapped)
	}
	if !strings.Contains(swapped, "<style>#aa{animation:x}</style>") {
		t.Errorf("CSS missing after reorder:\n%s", swapped)
	}
}
