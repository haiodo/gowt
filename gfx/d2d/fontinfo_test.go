package d2d

import (
	"os"
	"testing"
)

func TestReadFaceInfoPlex(t *testing.T) {
	dir := os.Getenv("GFX_FONT_DIR")
	if dir == "" {
		t.Skip("GFX_FONT_DIR not set")
	}
	cases := map[string]struct {
		w  int
		ps string
	}{"IBMPlexSans-SemiBold.otf": {600, "IBMPlexSans-SmBld"}, "IBMPlexSans-Regular.otf": {400, "IBMPlexSans"}, "IBMPlexSans-Bold.otf": {700, "IBMPlexSans-Bold"}}
	for f, want := range cases {
		fi, err := ReadFaceInfo(dir + "/" + f)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %+v", f, fi)
		if fi.Weight != want.w || fi.PostScript != want.ps || len(fi.Families) == 0 {
			t.Errorf("%s: got %+v", f, fi)
		}
	}
}

func TestResolveName(t *testing.T) {
	reg := map[string]FaceInfo{"ibmplexsans-smbld": {PostScript: "IBMPlexSans-SmBld", Families: []string{"IBM Plex Sans"}, Weight: 600}}
	has := func(f string, custom bool) bool { return custom == (f == "IBM Plex Sans") && (custom || f == "Arial") }
	if r := resolveName("IBMPlexSans-SmBld", reg, has); r != (Resolved{"IBM Plex Sans", 600, false, true}) {
		t.Errorf("plex: %+v", r)
	}
	if r := resolveName("HelveticaNeue-Bold", reg, has); r != (Resolved{systemSans, 700, false, false}) {
		t.Errorf("helvetica: %+v", r)
	}
	if r := resolveName("Arial", reg, has); r.Family != "Arial" {
		t.Errorf("arial: %+v", r)
	}
	if r := resolveName("NoSuchFont-Probe", reg, has); r.Family != systemSans {
		t.Errorf("unknown: %+v", r)
	}
}
