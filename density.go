package gowt

import "math"

// Density chooses how the Type and Spacing tokens are sized, see App.SetDensity. The zero value
// keeps the fixed Spacing of earlier releases; docs/cross-platform-look.md compares the choices.
type Density struct {
	mode   densityMode
	factor float64
}

type densityMode uint8

const (
	modeLegacy densityMode = iota
	modeNative
	modeFactor
)

var (
	// DensityNative sizes the tokens for the OS: the body is 13 px on macOS, 14 px on Windows 11 and
	// the GTK UI font on Linux, and Spacing is the macOS value times body/13, in steps of 4 px.
	DensityNative = Density{mode: modeNative}
	// DensitySame is a 13 px body and the macOS Spacing on every OS.
	DensitySame = DensityFactor(1)
)

// DensityFactor is DensitySame scaled by f: a body of 13*f px and f times the macOS Spacing.
func DensityFactor(f float64) Density {
	if f <= 0 {
		f = 1
	}
	return Density{mode: modeFactor, factor: f}
}

// macBody is the macOS body size in px, the unit of a density factor of 1.
const macBody = 13

// snapStep is the step, in px, that a scaled spacing value is rounded to. At 125%, 150% and 200% a
// multiple of 4 is a whole device pixel.
const snapStep = 4

func snap(v float64) int {
	n := int(math.Round(v/snapStep)) * snapStep
	if v > 0 && n < snapStep {
		return snapStep
	}
	return n
}

// densityFactor is the factor Scaled and the Spacing tokens apply; 1 until SetDensity picks one.
var densityFactor = 1.0

// densityChosen is the Density the app chose; the zero Density until SetDensity.
var densityChosen Density

// Scaled returns n px multiplied by the density factor and rounded to a multiple of 4. It returns n
// unchanged without SetDensity. Use it for a width or height that is not a Spacing token.
func Scaled(n int) int {
	if densityChosen.mode == modeLegacy {
		return n
	}
	return snap(float64(n) * densityFactor)
}

// ChosenDensity reports what App.SetDensity set; the zero Density without it.
func ChosenDensity() Density { return densityChosen }

// Legacy reports whether d is the zero value, the behaviour of the releases before Density.
func (d Density) Legacy() bool { return d.mode == modeLegacy }

// TypeScale is the set of text styles, as Fonts in logical px for Typeface and SetFont.
type TypeScale struct {
	Caption, Body, BodyStrong, Subtitle, Title, LargeTitle Font
	// Mono is the platform's monospace font at the body size.
	Mono Font
}

// Type is the application-wide text scale: Typeface(gowt.Type.Title) gives a widget the title style,
// whose size and family are the OS's, see docs/cross-platform-look.md. Run fills it before setup;
// read it after SetDensity, a Typeface built earlier keeps the values of its time.
var Type TypeScale

// system is what the OS tells the token resolver; tests give it by hand.
type system struct {
	goos   string
	bodyPx float64 // the system UI font size in px; Windows gives 14, the WinUI body
	family string  // the UI family for Type, "" for the system font
	// display is the family for the large styles; "" uses family.
	display string
	mono    string
	// systemPx is the size of the font a widget has without a Typeface.
	systemPx float64
}

func (s system) differsFromSystem(body float64) bool {
	return s.family != "" || math.Abs(body-s.systemPx) > 0.05
}

// A ramp is the sizes of the other styles as multiples of the body.
type ramp struct{ caption, subtitle, title, largeTitle float64 }

var (
	macRamp   = ramp{11.0 / 13, 15.0 / 13, 22.0 / 13, 26.0 / 13}
	winRamp   = ramp{12.0 / 14, 20.0 / 14, 28.0 / 14, 40.0 / 14}
	adwRamp   = ramp{0.82, 1.2, 1.4, 1.8} // libadwaita: .caption, .title-4, .title-3, .title-2
	rampByOS  = map[string]ramp{"darwin": macRamp, "windows": winRamp, "linux": adwRamp}
	baseSpace = SpacingScale{XS: 4, SM: 8, MD: 12, LG: 16, XL: 32, PageMargin: 16,
		RowHeight: 32, ControlHeight: 28, SidebarRow: 28, SidebarWidth: 220, InspectorWidth: 260}
)

// resolve returns the tokens of d on sys, the body size in px that new widgets get, and the density
// factor. d must not be the zero Density.
func resolveTokens(sys system, d Density) (t TypeScale, s SpacingScale, body, factor float64) {
	r := rampByOS[sys.goos]
	switch d.mode {
	case modeFactor:
		body, factor, r = macBody*d.factor, d.factor, macRamp
	default:
		body = sys.bodyPx
		factor = body / macBody
	}
	font := func(name string, px float64, w Weight) Font { return Font{Name: name, Px: px, Weight: w} }
	large := sys.display
	if large == "" {
		large = sys.family
	}
	t = TypeScale{
		Caption:    font(sys.family, body*r.caption, Regular),
		Body:       font(sys.family, body, Regular),
		BodyStrong: font(sys.family, body, Semibold),
		Subtitle:   font(sys.family, body*r.subtitle, Semibold),
		Title:      font(large, body*r.title, Semibold),
		LargeTitle: font(large, body*r.largeTitle, Semibold),
		Mono:       font(sys.mono, body, Regular),
	}
	sc := func(n int) int { return snap(float64(n) * factor) }
	b := baseSpace
	s = SpacingScale{
		XS: sc(b.XS), SM: sc(b.SM), MD: sc(b.MD), LG: sc(b.LG), XL: sc(b.XL), PageMargin: sc(b.PageMargin),
		RowHeight: sc(b.RowHeight), ControlHeight: sc(b.ControlHeight), SidebarRow: sc(b.SidebarRow),
		SidebarWidth: sc(b.SidebarWidth), InspectorWidth: sc(b.InspectorWidth),
	}
	return t, s, body, factor
}

// SetDensity sets how the Type and Spacing tokens are sized, see Density. Call it first in the setup
// function of Run, before any widget is built and before tuning a token: it replaces Type and Spacing
// as a whole, and widgets already built keep their sizes. With DensityNative or a DensityFactor, new
// widgets also get the body font, which differs from the system font on Windows (Segoe UI Variable
// Text, 14 px) and with DensitySame on Windows and Linux (13 px). The zero Density restores the
// default.
func (a *App) SetDensity(d Density) {
	densityChosen = d
	sys := a.systemInfo()
	if d.mode == modeLegacy {
		densityFactor, a.bodyPx, a.body = 1, 0, nil
		Type, _, _, _ = resolveTokens(sys, DensityNative)
		Spacing = defaultSpacing()
		return
	}
	var body float64
	var s SpacingScale
	Type, s, body, densityFactor = resolveTokens(sys, d)
	Spacing = s
	a.bodyPx = body
	a.body = nil
	if sys.differsFromSystem(body) {
		b := Type.Body
		a.body = &b
	}
}
