package swt

import "github.com/haiodo/gowt/internal/gtk"

// The file name sorts last: this init() runs after the generated ones, which call GTK at import.
// Without GTK those calls get zero values; calls after this one panic with the load error.
func init() { gtk.MarkInitDone() }
