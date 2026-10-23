package webview

// SetPageDialog replaces the sheet a page's own alert, confirm or prompt would show.
func SetPageDialog(f func(kind int32, msg, def string) (bool, string)) { pageDialog = f }
