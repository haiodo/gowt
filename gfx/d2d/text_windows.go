//go:build windows

package d2d

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

// DirectWrite vtable slots (IUnknown takes 0..2).
const (
	dwFactoryGetSystemFontCollection = 3
	dwFactoryCreateFontFileReference = 7
	dwFactoryCreateTextFormat        = 15
	dwFactoryCreateTextLayout        = 18
	dwFactoryCreateFontCollFromSet   = 37 // IDWriteFactory3
	dwFactoryCreateFontSetBuilder1   = 43 // IDWriteFactory5

	dwCollectionFindFamilyName = 5

	dwBuilderCreateFontSet = 6
	dwBuilderAddFontFile   = 7 // IDWriteFontSetBuilder1

	dwFormatSetWordWrapping = 5

	dwLayoutGetLineMetrics = 59
	dwLayoutGetMetrics     = 60
)

type dwTextMetrics struct {
	left, top, width, widthTrailing, height, layoutW, layoutH float32
	bidiDepth, lineCount                                      uint32
}

type dwLineMetrics struct {
	length, trailing, newline uint32
	height, baseline          float32
	trimmed                   int32
}

type textEngine struct {
	factory uintptr
	sys     uintptr // system font collection

	mu     sync.RWMutex
	custom uintptr             // collection built from the registered files, 0 if none
	faces  map[string]FaceInfo // by lowercase PostScript name, from fontReg
	gen    int                 // the fontReg generation the custom collection was built from

	resolved sync.Map // string -> Resolved (per gen, cleared on change)
	formats  sync.Map // fmtKey -> uintptr
	layouts  sync.Map // layKey -> *textLayout
}

type fmtKey struct {
	name string
	size float64
}

type layKey struct {
	name string
	size float64
	s    string
}

type textLayout struct {
	ptr uintptr
	gfx.TextMetrics
}

func newTextEngine() (*textEngine, error) {
	var fac uintptr
	hr, _, _ := procDWF.Call(0, uintptr(unsafe.Pointer(iidDWriteFactory5)), uintptr(unsafe.Pointer(&fac)))
	if failed(hr) {
		hr, _, _ = procDWF.Call(0, uintptr(unsafe.Pointer(iidDWriteFactory)), uintptr(unsafe.Pointer(&fac)))
		if failed(hr) {
			return nil, hrErr("DWriteCreateFactory", hr)
		}
	}
	te := &textEngine{factory: fac}
	if hr := vcall(fac, dwFactoryGetSystemFontCollection, ref(&te.sys), ui(0)); failed(hr) {
		return nil, hrErr("GetSystemFontCollection", hr)
	}
	return te, nil
}

func (te *textEngine) close() {
	te.formats.Range(func(_, v any) bool { release(v.(uintptr)); return true })
	te.layouts.Range(func(_, v any) bool { release(v.(*textLayout).ptr); return true })
	release(te.custom)
	release(te.sys)
	release(te.factory)
}

func (te *textEngine) hasFamily(coll uintptr, family string) bool {
	var idx uint32
	var exists int32
	w := utf16Z(family)
	vcall(coll, dwCollectionFindFamilyName, ref(w), ref(&idx), ref(&exists))
	return exists != 0
}

// The registered fonts are process-wide: every Renderer builds its custom collection from them.
var fontReg struct {
	mu    sync.RWMutex
	gen   int
	files map[string]bool
	faces map[string]FaceInfo // by lowercase PostScript name
}

// RegisterFonts makes the font files of dir (or the one file) usable by name by every Renderer of
// this process and returns how many were new. Call it before text is measured or drawn: lookups
// made earlier keep their fallback.
func RegisterFonts(dir string) (int, error) {
	var files []string
	st, err := os.Stat(dir)
	if err != nil {
		return 0, err
	}
	if st.IsDir() {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		for _, e := range ents {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	} else {
		files = []string{dir}
	}
	fontReg.mu.Lock()
	defer fontReg.mu.Unlock()
	if fontReg.files == nil {
		fontReg.files, fontReg.faces = map[string]bool{}, map[string]FaceInfo{}
	}
	n := 0
	for _, f := range files {
		switch strings.ToLower(filepath.Ext(f)) {
		case ".otf", ".ttf", ".ttc":
		default:
			continue
		}
		abs, err := filepath.Abs(f)
		if err != nil || fontReg.files[abs] {
			continue
		}
		fi, err := ReadFaceInfo(abs)
		if err != nil {
			continue
		}
		fontReg.files[abs] = true
		fontReg.faces[strings.ToLower(fi.PostScript)] = fi
		n++
	}
	if n > 0 {
		fontReg.gen++
	}
	return n, nil
}

// RegisterFonts is the package function, as a method for callers that hold a Renderer.
func (r *Renderer) RegisterFonts(dir string) (int, error) { return RegisterFonts(dir) }

// sync rebuilds the custom collection when fonts were registered since the last lookup.
func (te *textEngine) sync() {
	fontReg.mu.RLock()
	gen := fontReg.gen
	fontReg.mu.RUnlock()
	te.mu.RLock()
	same := gen == te.gen
	te.mu.RUnlock()
	if same {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()
	fontReg.mu.RLock()
	defer fontReg.mu.RUnlock()
	if fontReg.gen == te.gen {
		return
	}
	te.gen = fontReg.gen
	te.faces = fontReg.faces
	if err := te.rebuild(fontReg.files); err != nil {
		te.custom = 0 // the system fonts stay
	}
}

// rebuild makes the custom collection from files; mu is held.
func (te *textEngine) rebuild(files map[string]bool) error {
	var b uintptr
	if hr := vcall(te.factory, dwFactoryCreateFontSetBuilder1, ref(&b)); failed(hr) {
		return fmt.Errorf("gfx/d2d: custom fonts need Windows 10 1703 or later: %w", hrErr("CreateFontSetBuilder", hr))
	}
	defer release(b)
	for f := range files {
		var ff uintptr
		if hr := vcall(te.factory, dwFactoryCreateFontFileReference, wstr(f), ui(0), ref(&ff)); failed(hr) {
			continue
		}
		vcall(b, dwBuilderAddFontFile, ui(ff))
		release(ff)
	}
	var set uintptr
	if hr := vcall(b, dwBuilderCreateFontSet, ref(&set)); failed(hr) {
		return hrErr("CreateFontSet", hr)
	}
	defer release(set)
	var coll uintptr
	if hr := vcall(te.factory, dwFactoryCreateFontCollFromSet, ui(set), ref(&coll)); failed(hr) {
		return hrErr("CreateFontCollectionFromFontSet", hr)
	}
	release(te.custom)
	te.custom = coll
	te.resolved.Clear()
	te.formats.Range(func(k, v any) bool { release(v.(uintptr)); te.formats.Delete(k); return true })
	te.layouts.Range(func(k, v any) bool { release(v.(*textLayout).ptr); te.layouts.Delete(k); return true })
	return nil
}

func (te *textEngine) resolve(name string) Resolved {
	te.sync()
	if v, ok := te.resolved.Load(name); ok {
		return v.(Resolved)
	}
	te.mu.RLock()
	res := resolveName(name, te.faces, func(fam string, custom bool) bool {
		if custom {
			return te.custom != 0 && te.hasFamily(te.custom, fam)
		}
		return te.hasFamily(te.sys, fam)
	})
	te.mu.RUnlock()
	te.resolved.Store(name, res)
	return res
}

func (te *textEngine) format(f gfx.Font) uintptr {
	te.sync()
	k := fmtKey{f.Name, f.Size}
	if v, ok := te.formats.Load(k); ok {
		return v.(uintptr)
	}
	res := te.resolve(f.Name)
	te.mu.RLock()
	coll := uintptr(0)
	if res.Custom {
		coll = te.custom
	}
	te.mu.RUnlock()
	style := uintptr(0)
	if res.Italic {
		style = 2
	}
	size := f.Size
	if size <= 0 {
		size = 1
	}
	var fm uintptr
	hr := vcall(te.factory, dwFactoryCreateTextFormat, wstr(res.Family), ui(coll),
		ui(uintptr(res.Weight)), ui(style), ui(5), fl(size), wstr("en-us"), ref(&fm))
	if failed(hr) {
		return 0
	}
	vcall(fm, dwFormatSetWordWrapping, ui(1)) // NO_WRAP
	actual, loaded := te.formats.LoadOrStore(k, fm)
	if loaded {
		release(fm)
	}
	return actual.(uintptr)
}

// newLayout lays out one line of text; the caller releases ptr or keeps it in the cache.
func (te *textEngine) newLayout(s string, f gfx.Font) *textLayout {
	fm := te.format(f)
	if fm == 0 {
		return nil
	}
	w := utf16Slice(s)
	var lay uintptr
	hr := vcall(te.factory, dwFactoryCreateTextLayout, buf(w), ui(uintptr(len(w))), ui(fm), fl(1e6), fl(1e6), ref(&lay))
	if failed(hr) {
		return nil
	}
	var m dwTextMetrics
	vcall(lay, dwLayoutGetMetrics, ref(&m))
	var lm [4]dwLineMetrics
	var n uint32
	vcall(lay, dwLayoutGetLineMetrics, ref(&lm[0]), ui(uintptr(len(lm))), ref(&n))
	t := &textLayout{ptr: lay}
	t.Width = float64(m.widthTrailing)
	if n > 0 {
		t.Ascent = float64(lm[0].baseline)
		t.Descent = float64(lm[0].height - lm[0].baseline)
	}
	return t
}

func (te *textEngine) layout(s string, f gfx.Font, cache bool) *textLayout {
	te.sync()
	if !cache {
		return te.newLayout(s, f)
	}
	k := layKey{f.Name, f.Size, s}
	if v, ok := te.layouts.Load(k); ok {
		return v.(*textLayout)
	}
	t := te.newLayout(s, f)
	if t == nil {
		return nil
	}
	actual, loaded := te.layouts.LoadOrStore(k, t)
	if loaded {
		release(t.ptr)
	}
	return actual.(*textLayout)
}

// MeasureText returns the DirectWrite extents of s on one line, cached per font and string.
func (r *Renderer) MeasureText(s string, f gfx.Font) gfx.TextMetrics {
	t := r.text.layout(s, f, true)
	if t == nil {
		return gfx.EstimateText{}.MeasureText(s, f)
	}
	return t.TextMetrics
}
