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
