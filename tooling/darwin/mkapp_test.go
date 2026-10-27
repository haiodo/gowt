//go:build darwin

package mkapp

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// mkapp runs mkapp.sh on a copy of /usr/bin/true with the environment and returns Info.plist as JSON.
func mkapp(t *testing.T, env ...string) (app string, plist map[string]any) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "demo")
	data, err := os.ReadFile("/usr/bin/true")
	if err != nil {
		t.Skip(err)
	}
	os.WriteFile(bin, data, 0o755)
	cmd := exec.Command("bash", "mkapp.sh", bin, dir, "Demo")
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("mkapp.sh: %v\n%s", err, out)
	}
	app = filepath.Join(dir, "Demo.app")
	j, err := exec.Command("plutil", "-convert", "json", "-o", "-", filepath.Join(app, "Contents", "Info.plist")).Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(j, &plist); err != nil {
		t.Fatal(err)
	}
	return app, plist
}

func png(t *testing.T, path string, n int) {
	t.Helper()
	// a PNG made by sips from a system icon-sized image: any square PNG will do
	src := "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/GenericDocumentIcon.icns"
	if _, err := os.Stat(src); err != nil {
		t.Skip(err)
	}
	if out, err := exec.Command("sips", "-s", "format", "png", "-z", "1024", "1024", src, "--out", path).CombinedOutput(); err != nil {
		t.Skipf("sips: %v %s", err, out)
	}
}

func TestDefaults(t *testing.T) {
	_, p := mkapp(t)
	if p["CFBundleIdentifier"] != "com.github.haiodo.gowt.demo" || p["CFBundleShortVersionString"] != "0.1" || p["CFBundleVersion"] != "1" {
		t.Errorf("defaults: %v", p)
	}
	if _, ok := p["CFBundleIconFile"]; ok {
		t.Error("an icon is named without one")
	}
}

func TestBundleVersionAndIcon(t *testing.T) {
	icon := filepath.Join(t.TempDir(), "app.png")
	png(t, icon, 1024)
	app, p := mkapp(t, "BUNDLE_ID=org.example.Tenniarb", "VERSION=2.5.1", "BUILD=307", "ICON="+icon)
	if p["CFBundleIdentifier"] != "org.example.Tenniarb" || p["CFBundleShortVersionString"] != "2.5.1" || p["CFBundleVersion"] != "307" || p["CFBundleIconFile"] != "AppIcon" {
		t.Errorf("plist: %v", p)
	}
	b, err := os.ReadFile(filepath.Join(app, "Contents", "Resources", "AppIcon.icns"))
	if err != nil || string(b[:4]) != "icns" {
		t.Errorf("AppIcon.icns: %v %q", err, b[:min(4, len(b))])
	}
}

func TestDocumentTypeConformance(t *testing.T) {
	icon := filepath.Join(t.TempDir(), "doc.png")
	png(t, icon, 1024)
	app, p := mkapp(t, "DOC_EXTS=tnb tenniarb", "DOC_UTI=org.example.tnb", "DOC_UTI_CONFORMS=public.json public.text", "DOC_ICON="+icon)
	types := p["CFBundleDocumentTypes"].([]any)[0].(map[string]any)
	if types["CFBundleTypeIconFile"] != "doc.icns" {
		t.Errorf("document icon: %v", types)
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "Resources", "doc.icns")); err != nil {
		t.Error(err)
	}
	exp := p["UTExportedTypeDeclarations"].([]any)[0].(map[string]any)
	got := exp["UTTypeConformsTo"].([]any)
	if len(got) != 2 || got[0] != "public.json" || got[1] != "public.text" {
		t.Errorf("conforms to %v", got)
	}
	_, def := mkapp(t, "DOC_EXTS=x", "DOC_UTI=org.example.x")
	if c := def["UTExportedTypeDeclarations"].([]any)[0].(map[string]any)["UTTypeConformsTo"].([]any); len(c) != 1 || c[0] != "public.data" {
		t.Errorf("default conformance %v", c)
	}
}
