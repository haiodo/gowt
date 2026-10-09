// Package look sets the platform look of a gowt app: the window material, macOS 26 Liquid Glass,
// Windows 11 corners and dark content. Every call does nothing on a system without the feature.
package look

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// Backdrop is the material behind a window's content.
type Backdrop int

const (
	BackdropNone        Backdrop = iota // opaque, the platform default
	BackdropTranslucent                 // blurred desktop behind the window: NSVisualEffectView, Acrylic
	BackdropGlass                       // Liquid Glass on macOS 26, Mica on Windows 11
)

// Container is a gowt container that has a composite: Panel, Window, Group, Split and CoolBar.
type Container interface{ AsComposite() *swt.Composite }

// SetBackdrop sets the window material. A value the OS cannot show falls back to the nearest
// one it can; Linux ignores it.
func SetBackdrop(w *gowt.Window, b Backdrop) { setBackdrop(w.Unwrap(), b) }

// SetRoundedCorners forces rounded (true) or square (false) window corners on Windows 11; other
// systems and older Windows builds ignore it. Windows 11 rounds top-level windows by default.
func SetRoundedCorners(w *gowt.Window, on bool) { setRoundedCorners(w.Unwrap(), on) }

// Classic asks macOS 26 and newer to keep the look of earlier releases instead of Liquid
// Glass (the UIDesignRequiresCompatibility key). Call it before gowt.Run. Other systems ignore it.
// Without an Info.plist the key can only be set at run time and macOS may not read it there;
// building with -ldflags=-macsdk=15.0 records an older SDK in the binary, which is reliable.
func Classic() { classic() }

// SetFullSizeContent lets the window content extend under the title bar, which becomes
// transparent, so the content starts at the top edge of the window: leave a top margin of about
// 28 points for the traffic lights. The client area and Pack follow the new mask. macOS only;
// elsewhere it does nothing.
func SetFullSizeContent(w *gowt.Window, on bool) { setFullSizeContent(w.Unwrap(), on) }

// SetGlass puts a Liquid Glass surface behind the container's content, with rounded corners; the
// container and its children stop painting their own background. It needs macOS 26; older macOS
// shows a blurred material, other systems nothing.
func SetGlass(c Container, on bool) { setGlass(c.AsComposite(), on) }

// GlassButton gives a push button (not Check, Radio, Toggle or Arrow) the Liquid Glass bezel on macOS 26 and newer; elsewhere the
// button is unchanged.
func GlassButton() gowt.Option { return gowt.Custom(glassButton) }

// SetDarkContent on Windows turns on the dark theme of the window content (scroll bars, tables, trees,
// buttons) through undocumented uxtheme exports. Call before creating windows; it does not switch live.
// GTK and AppKit theme the content themselves, so it does nothing there.
func SetDarkContent(a *gowt.App, on bool) { setDarkContent(a.Unwrap(), on) }
