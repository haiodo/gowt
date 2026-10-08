package gowt

// ClassicLook asks macOS 26 and newer to keep the look of earlier releases instead of Liquid
// Glass (the UIDesignRequiresCompatibility key). Call it before Run. Other systems ignore it.
// Without an Info.plist the key can only be set at run time and macOS may not read it there;
// building with -ldflags=-macsdk=15.0 records an older SDK in the binary, which is reliable.
func ClassicLook() { classicLook() }

// SetFullSizeContent lets the window content extend under the title bar, which becomes
// transparent, so the content starts at the top edge of the window: leave a top margin of about
// 28 points for the traffic lights. The client area and Pack follow the new mask. macOS only;
// elsewhere it does nothing.
func (w *Window) SetFullSizeContent(on bool) { w.setFullSizeContent(on) }

// SetGlass puts a Liquid Glass surface behind the panel's content, with rounded corners; the
// panel and its children stop painting their own background (works on Group too). It needs macOS 26; older macOS shows a blurred material, other systems nothing.
func (p *panel) SetGlass(on bool) { p.setGlass(on) }

// GlassButton gives a push button (not Check, Radio, Toggle or Arrow) the Liquid Glass bezel on macOS 26 and newer; elsewhere the
// button is unchanged.
func GlassButton() Option { return glassButton() }
