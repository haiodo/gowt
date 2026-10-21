//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

// VoiceOverEnabled reports whether VoiceOver is running (NSWorkspace.isVoiceOverEnabled).
func VoiceOverEnabled() bool {
	return msg(msg(class("NSWorkspace"), "sharedWorkspace"), "isVoiceOverEnabled")&0xff != 0
}

var voiceOverChanged func()

// ObserveVoiceOver calls fn on the UI thread after VoiceOver was turned on or off (KVO on
// NSWorkspace.voiceOverEnabled). The observer lives for the process; a later call only replaces fn.
func ObserveVoiceOver(fn func()) {
	first := voiceOverChanged == nil
	voiceOverChanged = fn
	if !first {
		return
	}
	cls := newClass("GowtVoiceOverObserver", nil, map[string]objcMethod{
		"observeValueForKeyPath:ofObject:change:context:": {purego.NewCallback(func(self, _, keyPath, obj, change, ctx uintptr) {
			voiceOverChanged()
		}), "v@:@@@^v"},
	})
	obs := msg(msg(cls, "alloc"), "init")
	msg(msg(class("NSWorkspace"), "sharedWorkspace"), "addObserver:forKeyPath:options:context:", obs, nsString("voiceOverEnabled"), 0, 0)
}
