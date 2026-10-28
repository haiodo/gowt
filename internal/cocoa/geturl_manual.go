//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

const (
	aeGetURL        = 'G'<<24 | 'U'<<16 | 'R'<<8 | 'L' // kInternetEventClass and kAEGetURL
	keyDirectObject = '-'<<24 | '-'<<16 | '-'<<8 | '-'
)

var getURL func(url string)

// HandleGetURL calls fn with the URL of each kAEGetURL Apple event (a CFBundleURLTypes link). AppKit's
// application:openURLs: would also take file opens from application:openFiles:, so a handler instead.
// It lives for the process; a later call only replaces fn.
func HandleGetURL(fn func(url string)) {
	first := getURL == nil
	getURL = fn
	if !first {
		return
	}
	cls := newClass("GowtGetURLHandler", nil, map[string]objcMethod{
		"handleGetURLEvent:withReplyEvent:": {purego.NewCallback(func(self, _, event, reply uintptr) {
			if u := goString(msg(msg(event, "paramDescriptorForKeyword:", keyDirectObject), "stringValue")); u != "" {
				getURL(u)
			}
		}), "v@:@@"},
	})
	h := msg(msg(cls, "alloc"), "init")
	msg(msg(class("NSAppleEventManager"), "sharedAppleEventManager"), "setEventHandler:andSelector:forEventClass:andEventID:",
		h, sel("handleGetURLEvent:withReplyEvent:"), aeGetURL, aeGetURL)
}
