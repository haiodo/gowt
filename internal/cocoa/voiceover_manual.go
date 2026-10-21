//go:build darwin

package cocoa

// VoiceOverEnabled reports whether VoiceOver is running (NSWorkspace.isVoiceOverEnabled).
func VoiceOverEnabled() bool {
	return msg(msg(class("NSWorkspace"), "sharedWorkspace"), "isVoiceOverEnabled")&0xff != 0
}
