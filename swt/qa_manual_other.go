//go:build !darwin

package swt

// QAFocus is SetFocus: only macOS needs the focus of an app that is not active (see qa_manual_on_darwin.go).
func QAFocus(c *Control) bool { return c.SetFocus() }
