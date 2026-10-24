// Package swt is the Go translation of Eclipse SWT (the widgets, graphics, events, layouts,
// dialogs and drag and drop of org.eclipse.swt), with the Java names. It talks to AppKit on
// macOS, Win32 on Windows and GTK 3 on Linux without cgo.
//
// New programs start with package gowt, which wraps this one in an idiomatic Go API; Unwrap
// on a gowt widget returns the swt object. Use swt directly to port SWT code or for what
// gowt does not wrap: https://github.com/haiodo/gowt/blob/main/docs/swt-direct.md.
//
// Almost every file is generated from the SWT sources by tooling/j2go and is not edited by
// hand; see https://github.com/haiodo/gowt/blob/main/docs/internals.md. The API follows
// SWT's, so the SWT documentation applies with Java syntax turned into Go: constructors are
// NewShell, NewButton and the like, methods keep their Java names (SetText, AddSelectionListener).
//
// All widget calls belong to the UI thread, the one that created the Display.
package swt
