//go:build darwin

package cocoa

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func newObj(class_ string) int64 { return int64(msg(msg(class(class_), "alloc"), "init")) }

// A thrown NSException aborts the process, so a failing guard shows as a crashed test binary.
func TestNilStringArgumentsDoNotThrow(t *testing.T) {
	if s := NSStringStringWith(""); s == nil || s.Id == 0 {
		t.Fatal(`NSStringStringWith("") must be a real empty NSString, not nil`)
	}
	if u := NSURLFileURLWithPath(nil); u != nil {
		t.Fatal("NSURLFileURLWithPath(nil) must give nil")
	}
	NSURLFileURLWithPath(NSStringStringWith("")) // a URL or nil, but no throw
	if UTTypeTypeWithFilenameExtension(nil) != nil {
		t.Fatal("UTTypeTypeWithFilenameExtension(nil) must give nil")
	}
	if NSWorkspaceSharedWorkspace().IsFilePackageAtPath(nil) {
		t.Fatal("IsFilePackageAtPath(nil) must be false")
	}
	if s := NSStringStringWith("a").StringByAppendingString(nil); s.Id == 0 {
		t.Fatal("StringByAppendingString(nil) lost the receiver")
	}
	NSStringStringWith("a").StringByAppendingPathExtension(nil)
	NSStringStringWith("a").StringByReplacingOccurrencesOfString(nil, nil)

	ms := NewNSMutableStringOverload1(newObj("NSMutableString"))
	ms.AppendString(nil)
	ms.SetString(nil)
	ms.ReplaceCharactersInRange(NSRange{}, nil)
	NewNSMutableAttributedStringOverload1(int64(msg(msg(class("NSMutableAttributedString"), "alloc"), "initWithString:", uintptr(NSStringStringWith("a").Id)))).ReplaceCharactersInRange(NSRange{}, nil)
	NewNSAttributedStringOverload1(int64(msg(class("NSAttributedString"), "alloc"))).InitWithString(nil)
	NewNSAttributedStringOverload1(int64(msg(class("NSAttributedString"), "alloc"))).InitWithStringStrAttrs(nil, nil)

	NewNSMenuOverload1(int64(msg(class("NSMenu"), "alloc"))).InitWithTitle(nil)
	menu := NewNSMenuOverload1(newObj("NSMenu"))
	menu.SetTitle(nil)
	menu.AddItemWithTitle(nil, 0, nil)
	item := NewNSMenuItemOverload1(int64(msg(class("NSMenuItem"), "alloc"))).InitWithTitle(nil, 0, nil)
	item.SetTitle(nil)
	item.SetKeyEquivalent(nil)
	NewNSBoxOverload1(newObj("NSBox")).SetTitle(nil)
	NewNSCellOverload1(newObj("NSCell")).SetTitle(nil)
	NewNSButtonOverload1(newObj("NSButton")).SetKeyEquivalent(nil)
	NewNSTextFieldOverload1(newObj("NSTextField")).SetStringValue(nil)
}

// NSWindow, NSTextView, NSAlert and NSSavePanel only exist on the main thread: the child runs these in init, which
// Go runs there. None is shown.
func init() {
	if os.Getenv("GOWT_COCOA_NIL_CHILD") != "1" {
		return
	}
	runtime.LockOSThread()
	msg(class("NSApplication"), "sharedApplication")
	w := NewNSWindowOverload1(newObj("NSWindow"))
	w.SetTitle(nil)
	w.SetRepresentedFilename(nil)
	tv := NewNSTextViewOverload1(newObj("NSTextView"))
	tv.SetString(nil)
	tv.ReplaceCharactersInRange(NSRange{}, nil)
	NewNSAlertOverload1(newObj("NSAlert")).SetMessageText(nil)
	sp := NSSavePanelSavePanel()
	sp.SetNameFieldStringValue(nil)
	sp.SetDirectoryURL(NSURLFileURLWithPath(nil))
	os.Exit(0)
}

func TestNilStringArgumentsOnMainThread(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "GOWT_COCOA_NIL_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
