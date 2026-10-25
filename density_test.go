package gowt

import (
	"math"
	"testing"
)

const linuxBody = 11 * 96.0 / 72 // GNOME's 11 pt UI font at 96 dpi

var (
	macSys = system{goos: "darwin", bodyPx: 13, systemPx: 13, mono: "Menlo"}
	winSys = system{goos: "windows", bodyPx: 14, systemPx: 12, family: "Segoe UI Variable Text",
		display: "Segoe UI Variable Display", mono: "Cascadia Mono"}
	linSys = system{goos: "linux", bodyPx: linuxBody, systemPx: linuxBody, mono: "Ubuntu Sans Mono"}
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestTypeNative(t *testing.T) {
	for _, c := range []struct {
		sys                                        system
		caption, body, subtitle, title, largeTitle float64
		family, titleFamily, mono                  string
	}{
		{macSys, 11, 13, 15, 22, 26, "", "", "Menlo"},
		{winSys, 12, 14, 20, 28, 40, "Segoe UI Variable Text", "Segoe UI Variable Display", "Cascadia Mono"},
		{linSys, linuxBody * 0.82, linuxBody, linuxBody * 1.2, linuxBody * 1.4, linuxBody * 1.8, "", "", "Ubuntu Sans Mono"},
	} {
		ty, _, body, _ := resolveTokens(c.sys, DensityNative)
		if !near(body, c.body) || !near(ty.Caption.Px, c.caption) || !near(ty.Body.Px, c.body) ||
			!near(ty.Subtitle.Px, c.subtitle) || !near(ty.Title.Px, c.title) || !near(ty.LargeTitle.Px, c.largeTitle) {
			t.Errorf("%s sizes: %+v", c.sys.goos, ty)
		}
		if ty.Body.Name != c.family || ty.Title.Name != c.titleFamily || ty.Mono.Name != c.mono || !near(ty.Mono.Px, c.body) {
			t.Errorf("%s families: %+v", c.sys.goos, ty)
		}
		if ty.Body.Weight != Regular || ty.BodyStrong.Weight != Semibold || ty.Title.Weight != Semibold {
			t.Errorf("%s weights: %+v", c.sys.goos, ty)
		}
	}
}

func TestSpacingNative(t *testing.T) {
	for _, c := range []struct {
		sys                                           system
		factor                                        float64
		gap, margin, control, row, sideRow, sideWidth int
	}{
		{macSys, 1, 8, 16, 28, 32, 28, 220},
		{winSys, 14.0 / 13, 8, 16, 32, 36, 32, 236},
		{linSys, linuxBody / 13, 8, 20, 32, 36, 32, 248},
	} {
		_, s, _, f := resolveTokens(c.sys, DensityNative)
		if !near(f, c.factor) || s.SM != c.gap || s.PageMargin != c.margin || s.ControlHeight != c.control ||
			s.RowHeight != c.row || s.SidebarRow != c.sideRow || s.SidebarWidth != c.sideWidth {
			t.Errorf("%s factor %.3f: %+v", c.sys.goos, f, s)
		}
	}
}

func TestDensitySameIsOneLayout(t *testing.T) {
	_, ms, mb, _ := resolveTokens(macSys, DensitySame)
	for _, sys := range []system{winSys, linSys} {
		ty, s, body, f := resolveTokens(sys, DensitySame)
		if s != ms || body != mb || f != 1 || ty.Body.Px != 13 || !near(ty.Title.Px, 22) {
			t.Errorf("%s: %+v body %v factor %v", sys.goos, s, body, f)
		}
	}
	want := SpacingScale{XS: 4, SM: 8, MD: 12, LG: 16, XL: 32, PageMargin: 16, RowHeight: 32,
		ControlHeight: 28, SidebarRow: 28, SidebarWidth: 220, InspectorWidth: 260}
	if ms != want {
		t.Errorf("base: %+v", ms)
	}
}

func TestDensityFactor(t *testing.T) {
	ty, s, body, f := resolveTokens(winSys, DensityFactor(1.25))
	if body != 16.25 || f != 1.25 || s.LG != 20 || s.ControlHeight != 36 || !near(ty.Subtitle.Px, 15*1.25) {
		t.Errorf("%+v %+v %v", ty, s, body)
	}
	if DensityFactor(0) != DensitySame {
		t.Error("a factor of 0 should mean 1")
	}
}

func TestDefaultsUnchanged(t *testing.T) {
	d := defaultSpacing()
	if d.XS != 4 || d.SM != 8 || d.MD != 12 || d.LG != 16 || d.XL != 32 {
		t.Fatalf("legacy gaps moved: %+v", d)
	}
	if ChosenDensity() != (Density{}) || Scaled(37) != 37 {
		t.Fatalf("legacy Scaled changed")
	}
}

func TestSnap(t *testing.T) {
	for in, want := range map[float64]int{0: 0, 1: 4, 4.5: 4, 6: 8, 17.2: 16, 18.05: 20, 34.46: 36} {
		if got := snap(in); got != want {
			t.Errorf("snap(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestMonoFamily(t *testing.T) {
	for in, want := range map[string]string{
		"'Ubuntu Sans Mono 13'\n": "Ubuntu Sans Mono", "'Monospace 10'": "Monospace", "''": "Monospace", "'Menlo'": "Menlo",
	} {
		if got := monoFamily(in); got != want {
			t.Errorf("monoFamily(%q) = %q, want %q", in, got, want)
		}
	}
}
