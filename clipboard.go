package gowt

import "github.com/haiodo/gowt/swt"

// Clipboard is the system clipboard. App.Clipboard creates it; the App disposes it when Run
// returns, so there is nothing to dispose. UI thread only (from another goroutine use App.Sync).
//
// Getters return ok=false when the clipboard holds no such format. Setters return an error
// when the system clipboard cannot be written, which is routine on Windows while another
// process has it open. Setting one format clears the others; use SetHTML for HTML with a
// plain-text fallback. Other formats (RTF, URL, custom) and the X11 selection clipboard are
// reachable through Unwrap.
type Clipboard struct {
	app *App
	c   *swt.Clipboard
}

// Clipboard returns the system clipboard of the App, creating it on first use.
func (a *App) Clipboard() *Clipboard {
	if a.clip == nil {
		a.clip = &Clipboard{a, swt.NewClipboard(a.display)}
	}
	return a.clip
}

func (a *App) disposeClipboard() {
	if a.clip != nil && !a.clip.c.IsDisposed() {
		a.clip.c.Dispose()
	}
}

// Unwrap returns the underlying swt.Clipboard for the full API.
func (c *Clipboard) Unwrap() *swt.Clipboard { return c.c }

// Text returns the plain text on the clipboard.
func (c *Clipboard) Text() (string, bool) {
	s, ok := c.c.GetContents(swt.TextTransferGetInstance()).(string)
	return s, ok && s != ""
}

// SetText replaces the clipboard content with s.
func (c *Clipboard) SetText(s string) error {
	return c.set(s, swt.TextTransferGetInstance().AsTransfer())
}

// HTML returns the HTML fragment on the clipboard.
func (c *Clipboard) HTML() (string, bool) {
	s, ok := c.c.GetContents(swt.HTMLTransferGetInstance()).(string)
	return s, ok && s != ""
}

// SetHTML replaces the clipboard content with html and, for applications that do not read HTML, plain.
func (c *Clipboard) SetHTML(html, plain string) (err error) {
	defer catch(&err)
	c.c.SetContents([]any{html, plain}, []*swt.Transfer{
		swt.HTMLTransferGetInstance().AsTransfer(), swt.TextTransferGetInstance().AsTransfer()})
	return nil
}

// Files returns the paths of the files copied in a file manager, nil if there are none.
func (c *Clipboard) Files() []string {
	p, _ := c.c.GetContents(swt.FileTransferGetInstance()).([]string)
	return p
}

// SetFiles replaces the clipboard content with a list of file paths.
func (c *Clipboard) SetFiles(paths []string) error {
	return c.set(paths, swt.FileTransferGetInstance().AsTransfer())
}

// Image returns a copy of the image on the clipboard. The App owns it and disposes it when
// Run returns; Dispose may be called earlier.
func (c *Clipboard) Image() (img *Image, ok bool) {
	d, ok := c.c.GetContents(swt.ImageTransferGetInstance()).(*swt.ImageData)
	if !ok || d == nil {
		return nil, false
	}
	return c.app.track(swt.NewImageDeviceData(c.app.display, d)), true
}

// SetImage replaces the clipboard content with a copy of img.
func (c *Clipboard) SetImage(img *Image) error {
	return c.set(img.i.GetImageData(), swt.ImageTransferGetInstance().AsTransfer())
}

// Clear empties the clipboard.
func (c *Clipboard) Clear() { c.c.ClearContents() }

func (c *Clipboard) set(data any, t *swt.Transfer) (err error) {
	defer catch(&err)
	c.c.SetContents([]any{data}, []*swt.Transfer{t})
	return nil
}
