package d2d

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// FaceInfo is what a font file says about its first face: the names a Picture may use for it and
// the family, weight and slant DirectWrite needs to pick it.
type FaceInfo struct {
	PostScript string
	Families   []string // typographic family first, then the legacy one
	Weight     int      // 100..900
	Italic     bool
	File       string
}

// ReadFaceInfo reads the name and OS/2 tables of an OpenType or TrueType file.
func ReadFaceInfo(path string) (FaceInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FaceInfo{}, err
	}
	var fi FaceInfo
	fi.File = path
	var f *sfnt.Font
	if strings.EqualFold(filepath.Ext(path), ".ttc") {
		c, err := sfnt.ParseCollection(data)
		if err != nil {
			return fi, err
		}
		if f, err = c.Font(0); err != nil {
			return fi, err
		}
	} else if f, err = sfnt.Parse(data); err != nil {
		return fi, err
	}
	var buf sfnt.Buffer
	get := func(id sfnt.NameID) string {
		s, err := f.Name(&buf, id)
		if err != nil {
			return ""
		}
		return s
	}
	fi.PostScript = get(sfnt.NameIDPostScript)
	for _, id := range []sfnt.NameID{sfnt.NameIDTypographicFamily, sfnt.NameIDFamily} {
		if s := get(id); s != "" && (len(fi.Families) == 0 || fi.Families[0] != s) {
			fi.Families = append(fi.Families, s)
		}
	}
	fi.Weight, fi.Italic = os2(data)
	if fi.Weight == 0 {
		fi.Weight, fi.Italic = styleOf(fi.PostScript)
	}
	return fi, nil
}

// os2 reads usWeightClass and the italic bit of fsSelection from a single-font sfnt file; 0 if absent.
func os2(data []byte) (weight int, italic bool) {
	if len(data) < 12 {
		return 0, false
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		rec := 12 + 16*i
		if rec+16 > len(data) {
			return 0, false
		}
		if string(data[rec:rec+4]) != "OS/2" {
			continue
		}
		off := int(binary.BigEndian.Uint32(data[rec+8:]))
		if off+64 > len(data) {
			return 0, false
		}
		w := int(binary.BigEndian.Uint16(data[off+4:]))
		sel := binary.BigEndian.Uint16(data[off+62:])
		return w, sel&1 != 0
	}
	return 0, false
}

// styleOf guesses weight and slant from the style words of a PostScript name such as "Inter-SemiBold".
func styleOf(name string) (weight int, italic bool) {
	n := strings.ToLower(name)
	italic = strings.Contains(n, "italic") || strings.Contains(n, "oblique")
	switch {
	case strings.Contains(n, "thin"):
		weight = 100
	case strings.Contains(n, "extralight") || strings.Contains(n, "xlight"):
		weight = 200
	case strings.Contains(n, "light"):
		weight = 300
	case strings.Contains(n, "semibold") || strings.Contains(n, "smbld") || strings.Contains(n, "demibold"):
		weight = 600
	case strings.Contains(n, "extrabold") || strings.Contains(n, "xbold"):
		weight = 800
	case strings.Contains(n, "black") || strings.Contains(n, "heavy"):
		weight = 900
	case strings.Contains(n, "bold"):
		weight = 700
	case strings.Contains(n, "medium") || strings.Contains(n, "medm"):
		weight = 500
	default:
		weight = 400
	}
	return weight, italic
}

// Resolved is the DirectWrite request a Picture's font name turns into.
type Resolved struct {
	Family string
	Weight int
	Italic bool
	Custom bool // Family lives in the collection built from registered files
}

// systemSans stands in for the fonts a macOS design names and Windows lacks.
const systemSans = "Segoe UI"

// resolveName maps a gfx font name (PostScript name or family) to a family with weight and slant.
// registered is keyed by lowercase PostScript name; hasFamily asks the custom or system collection.
func resolveName(name string, registered map[string]FaceInfo, hasFamily func(family string, custom bool) bool) Resolved {
	key := strings.ToLower(name)
	if fi, ok := registered[key]; ok {
		for _, fam := range fi.Families {
			if hasFamily(fam, true) {
				return Resolved{fam, fi.Weight, fi.Italic, true}
			}
		}
	}
	if hasFamily(name, true) {
		return Resolved{name, 400, false, true}
	}
	w, it := styleOf(name)
	base := name
	if i := strings.IndexByte(name, '-'); i > 0 {
		base = name[:i]
	}
	if isMacSans(base) {
		return Resolved{systemSans, w, it, false}
	}
	if hasFamily(name, false) {
		return Resolved{name, 400, false, false}
	}
	if hasFamily(base, false) {
		return Resolved{base, w, it, false}
	}
	return Resolved{systemSans, w, it, false}
}

// isMacSans reports whether a family name is one of the macOS system sans faces.
func isMacSans(family string) bool {
	lb := strings.ToLower(strings.ReplaceAll(family, " ", ""))
	return strings.HasPrefix(lb, "helvetica") || strings.HasPrefix(lb, "sfpro") || strings.HasPrefix(lb, "sanfrancisco") ||
		lb == ".sfnstext" || lb == "system-ui" || lb == "-apple-system" || lb == "systemfont" || lb == "inter"
}

// resolveFamily is the request for a family at a CSS weight (0 is 400) and slant: the registered
// collection first, then the system one, the macOS sans names and anything else land on Segoe UI.
func resolveFamily(family string, weight int, italic bool, hasFamily func(family string, custom bool) bool) Resolved {
	if weight == 0 {
		weight = 400
	}
	switch {
	case family != "" && hasFamily(family, true):
		return Resolved{family, weight, italic, true}
	case family != "" && !isMacSans(family) && hasFamily(family, false):
		return Resolved{family, weight, italic, false}
	}
	return Resolved{systemSans, weight, italic, false}
}
