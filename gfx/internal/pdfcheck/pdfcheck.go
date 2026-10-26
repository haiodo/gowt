// Package pdfcheck reads what a test needs to know about a PDF file written by any backend: the
// number of pages, their sizes, the fonts and the embedded images. It looks at the bytes, not at
// the object graph: it inflates every Flate stream it finds (cairo packs objects into object
// streams) and scans the text of the file and of those streams.
package pdfcheck

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"regexp"
	"strconv"
)

// Info is what Inspect found.
type Info struct {
	Pages  []Size
	Fonts  []string // BaseFont names, as written ("/ABCDEF+Name" without the slash), sorted by first appearance
	Images int
	// Text is the decoded text of the whole file, for searching marks such as content operators.
	Text string
}

// Size is a page size in points.
type Size struct{ W, H float64 }

var (
	rePage   = regexp.MustCompile(`/Type\s*/Page\b`)
	reBox    = regexp.MustCompile(`/MediaBox\s*\[\s*([-0-9.]+)\s+([-0-9.]+)\s+([-0-9.]+)\s+([-0-9.]+)\s*\]`)
	reFont   = regexp.MustCompile(`/BaseFont\s*/([^\s/\[\]<>()]+)`)
	reImage  = regexp.MustCompile(`/Subtype\s*/Image\b`)
	reStream = regexp.MustCompile(`(?s)stream\r?\n`)
)

// Inspect parses data. It fails for data that is not a PDF.
func Inspect(data []byte) (*Info, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, errors.New("pdfcheck: no %PDF- header")
	}
	if !bytes.Contains(data[max(0, len(data)-2048):], []byte("%%EOF")) {
		return nil, errors.New("pdfcheck: no %%EOF at the end")
	}
	var all bytes.Buffer
	all.Write(data)
	for _, loc := range reStream.FindAllIndex(data, -1) {
		body := data[loc[1]:]
		if end := bytes.Index(body, []byte("endstream")); end >= 0 {
			body = body[:end]
		}
		zr, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			continue
		}
		plain, _ := io.ReadAll(zr)
		all.WriteByte('\n')
		all.Write(plain)
	}
	text := all.String()
	info := &Info{Text: text}
	// A Pages node has /Type /Pages and may carry a /MediaBox too; only leaves count, and the
	// box that belongs to a leaf is the nearest one.
	for _, loc := range rePage.FindAllStringIndex(text, -1) {
		lo := max(0, loc[0]-600)
		win := text[lo:min(len(text), loc[1]+600)]
		best, bestD := []string(nil), 1<<30
		for _, idx := range reBox.FindAllStringSubmatchIndex(win, -1) {
			if d := abs(lo + idx[0] - loc[0]); d < bestD {
				best, bestD = reBox.FindStringSubmatch(win[idx[0]:idx[1]]), d
			}
		}
		if best == nil {
			info.Pages = append(info.Pages, Size{})
			continue
		}
		x0, _ := strconv.ParseFloat(best[1], 64)
		y0, _ := strconv.ParseFloat(best[2], 64)
		x1, _ := strconv.ParseFloat(best[3], 64)
		y1, _ := strconv.ParseFloat(best[4], 64)
		info.Pages = append(info.Pages, Size{x1 - x0, y1 - y0})
	}
	seen := map[string]bool{}
	for _, m := range reFont.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			info.Fonts = append(info.Fonts, m[1])
		}
	}
	info.Images = len(reImage.FindAllString(text, -1))
	return info, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
