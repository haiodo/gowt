package pdf

import (
	"encoding/binary"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

func TestSubsetKeepsUsedGlyphsOnly(t *testing.T) {
	full, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var b sfnt.Buffer
	gA, _ := full.GlyphIndex(&b, 'A')
	gB, _ := full.GlyphIndex(&b, 'B')
	gC, _ := full.GlyphIndex(&b, 'C')

	// Go's fonts have no composite glyph: make C one that is built from A
	tables := readTables(goregular.TTF)
	glyf, head := tables["glyf"], append([]byte(nil), tables["head"]...)
	loca := make([]byte, 0, 4*(full.NumGlyphs()+1))
	old := tables["loca"]
	long := binary.BigEndian.Uint16(head[50:]) != 0
	for g := 0; g <= full.NumGlyphs(); g++ {
		if long {
			loca = binary.BigEndian.AppendUint32(loca, binary.BigEndian.Uint32(old[4*g:]))
		} else {
			loca = binary.BigEndian.AppendUint32(loca, 2*uint32(binary.BigEndian.Uint16(old[2*g:])))
		}
	}
	binary.BigEndian.PutUint16(head[50:], 1)
	composite := []byte{0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(gA >> 8), byte(gA), 0, 0}
	binary.BigEndian.PutUint32(loca[4*gC:], uint32(len(glyf)))
	glyf = append(append([]byte(nil), glyf...), composite...)
	for g := int(gC) + 1; g <= full.NumGlyphs(); g++ { // C ends where the appended glyph ends
		binary.BigEndian.PutUint32(loca[4*g:], uint32(len(glyf)))
	}
	tables["head"], tables["glyf"], tables["loca"] = head, glyf, loca
	input := buildTTF(tables)

	data := subsetTTF(input, []uint16{uint16(gC)})
	if data == nil || len(data) > len(goregular.TTF)/4 {
		t.Fatalf("subset is %d bytes of %d", len(data), len(goregular.TTF))
	}
	sub := readTables(data)["loca"] // the subset has no cmap, which sfnt insists on: read it by hand
	size := func(g sfnt.GlyphIndex) uint32 {
		return binary.BigEndian.Uint32(sub[4*g+4:]) - binary.BigEndian.Uint32(sub[4*g:])
	}
	if size(gC) < uint32(len(composite)) {
		t.Errorf("the used glyph is %d bytes, want %d", size(gC), len(composite))
	}
	if size(gA) == 0 {
		t.Error("the component of a composite glyph was dropped")
	}
	if size(gB) != 0 {
		t.Error("an unused glyph is still there")
	}
}
