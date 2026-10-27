package look

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

const (
	dwmaWindowCornerPreference         = 33
	dwmaSystemBackdropType             = 38
	dwmbbNone, dwmbbMica, dwmbbAcrylic = 1, 2, 3
	dwmcpDoNotRound, dwmcpRound        = 1, 2

	buildWin11      = 22000
	buildWin11_22H2 = 22621
)

var (
	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrame        = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procRtlGetVersion         = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
)

func windowsBuild() uint32 { return uint32(win32.OsVersionWIN32_BUILD) }

func dwmSet(h int64, attr, value int32) {
	procDwmSetWindowAttribute.Call(uintptr(h), uintptr(attr), uintptr(unsafe.Pointer(&value)), 4)
}

// backdropOnKey marks a shell whose background was set for the material.
const backdropOnKey = "gowt.look.backdropOn"

// setBackdrop needs Windows 11 22H2 (build 22621); older builds keep the opaque window.
// Translucent is Acrylic, Glass is Mica. The material shows only where the client area is painted black
// (DWM treats RGB 0,0,0 over an extended frame as transparent), so the shell background becomes black:
// widgets with their own opaque background are unaffected, but text drawn in pure black turns transparent -
// use a near-black foreground on the material.
func setBackdrop(s *swt.Shell, b Backdrop) {
	if windowsBuild() < buildWin11_22H2 {
		return
	}
	h := s.Handle
	kind, margin := int32(dwmbbNone), int32(0)
	switch b {
	case BackdropTranslucent:
		kind, margin = dwmbbAcrylic, -1
	case BackdropGlass:
		kind, margin = dwmbbMica, -1
	}
	dwmSet(h, dwmaSystemBackdropType, kind)
	m := [4]int32{margin, margin, margin, margin}
	procDwmExtendFrame.Call(uintptr(h), uintptr(unsafe.Pointer(&m)))
	if margin == 0 {
		if on, _ := s.GetDataKey(backdropOnKey).(bool); on {
			s.SetDataKeyValue(backdropOnKey, false)
			fluent.Background(&s.Composite)
		}
		return
	}
	s.SetDataKeyValue(backdropOnKey, true)
	s.SetBackgroundWithColor(swt.NewColorRedGreenBlue(0, 0, 0))
	s.SetBackgroundMode(swt.INHERIT_DEFAULT)
}

func topInset(*swt.Shell) int { return 0 }

func setRoundedCorners(s *swt.Shell, on bool) {
	if windowsBuild() < buildWin11 {
		return
	}
	pref := int32(dwmcpDoNotRound)
	if on {
		pref = dwmcpRound
	}
	dwmSet(s.Handle, dwmaWindowCornerPreference, pref)
}

func setDarkContent(d *swt.Display, on bool) {
	d.SetData("org.eclipse.swt.internal.win32.useDarkModeExplorerTheme", on)
}

func classic() {}

func setFullSizeContent(*swt.Shell, bool) {}

func setGlass(c *swt.Composite, on bool) { fluent.Glass(c, on) }

func setGlassRadius(c *swt.Composite, r float64) { fluent.GlassRadius(c, r) }

// setMaterial gives c the window background the sidebar and inspector panels inherit from the shell
// (fluentShell), also where the app coloured the content around c.
func setMaterial(c *swt.Composite, on bool) {
	if on {
		fluent.Background(c)
		return
	}
	c.SetBackgroundWithColor(nil)
	c.SetBackgroundMode(swt.INHERIT_NONE)
}

func glassButton(c *swt.Control) {
	if b := fluent.ButtonOf(c); b != nil {
		fluent.Button(b, false, false, false)
	}
}

func setGlassBorder(c *swt.Control, on bool) { fluent.GlassBorder(c, on) }

func setGlassFocusRing(c *swt.Control, on bool) { fluent.GlassFocusRing(c, on) }

func setToolbar(*swt.Shell, []ToolbarItem) bool { return false }

func setGlassGroup(*swt.Composite, float64) {}

func sidebarInset() int32 { return 0 }

// sidebarPadding is the side margin of the sidebar children, in points: the pill of a row starts 8 from
// the pane edge, as the item margin of a WinUI NavigationView does.
func sidebarPadding() int32 { return 8 }

// setSidebarChrome gives the Tables, Trees and Lists of the sidebar the pill and the insets of a sidebar
// row. A list added to the sidebar after SetSidebar keeps the look of a content list.
func setSidebarChrome(_ *swt.Shell, sb *swt.Composite, _ *swt.Control) {
	row := 0
	if !gowt.ChosenDensity().Legacy() {
		row = gowt.Spacing.SidebarRow
	}
	eachList(sb, func(c any) {
		switch v := c.(type) {
		case *swt.Table:
			fluent.SidebarTable(v, row)
		case *swt.Tree:
			fluent.SidebarTree(v, row)
		case *swt.List:
			fluent.SidebarList(v)
		}
	})
}

func syncSidebar(*swt.Shell, *swt.Composite, [4]int32, bool) {}

func inspectorInset() int32 { return 0 }

func syncInspector(*swt.Composite, [4]int32) {}

// sidebarDivider leaves a gap for the sash: an SWT sash here paints its own background and would cover
// the content edge.
func sidebarDivider() int32 { return 4 }

// itemsMinEdge has no native toolbar to measure; the side row of Sidebar items is measured instead.
func itemsMinEdge(*swt.Shell) (int32, bool) { return 0, false }

// startUnfocused does nothing: SWT keeps its choice, the first control.
func startUnfocused(*swt.Shell) {}
