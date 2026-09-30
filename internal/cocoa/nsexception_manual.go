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

// -[NSComboBox selectItemAtIndex:] and itemObjectValueAtIndex: throw NSRangeException for -1 (no
// selection) or a stale index; os.c turns that into a no-op / 0.
func (this *NSComboBox) SelectItemAtIndex(index int64) {
	if index < 0 || index >= this.NumberOfItems() {
		return
	}
	OSObjc_msgSendOverload44(this.Id, OSSel_selectItemAtIndex_, index)
}

func (this *NSComboBox) ItemObjectValueAtIndex(index int64) *id {
	if index < 0 || index >= this.NumberOfItems() {
		return nil
	}
	if result := OSObjc_msgSendOverload44(this.Id, OSSel_itemObjectValueAtIndex_, index); result != 0 {
		return NewidOverload1(result)
	}
	return nil
}
