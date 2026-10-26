//go:build windows

package d2d

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/pdf"
)

var _ gfx.PDFRenderer = (*Renderer)(nil)

// NewPDF starts a PDF file written to w by Close. Direct2D has no PDF output, so the pure Go
// writer of gfx/pdf draws the pictures; the fonts are the files DirectWrite's choice for each
// gfx.Font points at, embedded whole. See package gfx/pdf for what that writer leaves out.
func (r *Renderer) NewPDF(w io.Writer) (gfx.Document, error) {
	return pdf.New(w, &fontFiles{te: r.text}), nil
}

type fontFiles struct {
	te    *textEngine
	mu    sync.Mutex
	cache map[Resolved][]byte
}

// FontFile reads the file of the face the text engine resolves f to.
func (ff *fontFiles) FontFile(f gfx.Font) ([]byte, bool) {
	res := ff.te.resolve(f)
	ff.mu.Lock()
	defer ff.mu.Unlock()
	if data, ok := ff.cache[res]; ok {
		return data, data != nil
	}
	if ff.cache == nil {
		ff.cache = map[Resolved][]byte{}
	}
	var data []byte
	if path := faceFile(res); path != "" {
		data, _ = os.ReadFile(path)
	}
	ff.cache[res] = data
	return data, data != nil
}

// faceFile finds the font file of res among the registered files and the installed ones.
func faceFile(res Resolved) string {
	var cands []string
	fontReg.mu.RLock()
	for _, fi := range fontReg.faces {
		for _, fam := range fi.Families {
			if strings.EqualFold(fam, res.Family) {
				cands = append(cands, fi.File)
			}
		}
	}
	fontReg.mu.RUnlock()
	if !res.Custom {
		cands = append(cands, installedFonts(res.Family)...)
	}
	best, bestScore := "", 1<<30
	for _, c := range cands {
		fi, err := ReadFaceInfo(c)
		if err != nil {
			continue
		}
		in := false
		for _, fam := range fi.Families {
			in = in || strings.EqualFold(fam, res.Family)
		}
		if !in {
			continue
		}
		score := abs(fi.Weight - res.Weight)
		if fi.Italic != res.Italic {
			score += 1000
		}
		if strings.EqualFold(filepath.Ext(c), ".ttc") {
			score++ // a collection is drawn as outlines, a single file embeds
		}
		if score < bestScore {
			best, bestScore = c, score
		}
	}
	return best
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// installedFonts lists the font files the registry names for the family: the value names read
// "Segoe UI Bold (TrueType)".
func installedFonts(family string) []string {
	const key = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`
	dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	var out []string
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, key, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
			if !strings.HasPrefix(strings.ToLower(n), strings.ToLower(family)) {
				continue
			}
			if v, _, err := k.GetStringValue(n); err == nil {
				if !filepath.IsAbs(v) {
					v = filepath.Join(dir, v)
				}
				out = append(out, v)
			}
		}
		k.Close()
	}
	return out
}
