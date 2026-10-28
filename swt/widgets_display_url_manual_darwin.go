package swt

import "github.com/haiodo/gowt/internal/cocoa"

// hookGetURL turns a link to the app's URL scheme into SWT.OpenUrl. SWT leaves this to the Eclipse
// launcher, which calls the custom application:openUrls: selector; a gowt program has no launcher.
func (this *Display) hookGetURL() {
	cocoa.HandleGetURL(func(url string) {
		d := DisplayGetCurrent()
		if d == nil || d.IsDisposed() {
			return
		}
		event := NewEvent()
		event.Text = url
		d.SendEvent(OpenUrl, event)
	})
}
