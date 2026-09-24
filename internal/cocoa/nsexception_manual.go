// SWT's os.c wraps every native call in @try/@catch and hands back 0 when AppKit throws, and a
// few SWT call sites rely on that. purego can't catch an NSException (it aborts the process), so
// each such selector is guarded here so that it never throws in the first place.
package cocoa

// -[NSColor colorSpace] throws for catalog/pattern colors (e.g. windowFrameTextColor);
// Display.getNSColorRGB expects null and then converts via colorUsingColorSpaceName:.
func (this *NSColor) ColorSpace() *NSColorSpace {
	const nsColorTypeComponentBased = 0
	if OSObjc_msgSend(this.Id, OSSel_registerName("type")) != nsColorTypeComponentBased {
		return nil
	}
	result := OSObjc_msgSend(this.Id, OSSel_colorSpace)
	if result == 0 {
		return nil
	}
	return NewNSColorSpaceOverload1(result)
}
