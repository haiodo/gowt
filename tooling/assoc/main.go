// Command assoc writes the files that associate a document type with a program: a Windows .reg file,
// a Linux .desktop file, a D-Bus service file and a shared-mime-info XML. See docs/packaging.md.
//
//	assoc -kind reg     -exe C:\App\app.exe -ext .tnb,.tenniarb -progid Org.App.Doc -name "App document" -icon C:\App\doc.ico
//	assoc -kind desktop -id org.example.App -name "App" -exe /opt/app/app -ext tnb -mime application/x-app [-dbus]
//	assoc -kind service -id org.example.App -exe /opt/app/app
//	assoc -kind mime    -ext tnb,tenniarb -mime application/x-app -name "App document" [-icon application-x-app]
//	assoc -kind icons   -src doc.png -mime application/x-app -out ~/.local/share/icons   (hicolor/NxN/mimetypes/application-x-app.png)
//	assoc -kind icons   -src app.png -iconname app -context apps -out ~/.local/share/icons
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/image/draw"
)

type options struct {
	kind, exe, ext, progid, name, icon, id, mime, categories, wmclass string
	src, out, iconname, context                                       string
	dbus                                                              bool
}

func main() {
	var o options
	flag.StringVar(&o.kind, "kind", "", "reg, desktop, service or mime")
	flag.StringVar(&o.exe, "exe", "", "path of the program")
	flag.StringVar(&o.ext, "ext", "", "comma-separated extensions, with or without the dot")
	flag.StringVar(&o.progid, "progid", "", "reg: ProgID of the document type, such as Org.App.Doc")
	flag.StringVar(&o.name, "name", "", "reg, mime: name of the document type; desktop: name of the program")
	flag.StringVar(&o.icon, "icon", "", "reg: .ico path (add ,N for an icon index); desktop: icon name")
	flag.StringVar(&o.id, "id", "", "desktop, service: application id, the D-Bus name given to SingleInstance")
	flag.StringVar(&o.mime, "mime", "", "desktop, mime: MIME type, such as application/x-app")
	flag.StringVar(&o.categories, "categories", "Utility;", "desktop: Categories")
	flag.StringVar(&o.wmclass, "wmclass", "", "desktop: StartupWMClass, the name given to SetAppName")
	flag.BoolVar(&o.dbus, "dbus", false, "desktop: DBusActivatable=true (needs the service file)")
	flag.StringVar(&o.src, "src", "", "icons: the source PNG, square, 512 or larger for the largest size")
	flag.StringVar(&o.out, "out", "", "icons: the icon theme directory to write hicolor/ into")
	flag.StringVar(&o.iconname, "iconname", "", "icons: icon name; default is the MIME type with - for /")
	flag.StringVar(&o.context, "context", "mimetypes", "icons: mimetypes or apps")
	flag.Parse()
	if o.kind == "icons" {
		paths, err := writeIcons(o)
		if err != nil {
			fmt.Fprintln(os.Stderr, "assoc:", err)
			os.Exit(2)
		}
		fmt.Println(strings.Join(paths, "\n"))
		return
	}
	out, err := generate(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "assoc:", err)
		os.Exit(2)
	}
	os.Stdout.Write(out)
}

func generate(o options) ([]byte, error) {
	var s string
	var err error
	switch o.kind {
	case "reg":
		s, err = regFile(o)
		// regedit reads Version 5.00 files as UTF-16 with a BOM; ASCII also works as ANSI.
		if err == nil && strings.ContainsFunc(s, func(r rune) bool { return r > 127 }) {
			return utf16le(s), nil
		}
	case "desktop":
		s, err = desktopFile(o)
	case "service":
		s, err = serviceFile(o)
	case "mime":
		s, err = mimeXML(o)
	default:
		err = errors.New("-kind must be reg, desktop, service or mime")
	}
	return []byte(s), err
}

func utf16le(s string) []byte {
	b := []byte{0xff, 0xfe}
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func exts(list string) ([]string, error) {
	var out []string
	for _, e := range strings.Split(list, ",") {
		if e = strings.TrimPrefix(strings.TrimSpace(e), "."); e != "" {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("-ext is empty")
	}
	return out, nil
}

func need(o options, names ...string) error {
	vals := map[string]string{"exe": o.exe, "progid": o.progid, "name": o.name, "id": o.id, "mime": o.mime}
	for _, n := range names {
		if vals[n] == "" {
			return errors.New("-" + n + " is required for -kind " + o.kind)
		}
		if strings.ContainsAny(vals[n], "\r\n") {
			return errors.New("-" + n + " has a line break")
		}
	}
	return nil
}

func regQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func regFile(o options) (string, error) {
	if err := need(o, "exe", "progid", "name"); err != nil {
		return "", err
	}
	es, err := exts(o.ext)
	if err != nil {
		return "", err
	}
	const classes = `[HKEY_CURRENT_USER\Software\Classes\`
	var b strings.Builder
	b.WriteString("Windows Registry Editor Version 5.00\r\n")
	for _, e := range es {
		fmt.Fprintf(&b, "\r\n%s.%s]\r\n@=%s\r\n", classes, e, regQuote(o.progid))
	}
	fmt.Fprintf(&b, "\r\n%s%s]\r\n@=%s\r\n", classes, o.progid, regQuote(o.name))
	if o.icon != "" {
		fmt.Fprintf(&b, "\r\n%s%s\\DefaultIcon]\r\n@=%s\r\n", classes, o.progid, regQuote(o.icon))
	}
	fmt.Fprintf(&b, "\r\n%s%s\\shell\\open\\command]\r\n@=%s\r\n", classes, o.progid, regQuote(`"`+o.exe+`" "%1"`))
	return b.String(), nil
}

func desktopFile(o options) (string, error) {
	if err := need(o, "id", "name", "exe", "mime"); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("[Desktop Entry]\nType=Application\n")
	fmt.Fprintf(&b, "Name=%s\nExec=%s %%U\n", o.name, o.exe)
	if o.icon != "" {
		fmt.Fprintf(&b, "Icon=%s\n", o.icon)
	}
	fmt.Fprintf(&b, "Categories=%s\nMimeType=%s;\n", o.categories, o.mime)
	if o.wmclass != "" {
		fmt.Fprintf(&b, "StartupWMClass=%s\n", o.wmclass)
	}
	if o.dbus {
		b.WriteString("DBusActivatable=true\n")
	}
	return b.String(), nil
}

func serviceFile(o options) (string, error) {
	if err := need(o, "id", "exe"); err != nil {
		return "", err
	}
	return fmt.Sprintf("[D-BUS Service]\nName=%s\nExec=%s\n", o.id, o.exe), nil
}

func mimeXML(o options) (string, error) {
	if err := need(o, "mime", "name"); err != nil {
		return "", err
	}
	es, err := exts(o.ext)
	if err != nil {
		return "", err
	}
	esc := func(s string) string {
		var b bytes.Buffer
		xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">` + "\n")
	fmt.Fprintf(&b, "  <mime-type type=\"%s\">\n    <comment>%s</comment>\n", esc(o.mime), esc(o.name))
	if o.icon != "" {
		fmt.Fprintf(&b, "    <icon name=\"%s\"/>\n", esc(o.icon))
	}
	for _, e := range es {
		fmt.Fprintf(&b, "    <glob pattern=\"*.%s\"/>\n", esc(e))
	}
	b.WriteString("  </mime-type>\n</mime-info>\n")
	return b.String(), nil
}

// iconSizes are the hicolor sizes written for an icon.
var iconSizes = []int{16, 22, 24, 32, 48, 64, 128, 256, 512}

// writeIcons scales the source PNG into <out>/hicolor/NxN/<context>/<name>.png and returns the paths.
// A size larger than the source is left out, so a small source gives fewer sizes, not blurred ones.
func writeIcons(o options) ([]string, error) {
	if o.src == "" || o.out == "" {
		return nil, errors.New("-src and -out are required for -kind icons")
	}
	if o.context != "mimetypes" && o.context != "apps" {
		return nil, errors.New("-context must be mimetypes or apps")
	}
	name := o.iconname
	if name == "" {
		if o.mime == "" {
			return nil, errors.New("-mime or -iconname is required for -kind icons")
		}
		name = strings.ReplaceAll(o.mime, "/", "-")
	}
	f, err := os.Open(o.src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.src, err)
	}
	if b := src.Bounds(); b.Dx() != b.Dy() {
		return nil, fmt.Errorf("%s is %dx%d: the icon must be square", o.src, b.Dx(), b.Dy())
	}
	var paths []string
	for _, n := range iconSizes {
		if n > src.Bounds().Dx() {
			continue
		}
		dst := image.NewNRGBA(image.Rect(0, 0, n, n))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		dir := filepath.Join(o.out, "hicolor", fmt.Sprintf("%dx%d", n, n), o.context)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		p := filepath.Join(dir, name+".png")
		var b bytes.Buffer
		if err := png.Encode(&b, dst); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s is smaller than %d pixels", o.src, iconSizes[0])
	}
	return paths, nil
}
