//go:build darwin

// AppKit and Foundation throw on a nil NSString argument in the selectors below (measured on macOS 26:
// "nil argument", "Invalid parameter not satisfying: aString != nil"). Java null reaches them as a nil
// *NSString, e.g. DirectoryDialog.open's fileURLWithPath(null) when no filter path is set; os.c's @try
// hid that. A nil text goes out as the empty string; a nil path or extension gives "nothing": nil or
// false. Go "" is never nil: NSStringStringWith("") is a real empty NSString. Skipped in
// tooling/j2go/manual.txt, so the generated twins are gone.
package cocoa

// nonNilString is the id of s, or of an empty NSString for nil.
func nonNilString(s *NSString) int64 {
	if s != nil {
		return s.Id
	}
	return NSStringString().Id
}

func sendText(recv int64, sel int64, s *NSString) {
	OSObjc_msgSendOverload44(recv, sel, nonNilString(s))
}

func (this *NSMenu) SetTitle(title *NSString)         { sendText(this.Id, OSSel_setTitle_, title) }
func (this *NSMenuItem) SetTitle(title *NSString)     { sendText(this.Id, OSSel_setTitle_, title) }
func (this *NSBox) SetTitle(title *NSString)          { sendText(this.Id, OSSel_setTitle_, title) }
func (this *NSCell) SetTitle(title *NSString)         { sendText(this.Id, OSSel_setTitle_, title) }
func (this *NSWindow) SetTitle(title *NSString)       { sendText(this.Id, OSSel_setTitle_, title) }
func (this *NSControl) SetStringValue(s *NSString)    { sendText(this.Id, OSSel_setStringValue_, s) }
func (this *NSMenuItem) SetKeyEquivalent(k *NSString) { sendText(this.Id, OSSel_setKeyEquivalent_, k) }
func (this *NSButton) SetKeyEquivalent(k *NSString)   { sendText(this.Id, OSSel_setKeyEquivalent_, k) }
func (this *NSAlert) SetMessageText(s *NSString)      { sendText(this.Id, OSSel_setMessageText_, s) }
func (this *NSWindow) SetRepresentedFilename(s *NSString) {
	sendText(this.Id, OSSel_setRepresentedFilename_, s)
}
func (this *NSText) SetString(s *NSString)             { sendText(this.Id, OSSel_setString_, s) }
func (this *NSMutableString) AppendString(s *NSString) { sendText(this.Id, OSSel_appendString_, s) }
func (this *NSMutableString) SetString(s *NSString)    { sendText(this.Id, OSSel_setString_, s) }
func (this *NSSavePanel) SetNameFieldStringValue(s *NSString) {
	sendText(this.Id, OSSel_setNameFieldStringValue_, s)
}

func (this *NSText) ReplaceCharactersInRange(range_ NSRange, s *NSString) {
	OSObjc_msgSendOverload11(this.Id, OSSel_replaceCharactersInRange_withString_, range_, nonNilString(s))
}

func (this *NSMutableString) ReplaceCharactersInRange(range_ NSRange, s *NSString) {
	OSObjc_msgSendOverload11(this.Id, OSSel_replaceCharactersInRange_withString_, range_, nonNilString(s))
}

func (this *NSMutableAttributedString) ReplaceCharactersInRange(range_ NSRange, s *NSString) {
	OSObjc_msgSendOverload11(this.Id, OSSel_replaceCharactersInRange_withString_, range_, nonNilString(s))
}

func (this *NSString) StringByAppendingString(s *NSString) *NSString {
	return nsStringResult(this, OSObjc_msgSendOverload44(this.Id, OSSel_stringByAppendingString_, nonNilString(s)))
}

// A nil extension appends nothing, like an empty one.
func (this *NSString) StringByAppendingPathExtension(s *NSString) *NSString {
	return nsStringResult(this, OSObjc_msgSendOverload44(this.Id, OSSel_stringByAppendingPathExtension_, nonNilString(s)))
}

func (this *NSString) StringByReplacingOccurrencesOfString(target *NSString, replacement *NSString) *NSString {
	return nsStringResult(this, OSObjc_msgSendOverload54(this.Id, OSSel_stringByReplacingOccurrencesOfString_withString_, nonNilString(target), nonNilString(replacement)))
}

func nsStringResult(this *NSString, result int64) *NSString {
	if result == this.Id {
		return this
	}
	if result != 0 {
		return NewNSStringOverload1(result)
	}
	return nil
}

func (this *NSAttributedString) InitWithString(str *NSString) *NSAttributedString {
	return attributedResult(this, OSObjc_msgSendOverload44(this.Id, OSSel_initWithString_, nonNilString(str)))
}

func (this *NSAttributedString) InitWithStringStrAttrs(str *NSString, attrs *NSDictionary) *NSAttributedString {
	var attrsId int64
	if attrs != nil {
		attrsId = attrs.Id
	}
	return attributedResult(this, OSObjc_msgSendOverload54(this.Id, OSSel_initWithString_attributes_, nonNilString(str), attrsId))
}

func attributedResult(this *NSAttributedString, result int64) *NSAttributedString {
	if result == this.Id {
		return this
	}
	if result != 0 {
		return NewNSAttributedStringOverload1(result)
	}
	return nil
}

func (this *NSMenu) InitWithTitle(aTitle *NSString) *NSMenu {
	result := OSObjc_msgSendOverload44(this.Id, OSSel_initWithTitle_, nonNilString(aTitle))
	if result == this.Id {
		return this
	}
	if result != 0 {
		return NewNSMenuOverload1(result)
	}
	return nil
}

func (this *NSMenu) AddItemWithTitle(aString *NSString, aSelector int64, charCode *NSString) *NSMenuItem {
	result := OSObjc_msgSendOverload58(this.Id, OSSel_addItemWithTitle_action_keyEquivalent_, nonNilString(aString), aSelector, nonNilString(charCode))
	if result != 0 {
		return NewNSMenuItemOverload1(result)
	}
	return nil
}

func (this *NSMenuItem) InitWithTitle(aString *NSString, aSelector int64, charCode *NSString) *NSMenuItem {
	result := OSObjc_msgSendOverload58(this.Id, OSSel_initWithTitle_action_keyEquivalent_, nonNilString(aString), aSelector, nonNilString(charCode))
	if result == this.Id {
		return this
	}
	if result != 0 {
		return NewNSMenuItemOverload1(result)
	}
	return nil
}

// A nil path has no file URL: nil, which NSSavePanel.setDirectoryURL takes as "no directory".
func NSURLFileURLWithPath(path *NSString) *NSURL {
	if path == nil {
		return nil
	}
	if result := OSObjc_msgSendOverload44(OSClass_NSURL, OSSel_fileURLWithPath_, path.Id); result != 0 {
		return NewNSURLOverload1(result)
	}
	return nil
}

func UTTypeTypeWithFilenameExtension(filenameExtension *NSString) *UTType {
	if filenameExtension == nil {
		return nil
	}
	if result := OSObjc_msgSendOverload44(OSClass_UTType, OSSel_typeWithFilenameExtension_, filenameExtension.Id); result != 0 {
		return NewUTTypeOverload1(result)
	}
	return nil
}

func (this *NSWorkspace) IsFilePackageAtPath(fullPath *NSString) bool {
	if fullPath == nil {
		return false
	}
	return OSObjc_msgSend_boolOverload5(this.Id, OSSel_isFilePackageAtPath_, fullPath.Id)
}
