package main

import (
	"bytes"
	"debug/pe"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// walk reads the resource tree of a .rsrc section: data[type][id] is the data of the entry.
func walk(t *testing.T, sec []byte, rva int) map[uint16]map[uint16][]byte {
	t.Helper()
	got := map[uint16]map[uint16][]byte{}
	u32 := func(o int) int { return int(le.Uint32(sec[o:])) }
	entries := func(dir int) (ids, offs []int) {
		n := int(le.Uint16(sec[dir+12:])) + int(le.Uint16(sec[dir+14:]))
		for i := 0; i < n; i++ {
			ids = append(ids, u32(dir+16+8*i))
			offs = append(offs, u32(dir+20+8*i))
		}
		return
	}
	tids, toffs := entries(0)
	for i, typ := range tids {
		got[uint16(typ)] = map[uint16][]byte{}
		nids, noffs := entries(toffs[i] &^ 0x80000000)
		for j, id := range nids {
			lids, loffs := entries(noffs[j] &^ 0x80000000)
			if len(lids) != 1 || lids[0] != langEnUS {
				t.Fatalf("type %d id %d: languages %v", typ, id, lids)
			}
			e := loffs[0]
			if e&0x80000000 != 0 {
				t.Fatal("a language entry points at a directory")
			}
			off, size := u32(e)-rva, u32(e+4) // in an executable the offset is an RVA
			got[uint16(typ)][uint16(id)] = sec[off : off+size]
		}
	}
	return got
}

func testPNG(t *testing.T, dir string, n int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i := range img.Pix {
		img.Pix[i] = byte(i * 7)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "i"+strings.Repeat("x", n/16)+".png")
	if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func utf16At(b []byte) string {
	var u []uint16
	for i := 0; i+1 < len(b) && (b[i] != 0 || b[i+1] != 0); i += 2 {
		u = append(u, le.Uint16(b[i:]))
	}
	return string(utf16.Decode(u))
}

func TestResourcesAreWrittenAndLinked(t *testing.T) {
	dir := t.TempDir()
	icon := testPNG(t, dir, 16) + "," + testPNG(t, dir, 48) + "," + testPNG(t, dir, 256)
	ics, err := icons(icon)
	if err != nil {
		t.Fatal(err)
	}
	ver, err := versionResource(versionInfo{version: "1.2.3.4", product: "Prod", company: "Org", description: "A program", copyright: "(c) Org", original: "app.exe"})
	if err != nil {
		t.Fatal(err)
	}
	rs := append(ics, resource{rtVersion, 1, ver}, resource{rtManifest, 1, []byte("<assembly/>")})

	for _, arch := range []string{"amd64", "arm64"} {
		obj, err := coff(arch, append([]resource(nil), rs...))
		if err != nil {
			t.Fatal(err)
		}
		// the object as a COFF file: one .rsrc section, one relocation per data entry
		f, err := pe.NewFile(bytes.NewReader(obj))
		if err != nil {
			t.Fatalf("%s: not a COFF file: %v", arch, err)
		}
		sec := f.Section(".rsrc")
		if sec == nil || int(sec.NumberOfRelocations) != len(rs) {
			t.Fatalf("%s: section %v", arch, sec)
		}
		raw, _ := sec.Data()
		got := walk(t, raw, 0)
		if len(got[rtIcon]) != 3 || len(got[rtGroupIcon]) != 1 || !bytes.Equal(got[rtManifest][1], []byte("<assembly/>")) {
			t.Fatalf("%s: resources %v", arch, got)
		}
		g := got[rtGroupIcon][1] // 3 images: sizes 16, 48 and 256 (stored as 0)
		if le.Uint16(g[4:]) != 3 || g[6] != 16 || g[6+14] != 48 || g[6+28] != 0 {
			t.Errorf("%s: icon group %v", arch, g)
		}
		if !bytes.HasPrefix(got[rtIcon][1], []byte("\x89PNG")) {
			t.Errorf("%s: the icon image is not the PNG file", arch)
		}
	}

	// the version resource, read back by hand
	v := ver
	if le.Uint16(v) != uint16(len(v)) || le.Uint16(v[2:]) != 52 || utf16At(v[6:]) != "VS_VERSION_INFO" {
		t.Fatalf("version header %v", v[:40])
	}
	fixed := v[40:] // 6 header bytes and the 32 of the key, padded to 4
	if le.Uint32(fixed) != 0xFEEF04BD {
		t.Fatalf("fixed info is not where it should be: %x", fixed[:8])
	}
	if ms, ls := le.Uint32(fixed[8:]), le.Uint32(fixed[12:]); ms != 1<<16|2 || ls != 3<<16|4 {
		t.Errorf("file version %x %x", ms, ls)
	}
	text := string(v)
	for _, want := range []string{"CompanyName", "Org", "FileDescription", "A program", "ProductName", "Prod", "1.2.3.4", "OriginalFilename", "app.exe"} {
		if !strings.Contains(text, string(wstr(want)[:len(wstr(want))-2])) {
			t.Errorf("version info lacks %q", want)
		}
	}
}

// The linker accepts the object: a program built with it carries the resources in its .rsrc.
func TestLinkedExecutableHasResources(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program")
	}
	dir := t.TempDir()
	ics, err := icons(testPNG(t, dir, 32))
	if err != nil {
		t.Fatal(err)
	}
	ver, _ := versionResource(versionInfo{version: "2.0", product: "P"})
	obj, err := coff("arm64", append(ics, resource{rtVersion, 1, ver}, resource{rtManifest, 1, []byte("<assembly/>")}))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\ngo 1.21\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "res_windows_arm64.syso"), obj, 0o644)
	cmd := exec.Command("go", "build", "-o", "x.exe", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=arm64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	f, err := pe.Open(filepath.Join(dir, "x.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sec := f.Section(".rsrc")
	if sec == nil {
		t.Fatal("the executable has no .rsrc")
	}
	raw, _ := sec.Data()
	got := walk(t, raw, int(sec.VirtualAddress))
	if len(got[rtIcon]) != 1 || len(got[rtGroupIcon]) != 1 || len(got[rtVersion]) != 1 || len(got[rtManifest]) != 1 {
		t.Errorf("linked resources: %v", got)
	}
}

func TestIcoFileAndErrors(t *testing.T) {
	dir := t.TempDir()
	p := testPNG(t, dir, 16)
	pngData, _ := os.ReadFile(p)
	ico := []byte{0, 0, 1, 0, 1, 0, 16, 16, 0, 0, 1, 0, 32, 0}
	ico = le.AppendUint32(ico, uint32(len(pngData)))
	ico = le.AppendUint32(ico, 22)
	ico = append(ico, pngData...)
	icoPath := filepath.Join(dir, "a.ico")
	os.WriteFile(icoPath, ico, 0o644)
	rs, err := icons(icoPath)
	if err != nil || len(rs) != 2 || !bytes.Equal(rs[0].data, pngData) {
		t.Fatalf("ico: %v %v", rs, err)
	}
	if _, err := icons(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("a missing file is accepted")
	}
	wide := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	var b bytes.Buffer
	png.Encode(&b, wide)
	wp := filepath.Join(dir, "wide.png")
	os.WriteFile(wp, b.Bytes(), 0o644)
	if _, err := icons(wp); err == nil {
		t.Error("a non-square image is accepted")
	}
	if _, err := parseVersion("1.2.3.4.5"); err == nil {
		t.Error("five version parts are accepted")
	}
	if _, err := coff("amd64", nil); err == nil {
		t.Error("an empty resource list is accepted")
	}
}
