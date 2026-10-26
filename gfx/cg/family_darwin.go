//go:build darwin

package cg

import (
	"math"
	"sort"
	"sync"
	"unsafe"
)

const (
	kCFNumberFloat64Type = 6
	kCFNumberSInt32Type  = 3
	kCTFontUIFontSystem  = 2
	kCTFontItalicTrait   = 1 << 0
	// fallbackFamily stands in for a family nobody has; it is also the sans face of a Font that
	// asks for a weight but names no family.
	fallbackFamily = "Helvetica Neue"
)

// weightTrait pairs a CSS weight with the kCTFontWeightTrait value AppKit gives it (NSFontWeight).
var weightTrait = [...]struct{ css, trait float64 }{
	{100, -0.8}, {200, -0.6}, {300, -0.4}, {400, 0}, {500, 0.23}, {600, 0.3}, {700, 0.4}, {800, 0.56}, {900, 0.62},
}

// cssWeight converts a CoreText weight trait (-1..1) to a CSS weight by the points above.
func cssWeight(t float64) float64 {
	w := weightTrait[:]
	if t <= w[0].trait {
		return w[0].css
	}
	for i := 1; i < len(w); i++ {
		if t <= w[i].trait {
			a, b := w[i-1], w[i]
			return a.css + (b.css-a.css)*(t-a.trait)/(b.trait-a.trait)
		}
	}
	return w[len(w)-1].css
}

type famFace struct {
	desc   uintptr
	weight float64 // CSS scale
	italic bool
	width  float64 // kCTFontWidthTrait: 0 is normal, negative condensed
}

var (
	famMu    sync.Mutex
	famFaces = map[string][]famFace{}
)

// familyFaces lists the faces CoreText has under a family name, sorted by weight; nil when it has none.
func familyFaces(family string) []famFace {
	famMu.Lock()
	defer famMu.Unlock()
	if fs, ok := famFaces[family]; ok {
		return fs
	}
	name := cfString(family)
	keys := [1]uintptr{kFamilyAttr}
	vals := [1]uintptr{name}
	attrs := cfDictionaryCreate(0, &keys[0], &vals[0], 1, kKeyCB, kValCB)
	query := ctDescCreate(attrs)
	arr := ctDescMatching(query, 0)
	cfRelease(query)
	cfRelease(attrs)
	cfRelease(name)
	var fs []famFace
	if arr != 0 {
		for i := 0; i < cfArrayCount(arr); i++ {
			d := cfArrayAt(arr, i)
			tr := ctDescCopyAttr(d, kTraitsAttr)
			if tr == 0 {
				continue
			}
			var wt, wd float64
			var sym int32
			if n := cfDictGet(tr, kWeightTrait); n != 0 {
				cfNumberGet(n, kCFNumberFloat64Type, unsafe.Pointer(&wt))
			}
			if n := cfDictGet(tr, kSymbolicTrait); n != 0 {
				cfNumberGet(n, kCFNumberSInt32Type, unsafe.Pointer(&sym))
			}
			if n := cfDictGet(tr, kWidthTrait); n != 0 {
				cfNumberGet(n, kCFNumberFloat64Type, unsafe.Pointer(&wd))
			}
			cfRelease(tr)
			cfRetain(d)
			fs = append(fs, famFace{d, cssWeight(wt), uint32(sym)&kCTFontItalicTrait != 0, math.Abs(wd)})
		}
		cfRelease(arr)
	}
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].weight < fs[j].weight })
	famFaces[family] = fs
	return fs
}

// pickFace takes the face closest in weight among those of the asked slant, or among all when none
// has it; fs is sorted by weight, so the lighter face wins a tie.
func pickFace(fs []famFace, weight int, italic bool) famFace {
	strict := false
	for _, f := range fs {
		strict = strict || f.italic == italic
	}
	best, bd := fs[0], math.Inf(1)
	for _, f := range fs {
		if strict && f.italic != italic {
			continue
		}
		// a width off normal (condensed, expanded) only breaks a tie in weight
		if d := math.Abs(f.weight-float64(weight)) + f.width*0.01; d < bd {
			best, bd = f, d
		}
	}
	return best
}

// fontByFamily makes the CTFont of a family at a CSS weight; ok is false when CoreText lacks the family.
func fontByFamily(family string, weight int, italic bool, size float64) (uintptr, bool) {
	fs := familyFaces(family)
	if len(fs) == 0 {
		return 0, false
	}
	f := pickFace(fs, weight, italic)
	return ctFontFromDesc(f.desc, size, 0), true
}
