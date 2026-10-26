package main

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegFile(t *testing.T) {
	got, err := regFile(options{exe: `C:\Program Files\App\app.exe`, ext: ".tnb, tenniarb", progid: "Org.App.Doc", name: `App "doc"`, icon: `C:\Program Files\App\doc.ico,0`})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Windows Registry Editor Version 5.00\r\n",
		"[HKEY_CURRENT_USER\\Software\\Classes\\.tnb]\r\n@=\"Org.App.Doc\"\r\n",
		"[HKEY_CURRENT_USER\\Software\\Classes\\.tenniarb]\r\n@=\"Org.App.Doc\"\r\n",
		"[HKEY_CURRENT_USER\\Software\\Classes\\Org.App.Doc]\r\n@=\"App \\\"doc\\\"\"\r\n",
		"\\Org.App.Doc\\DefaultIcon]\r\n@=\"C:\\\\Program Files\\\\App\\\\doc.ico,0\"\r\n",
		"\\Org.App.Doc\\shell\\open\\command]\r\n@=\"\\\"C:\\\\Program Files\\\\App\\\\app.exe\\\" \\\"%1\\\"\"\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("reg file lacks %q:\n%s", want, got)
		}
	}
}

func TestRegUTF16ForNonASCII(t *testing.T) {
	b, err := generate(options{kind: "reg", exe: `C:\a.exe`, ext: "x", progid: "A.B", name: "\u0414\u043e\u043a\u0443\u043c\u0435\u043d\u0442"})
	if err != nil || len(b) < 2 || b[0] != 0xff || b[1] != 0xfe {
		t.Errorf("non-ASCII reg is not UTF-16 with a BOM: %v %x", err, b[:2])
	}
}

func TestDesktopAndService(t *testing.T) {
	got, err := desktopFile(options{id: "org.example.App", name: "App", exe: "/opt/app/app", mime: "application/x-app", icon: "app", dbus: true, wmclass: "app"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[Desktop Entry]\n", "Exec=/opt/app/app %U\n", "MimeType=application/x-app;\n", "DBusActivatable=true\n", "StartupWMClass=app\n", "Icon=app\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("desktop file lacks %q:\n%s", want, got)
		}
	}
	svc, _ := serviceFile(options{id: "org.example.App", exe: "/opt/app/app"})
	if svc != "[D-BUS Service]\nName=org.example.App\nExec=/opt/app/app\n" {
		t.Errorf("service file:\n%s", svc)
	}
}

func TestMimeXML(t *testing.T) {
	got, err := mimeXML(options{mime: "application/x-app", name: "A & B doc", ext: "tnb,.tenniarb"})
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Types []struct {
			Type    string `xml:"type,attr"`
			Comment string `xml:"comment"`
			G       []struct {
				P string `xml:"pattern,attr"`
			} `xml:"glob"`
		} `xml:"mime-type"`
	}
	if err := xml.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("not XML: %v\n%s", err, got)
	}
	mt := v.Types[0]
	if mt.Type != "application/x-app" || mt.Comment != "A & B doc" || len(mt.G) != 2 || mt.G[0].P != "*.tnb" || mt.G[1].P != "*.tenniarb" {
		t.Errorf("parsed %+v", mt)
	}
}

func TestMissingAndBadInput(t *testing.T) {
	for _, o := range []options{{kind: "reg"}, {kind: "desktop", id: "a.b"}, {kind: "mime", mime: "a/b", name: "n"}, {kind: "x"}, {kind: "service", id: "a.b", exe: "x\ny"}} {
		if _, err := generate(o); err == nil {
			t.Errorf("%+v: no error", o)
		}
	}
}

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
}

func TestHicolorIcons(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "doc.png")
	writePNG(t, src, 128, 128)
	paths, err := writeIcons(options{src: src, out: dir, mime: "application/x-app", context: "mimetypes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 7 { // 16 22 24 32 48 64 128: nothing is scaled up
		t.Fatalf("%d icons: %v", len(paths), paths)
	}
	for _, n := range []int{16, 48, 128} {
		p := filepath.Join(dir, "hicolor", fmt.Sprintf("%dx%d", n, n), "mimetypes", "application-x-app.png")
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil || cfg.Width != n || cfg.Height != n {
			t.Errorf("%s: %v %v", p, cfg, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "hicolor", "256x256")); err == nil {
		t.Error("a 256 icon was made from a 128 source")
	}
	wide := filepath.Join(dir, "wide.png")
	writePNG(t, wide, 20, 10)
	if _, err := writeIcons(options{src: wide, out: dir, iconname: "x", context: "apps"}); err == nil {
		t.Error("a non-square source is accepted")
	}
	if _, err := writeIcons(options{src: src, out: dir, context: "mimetypes"}); err == nil {
		t.Error("no icon name is accepted")
	}
}

func TestMimeXMLIcon(t *testing.T) {
	got, err := mimeXML(options{mime: "application/x-app", name: "A", ext: "tnb", icon: "application-x-app"})
	if err != nil || !strings.Contains(got, `<icon name="application-x-app"/>`) {
		t.Errorf("%v\n%s", err, got)
	}
}
